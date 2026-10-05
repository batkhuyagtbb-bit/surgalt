package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"surgalt/internal/oauth"
	"surgalt/internal/store"
)

const oauthCookie = "surgalt_oauth"

type oauthState struct {
	State    string `json:"s"`
	Verifier string `json:"v,omitempty"`
	Role     string `json:"r"`
	Next     string `json:"n"`
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	out := []map[string]string{}
	if s.OAuth != nil {
		for _, p := range s.OAuth.List() {
			out = append(out, map[string]string{"name": p.Name, "label": p.Label, "url": "/auth/" + p.Name + "/start"})
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, out)
}

func safeNext(n string) string {
	if !strings.HasPrefix(n, "/") || strings.HasPrefix(n, "//") || strings.HasPrefix(n, "/\\") {
		return "/me"
	}
	return n
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) provider(w http.ResponseWriter, r *http.Request) (*oauth.Provider, bool) {
	if s.OAuth != nil {
		if p, ok := s.OAuth.Get(r.PathValue("provider")); ok {
			return p, true
		}
	}
	http.Redirect(w, r, "/login#error="+url.QueryEscape("Энэ нэвтрэх арга идэвхгүй байна"), http.StatusFound)
	return nil, false
}

// handleOAuthStart: /auth/{provider}/start?role=teacher&next=/me
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	p, ok := s.provider(w, r)
	if !ok {
		return
	}
	st := oauthState{State: randHex(16), Role: string(store.RoleStudent), Next: safeNext(r.URL.Query().Get("next"))}
	if r.URL.Query().Get("role") == string(store.RoleTeacher) {
		st.Role = string(store.RoleTeacher)
	}
	opts := []oauth2.AuthCodeOption{}
	if p.PKCE {
		st.Verifier = oauth2.GenerateVerifier()
		opts = append(opts, oauth2.S256ChallengeOption(st.Verifier))
	}
	sealed, err := s.tokens.Seal(st, 10*time.Minute)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthCookie, Value: sealed, Path: "/auth/", MaxAge: 600, HttpOnly: true,
		Secure: strings.HasPrefix(s.baseURL(r), "https://"), SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, p.Config.AuthCodeURL(st.State, opts...), http.StatusFound)
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(msg string) {
		http.Redirect(w, r, "/login#error="+url.QueryEscape(msg), http.StatusFound)
	}
	p, ok := s.provider(w, r)
	if !ok {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthCookie, Value: "", Path: "/auth/", MaxAge: -1, HttpOnly: true})
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		fail("Нэвтрэлт цуцлагдлаа")
		return
	}
	var st oauthState
	ck, err := r.Cookie(oauthCookie)
	if err != nil || s.tokens.Open(ck.Value, &st) != nil || !constantEq(st.State, q.Get("state")) {
		fail("Нэвтрэх хугацаа дууссан, дахин оролдоно уу")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var opts []oauth2.AuthCodeOption
	if st.Verifier != "" {
		opts = append(opts, oauth2.VerifierOption(st.Verifier))
	}
	tok, err := p.Config.Exchange(ctx, q.Get("code"), opts...)
	if err != nil {
		s.log.Warn("oauth exchange", "provider", p.Name, "err", err)
		fail(p.Label + "-тай холбогдож чадсангүй")
		return
	}
	id, err := p.FetchIdentity(ctx, tok)
	if err != nil {
		s.log.Warn("oauth userinfo", "provider", p.Name, "err", err)
		fail(p.Label + "-аас мэдээлэл авч чадсангүй")
		return
	}
	u, err := s.loginWithIdentity(ctx, id, store.Role(st.Role))
	if err != nil {
		s.log.Error("oauth login", "provider", p.Name, "err", err)
		fail("Бүртгэл үүсгэж чадсангүй")
		return
	}
	jwt, err := s.tokens.SignUser(u.ID, string(u.Role), u.Username)
	if err != nil {
		fail("Токен үүсгэж чадсангүй")
		return
	}
	// Токеныг URL fragment-ээр дамжуулна: сервер лог, Referer-т үлдэхгүй.
	http.Redirect(w, r, "/login#token="+url.QueryEscape(jwt)+"&next="+url.QueryEscape(st.Next), http.StatusFound)
}

var nonUser = regexp.MustCompile(`[^a-z0-9_]+`)

func usernameBase(id oauth.Identity) string {
	for _, c := range []string{id.Username, strings.SplitN(id.Email, "@", 2)[0], id.Name} {
		b := strings.Trim(nonUser.ReplaceAllString(strings.ToLower(c), "_"), "_")
		if len(b) > 20 {
			b = b[:20]
		}
		if len(b) >= 3 {
			return b
		}
	}
	return "user"
}

// loginWithIdentity: холбогдсон данс → баталгаажсан имэйлээр холбох → шинэ бүртгэл.
func (s *Server) loginWithIdentity(ctx context.Context, id oauth.Identity, role store.Role) (*store.User, error) {
	if u, err := s.store.UserByIdentity(ctx, id.Provider, id.Subject); err == nil {
		return u, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	if id.Email != "" && id.EmailVerified {
		if u, err := s.store.UserByEmail(ctx, id.Email); err == nil {
			if err := s.store.LinkIdentity(ctx, u.ID, id.Provider, id.Subject); err != nil {
				return nil, err
			}
			return u, nil
		} else if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	name := strings.TrimSpace(id.Name)
	if name == "" {
		name = usernameBase(id)
	}
	if r := []rune(name); len(r) > 80 {
		name = string(r[:80])
	}
	email := id.Email
	if email == "" || !id.EmailVerified {
		// Баталгаажаагүй имэйлийг бусдын бүртгэлтэй мөргөлдүүлэхгүйн тулд орлуулна.
		email = id.Provider + "_" + id.Subject + "@users.noreply.surgalt"
	}
	avatar := ""
	if validURL(id.AvatarURL) {
		avatar = id.AvatarURL
	}
	base := usernameBase(id)
	var u *store.User
	for attempt := 0; attempt < 6; attempt++ {
		uname := base
		if attempt > 0 {
			uname = base + "_" + randHex(2)
		}
		u = &store.User{Username: uname, Email: email, Role: role, DisplayName: name, AvatarURL: avatar}
		err := s.store.CreateUser(ctx, u)
		if err == nil {
			break
		}
		if !errors.Is(err, store.ErrConflict) {
			return nil, err
		}
		if attempt == 5 {
			return nil, err
		}
	}
	s.onUserCreated(u)
	if err := s.store.LinkIdentity(ctx, u.ID, id.Provider, id.Subject); err != nil {
		// Зэрэг хоёр callback: нөгөөх нь түрүүлж холбосон бол түүнийг ашиглана.
		if existing, e2 := s.store.UserByIdentity(ctx, id.Provider, id.Subject); e2 == nil {
			return existing, nil
		}
		return nil, err
	}
	return u, nil
}
