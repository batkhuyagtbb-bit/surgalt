package httpapi

import (
	"errors"
	"math"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/auth"
	"surgalt/internal/files"
	"surgalt/internal/oauth"
	"surgalt/internal/store"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

// minPassword — нууц үгийн доод урт (login.html-ийн minlength-тэй ижил байх ёстой).
const minPassword = 6

type authResp struct {
	Token string `json:"token"`
	User  meView `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		Role        string `json:"role"` // хуучин клиентүүд илгээж болно — үл тоомсорлоно (бүртгэл бүр багш)
		DisplayName string `json:"display_name"`
		AcceptTerms bool   `json:"accept_terms"` // нөхцөл, нууцлалын бодлогыг зөвшөөрсөн (заавал)
	}
	if !decode(w, r, &in) {
		return
	}
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	// Хэрэглэгчийн нэр заавал биш: хоосон бол имэйлээс автоматаар үүсгэнэ (бүртгэлийн маягт зөвхөн имэйл, нууц үг асууна).
	auto := in.Username == ""
	switch {
	case !auto && !usernameRe.MatchString(in.Username):
		writeErr(w, http.StatusBadRequest, "username: 3-32 тэмдэгт, a-z 0-9 _")
		return
	case func() bool { _, err := mail.ParseAddress(in.Email); return err != nil }():
		writeErr(w, http.StatusBadRequest, "имэйл буруу")
		return
	case len(in.Password) < minPassword || len(in.Password) > 72:
		writeErr(w, http.StatusBadRequest, "нууц үг 6-72 тэмдэгт")
		return
	case !in.AcceptTerms:
		writeErr(w, http.StatusBadRequest, "үйлчилгээний нөхцөл, нууцлалын бодлогыг зөвшөөрнө үү")
		return
	}
	// Төрөл сонгуулахгүй: бүртгэл бүр багш (суралцах эрх нь хэвээр).
	role := store.RoleTeacher
	base := in.Username
	if auto {
		base = usernameBase(oauth.Identity{Email: in.Email})
	}
	if in.DisplayName = strings.TrimSpace(in.DisplayName); in.DisplayName == "" {
		in.DisplayName = base
	}
	if utf8.RuneCountInString(in.DisplayName) > 80 {
		writeErr(w, http.StatusBadRequest, "нэр хэт урт")
		return
	}
	hash, err := auth.HashPassword(r.Context(), in.Password)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "түр ачаалалтай байна")
		return
	}
	if auto {
		// Имэйл давхардсаныг нэрийн мөргөлдөөнөөс ялгаж, ойлгомжтой алдаа өгнө.
		if _, err := s.store.UserByEmail(r.Context(), in.Email); err == nil {
			writeErr(w, http.StatusConflict, "энэ имэйл бүртгэлтэй байна — нэвтэрнэ үү")
			return
		} else if !errors.Is(err, store.ErrNotFound) {
			s.storeErr(w, r, err)
			return
		}
	}
	var u *store.User
	for attempt := 0; ; attempt++ {
		uname := base
		if attempt > 0 {
			uname = base + "_" + randHex(2) // нэр давхардвал богино дагавар нэмнэ
		}
		u = &store.User{Username: uname, Email: in.Email, PasswordHash: hash, Role: role, DisplayName: in.DisplayName}
		consentNow(u)
		err := s.store.CreateUser(r.Context(), u)
		if err == nil {
			break
		}
		if !auto || attempt == 5 || !errors.Is(err, store.ErrConflict) {
			s.storeErr(w, r, err)
			return
		}
	}
	s.onUserCreated(u)
	s.issue(w, http.StatusCreated, u)
}

func (s *Server) issue(w http.ResponseWriter, status int, u *store.User) {
	tok, err := s.tokens.SignUser(u.ID, string(u.Role), u.Username)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "токен үүсгэж чадсангүй")
		return
	}
	writeJSON(w, status, authResp{Token: tok, User: s.me(u)})
}

// asTeacher: нэвтэрсэн хүн бүр багшийн эрхээр орно — хуучин «суралцагч» бүртгэлийг нэвтрэх үед нь багш болгоно
// (сурсан хичээл, худалдан авалт хэвээр; студи, файлын сан нэмэгдэнэ). Алдаа гарвал хуучин төрлөөр нь нэвтрүүлнэ.
func (s *Server) asTeacher(r *http.Request, u *store.User) *store.User {
	if u.Role != store.RoleStudent {
		return u
	}
	now := time.Now()
	nu, err := s.store.UpdateUser(r.Context(), u.ID, func(x *store.User) error {
		x.Role = store.RoleTeacher
		if x.Plan.TrialFrom == nil { // үнэгүй хугацаа багш болсон өдрөөс (admin_role.go-тэй ижил)
			x.Plan.TrialFrom = &now
		}
		return nil
	})
	if err != nil {
		s.log.Error("суралцагчийг багш болгох", "err", err, "user", u.ID)
		return u
	}
	s.onUserCreated(nu)
	return nu
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	u, err := s.store.UserByEmail(r.Context(), strings.TrimSpace(in.Email))
	hash := ""
	if err == nil {
		hash = u.PasswordHash
	} else if err != store.ErrNotFound {
		s.storeErr(w, r, err)
		return
	}
	ok, err := auth.CheckPassword(r.Context(), hash, in.Password)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "түр ачаалалтай байна")
		return
	}
	if !ok {
		writeErr(w, http.StatusUnauthorized, "имэйл эсвэл нууц үг буруу")
		return
	}
	s.issue(w, http.StatusOK, s.asTeacher(r, u))
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	v := s.me(u)
	v.Orgs = s.managedOrgs(r.Context(), u.ID) // удирддаг байгууллагууд (дээд цэсэнд)
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		DisplayName string            `json:"display_name"`
		Headline    string            `json:"headline"`
		Bio         string            `json:"bio"`
		AvatarURL   string            `json:"avatar_url"`
		CoverURL    string            `json:"cover_url"`
		Subjects    []string          `json:"subjects"`
		Location    string            `json:"location"`
		Links       map[string]string `json:"links"`
		Phone       *string           `json:"phone"` // ирээгүй бол хэвээр (зураг солих зэрэг хэсэгчилсэн хадгалалт)
	}
	if !decode(w, r, &in) {
		return
	}
	in.DisplayName, in.Headline = strings.TrimSpace(in.DisplayName), strings.TrimSpace(in.Headline)
	in.Location = strings.Join(strings.Fields(in.Location), " ")
	switch {
	case in.DisplayName == "" || utf8.RuneCountInString(in.DisplayName) > 80:
		writeErr(w, http.StatusBadRequest, "нэр 1-80 тэмдэгт")
		return
	case utf8.RuneCountInString(in.Headline) > 120:
		writeErr(w, http.StatusBadRequest, "мэргэжил/гарчиг 120 тэмдэгтээс ихгүй")
		return
	case utf8.RuneCountInString(in.Bio) > 4000:
		writeErr(w, http.StatusBadRequest, "танилцуулга хэт урт")
		return
	case !ownImage(in.AvatarURL, c.UID) || !(ownImage(in.CoverURL, c.UID) || coverTemplate(in.CoverURL)):
		writeErr(w, http.StatusBadRequest, "зураг: http(s) холбоос, өөрийн public файл эсвэл толгойн зургийн загвар")
		return
	case utf8.RuneCountInString(in.Location) > maxLocationLen:
		writeErr(w, http.StatusBadRequest, "байршил 60 тэмдэгтээс ихгүй")
		return
	}
	subjects, msg := cleanSubjects(in.Subjects)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	links, msg := cleanLinks(in.Links)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	cur, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	phone := cur.Phone
	if in.Phone != nil {
		var ok bool
		if phone, ok = cleanPhone(*in.Phone); !ok {
			writeErr(w, http.StatusBadRequest, "утасны дугаар: 8 оронтой (9911 2233) эсвэл «+»-ээр эхэлсэн олон улсын дугаар")
			return
		}
	}
	err = s.store.UpdateProfile(r.Context(), c.UID, store.ProfileUpdate{DisplayName: in.DisplayName, Headline: in.Headline,
		Bio: in.Bio, AvatarURL: in.AvatarURL, CoverURL: in.CoverURL, Subjects: subjects, Location: in.Location, Links: links, Phone: phone})
	if s.storeErr(w, r, err) {
		return
	}
	s.invalidateTeacher(r, c.UID, c.Name)
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

// handleCoverPos: PUT /api/me/cover-pos {"y": 0-100} — нүүр зургийн босоо байрлал (эзэмшигч зургийг чирж тохируулна).
func (s *Server) handleCoverPos(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Y float64 `json:"y"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Y < 0 || in.Y > 100 || in.Y != in.Y {
		writeErr(w, http.StatusBadRequest, "байрлал 0-100")
		return
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		u.CoverY = int(math.Round(in.Y))
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	s.profiles.Delete(u.Username) // профайлын хуудас, хуваалцах зураг шинэ байрлалаар
	writeJSON(w, http.StatusOK, map[string]any{"cover_y": u.CoverY})
}

// handleUpdateName: PUT /api/me/name {"display_name"} — профайл, «Сурах» хуудсан дээрх нэрийг шууд (inline) засна.
// Зөвхөн нэр солигдоно; бусад талбар (танилцуулга, холбоос, утас …) одоогийнхоороо үлдэнэ.
func (s *Server) handleUpdateName(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		DisplayName string `json:"display_name"`
	}
	if !decode(w, r, &in) {
		return
	}
	name := strings.Join(strings.Fields(in.DisplayName), " ")
	if name == "" || utf8.RuneCountInString(name) > 80 {
		writeErr(w, http.StatusBadRequest, "нэр 1-80 тэмдэгт")
		return
	}
	cur, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if cur.DisplayName != name {
		err = s.store.UpdateProfile(r.Context(), c.UID, store.ProfileUpdate{DisplayName: name, Headline: cur.Headline, Bio: cur.Bio,
			AvatarURL: cur.AvatarURL, CoverURL: cur.CoverURL, Subjects: cur.Subjects, Location: cur.Location, Links: cur.Links, Phone: cur.Phone})
		if s.storeErr(w, r, err) {
			return
		}
		s.invalidateTeacher(r, c.UID, cur.Username)
		if cur, err = s.store.UserByID(r.Context(), c.UID); s.storeErr(w, r, err) {
			return
		}
	}
	writeJSON(w, http.StatusOK, s.me(cur))
}

// handleUpdateUsername: багш профайлын холбоосоо (/t/<нэр>) солино.
// Токенд хэрэглэгчийн нэр агуулагддаг тул шинэ токен буцаана.
func (s *Server) handleUpdateUsername(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		Username string `json:"username"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if !usernameRe.MatchString(in.Username) {
		writeErr(w, http.StatusBadRequest, "холбоос: 3-32 тэмдэгт, зөвхөн a-z, 0-9, _")
		return
	}
	old, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if err := s.store.UpdateUsername(r.Context(), c.UID, in.Username); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, http.StatusConflict, "энэ холбоос эзэнтэй байна — өөр нэр сонгоно уу")
			return
		}
		s.storeErr(w, r, err)
		return
	}
	// Хуучин болон шинэ нэрийн профайл, мөн багшийн нэрийг агуулсан сургалтын хуудсуудын кэшийг цэвэрлэнэ.
	s.invalidateTeacher(r, c.UID, old.Username)
	s.profiles.Delete(in.Username)
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	s.issue(w, http.StatusOK, u)
}

// onUserCreated: багш бол түүний файлын сангийн хавтсыг үүсгэнэ.
func (s *Server) onUserCreated(u *store.User) {
	if u.Role == store.RoleTeacher {
		if err := s.files.EnsureTeacherDir(u.ID); err != nil {
			s.log.Error("teacher dir", "err", err, "user", u.ID)
		}
	}
}

// ownImage: хоосон, http(s) холбоос, эсвэл тухайн хэрэглэгчийн нээлттэй сангийн файл.
// coverTemplate — сургалтын толгойн зургийн загвар (/static/covers/t01–t20.svg): профайлын нүүр зурагт ч зөвшөөрнө.
func coverTemplate(u string) bool {
	id, ok := strings.CutPrefix(u, "/static/covers/")
	id, ok2 := strings.CutSuffix(id, ".svg")
	return ok && ok2 && coverRe.MatchString(id)
}

func ownImage(u, uid string) bool {
	return u == "" || validURL(u) || strings.HasPrefix(u, "/files/"+uid+"/"+files.Public+"/")
}

func validURL(u string) bool {
	return (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")) && len(u) <= 1000
}
