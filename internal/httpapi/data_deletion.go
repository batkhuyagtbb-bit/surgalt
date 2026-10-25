package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"surgalt/internal/store"
)

// Facebook-ийн «өгөгдөл устгах хүсэлт» (Data Deletion Request Callback): хэрэглэгч Facebook-ийн тохиргооноос
// surgalt.mn-ийг хасаж өгөгдлөө устгуулахыг хүсэхэд Facebook энд signed_request илгээнэ. Facebook-ээс авсан
// холболт, зургийг устгана; бүртгэл зөвхөн Facebook-ээр нэвтэрдэг (нууц үг, өөр нэвтрэлтгүй) бол бүртгэлийг
// бүхэлд нь устгана. Хариуд нь төлөв харах холбоос, баталгаажуулах код өгнө (/data-deletion?code=…).

const fbDelPurpose = "fb-deletion"

// Хүсэлтийн төлөв (баталгаажуулах кодын эхний үсэг).
const (
	fbDelErased   = 'd' // бүртгэл бүхэлдээ устсан
	fbDelUnlinked = 'u' // Facebook холболт, зураг устсан; бүртгэл нууц үг/өөр нэвтрэлттэй тул үлдсэн
	fbDelPartial  = 'p' // Facebook холболт устсан; сургалт, номтой багшийн бүртгэлийг гараар шийднэ
	fbDelNone     = 'n' // энэ Facebook данстай бүртгэл байхгүй
)

var errSignedRequest = errors.New("signed_request буруу")

// fbSignedRequest — "<sig>.<payload>" (base64url), sig = HMAC-SHA256(payload, апп-ын нууц түлхүүр). Facebook user_id буцаана.
func fbSignedRequest(sr, secret string) (string, error) {
	sig, payload, ok := strings.Cut(sr, ".")
	if !ok || secret == "" {
		return "", errSignedRequest
	}
	got, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(sig, "="))
	if err != nil {
		return "", errSignedRequest
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return "", errSignedRequest
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(payload, "="))
	if err != nil {
		return "", errSignedRequest
	}
	var v struct {
		Algorithm string `json:"algorithm"`
		UserID    string `json:"user_id"`
	}
	if json.Unmarshal(raw, &v) != nil || !strings.EqualFold(v.Algorithm, "HMAC-SHA256") || v.UserID == "" {
		return "", errSignedRequest
	}
	return v.UserID, nil
}

// fbDelCode — DB-гүй баталгаажуулах код: <төлөв><unix, 36-тын> "-" <гарын үсэг>.
func (s *Server) fbDelCode(st byte, at time.Time) string {
	p := string(st) + strconv.FormatInt(at.Unix(), 36)
	return p + "-" + s.tokens.MAC(fbDelPurpose, p)[:12]
}

func (s *Server) fbDelStatus(code string) (st byte, at time.Time, ok bool) {
	p, sig, found := strings.Cut(code, "-")
	if !found || len(p) < 2 || len(code) > 40 || !hmac.Equal([]byte(sig), []byte(s.tokens.MAC(fbDelPurpose, p)[:12])) {
		return 0, at, false
	}
	n, err := strconv.ParseInt(p[1:], 36, 64)
	if err != nil {
		return 0, at, false
	}
	return p[0], time.Unix(n, 0), true
}

// POST /auth/facebook/data-deletion (form: signed_request) — Facebook-ийн серверээс.
func (s *Server) handleFBDataDeletion(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт")
		return
	}
	secret := ""
	if s.OAuth != nil {
		if p, ok := s.OAuth.Get("facebook"); ok {
			secret = p.Config.ClientSecret
		}
	}
	if secret == "" {
		writeErr(w, http.StatusNotFound, "Facebook нэвтрэлт идэвхгүй")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	fbID, err := fbSignedRequest(r.PostFormValue("signed_request"), secret)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, st, uid := r.Context(), byte(fbDelNone), ""
	u, err := s.store.UserByIdentity(ctx, "facebook", fbID)
	switch {
	case errors.Is(err, store.ErrNotFound):
	case s.storeErr(w, r, err):
		return
	default:
		uid = u.ID
		if st, err = s.fbErase(ctx, u, fbID); s.storeErr(w, r, err) {
			return
		}
	}
	code := s.fbDelCode(st, time.Now())
	s.log.Info("facebook: өгөгдөл устгах хүсэлт", "status", string(st), "user", uid)
	writeJSON(w, http.StatusOK, map[string]string{"url": s.cfg.PublicURL + "/data-deletion?code=" + code, "confirmation_code": code})
}

// fbErase — Facebook холболт ба Facebook-ээс авсан зургийг устгана; бүртгэлд өөр нэвтрэх арга байхгүй бол бүртгэлийг бүхэлд нь.
func (s *Server) fbErase(ctx context.Context, u *store.User, fbID string) (byte, error) {
	if err := s.store.UnlinkIdentity(ctx, "facebook", fbID); err != nil {
		return 0, err
	}
	others, err := s.store.IdentityProviders(ctx, u.ID)
	if err != nil {
		return 0, err
	}
	others = slices.DeleteFunc(others, func(p string) bool { return p == "facebook" })
	st := byte(fbDelUnlinked)
	if u.PasswordHash == "" && len(others) == 0 {
		blocked, err := s.eraseAccount(ctx, u)
		if err != nil {
			return 0, err
		}
		if blocked == "" {
			return fbDelErased, nil
		}
		s.log.Warn("facebook: бүртгэлийг автоматаар устгаагүй", "user", u.ID, "reason", blocked)
		st = fbDelPartial
	}
	if fbAvatar(u.AvatarURL) {
		if _, err := s.store.UpdateUser(ctx, u.ID, func(x *store.User) error { x.AvatarURL = ""; return nil }); err != nil {
			return 0, err
		}
		s.profiles.Delete(u.Username)
	}
	return st, nil
}

// fbAvatar — Facebook-ийн CDN дээрх профайл зураг (нэвтрэхэд Facebook-ээс авсан).
func fbAvatar(u string) bool {
	for _, h := range []string{"fbcdn.net", "fbsbx.com", "facebook.com"} {
		if strings.Contains(u, h) {
			return true
		}
	}
	return false
}

// GET /data-deletion[?code=…] — Facebook-ээр нэвтэрсэн мэдээллээ устгуулах заавар ба хүсэлтийн төлөв.
func (s *Server) pageDataDeletion(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{"PublicURL": s.cfg.PublicURL, "Contact": s.cfg.ContactEmail}
	if code := strings.TrimSpace(r.URL.Query().Get("code")); code != "" {
		data["Code"], data["Status"] = code, ""
		if st, at, ok := s.fbDelStatus(code); ok {
			data["Status"], data["At"] = string(st), at.In(mnLoc).Format("2006-01-02 15:04")
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "data-deletion.html", data); err != nil {
		s.log.Error("template", "err", err, "page", "data-deletion")
	}
}
