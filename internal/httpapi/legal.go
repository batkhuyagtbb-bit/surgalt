package httpapi

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// ---- Хууль, итгэл: үйлчилгээний нөхцөл, нууцлалын бодлого (Хувь хүний мэдээлэл хамгаалах тухай хууль) ----

// TermsVersion — /terms ба /privacy хуудасны одоогийн хувилбар. Агуулгыг өөрчлөх бүрт шинэчилбэл хэрэглэгч бүр
// дараагийн удаа нэвтрэхдээ дахин зөвшөөрнө.
const TermsVersion = "2026-10-12"

// meView: өөрийн мэдээлэл — нээлттэй JSON-д гардаггүй утас, зөвшөөрөл, насны мэдээллийг эзэмшигчид л нэмж өгнө.
type meView struct {
	*store.User
	Phone          string          `json:"phone"`
	ConsentVersion string          `json:"consent_version"`
	ConsentNeeded  bool            `json:"consent_needed"` // нөхцөлийг (суралцагч бол насаа ч) дахин асууна
	AgeGroup       string          `json:"age_group,omitempty"`
	Guardian       *store.Guardian `json:"guardian,omitempty"`      // 18 хүрээгүй: эцэг эх, асран хамгаалагчийн өгсөн зөвшөөрөл
	GuardianLink   string          `json:"guardian_link,omitempty"` // 18 хүрээгүй: эцэг эхдээ илгээх зөвшөөрлийн холбоос
	IsAdmin        bool            `json:"is_admin,omitempty"`      // чатын мэдүүлэг шийдвэрлэх хянагч (moderation.go)
}

func (s *Server) me(u *store.User) meView {
	v := meView{User: u, Phone: u.Phone, ConsentVersion: u.ConsentVersion, AgeGroup: u.AgeGroup, IsAdmin: s.isAdmin(u),
		ConsentNeeded: u.ConsentVersion != TermsVersion || (u.Role == store.RoleStudent && u.AgeGroup == "")}
	if u.AgeGroup == store.AgeMinor {
		v.GuardianLink = "/guardian/" + s.guardianToken(u.ID)
		if u.Guardian.At != nil {
			g := u.Guardian
			g.Phone = "" // хүүхдэд эцэг эхийнх нь утсыг буцааж харуулах шаардлагагүй
			v.Guardian = &g
		}
	}
	return v
}

// consentNow — бүртгүүлэх (эсвэл зөвшөөрөх) үед зөвшөөрлийг тэмдэглэнэ.
func consentNow(u *store.User) {
	now := time.Now()
	u.ConsentVersion, u.ConsentAt = TermsVersion, &now
}

var errAgeRequired = errors.New("насаа сонгоно уу")

func validAge(g string) bool { return g == store.AgeAdult || g == store.AgeMinor }

// handleConsent: POST /api/me/consent {version, age_group} — нөхцөл, нууцлалын бодлогыг зөвшөөрнө (хуучин хуудаснаас
// ирсэн хувилбар таарахгүй бол дахин уншуулна). Насаа хэлээгүй суралцагч энд нэг удаа сонгоно; дараа нь өөрчлөхгүй.
func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Version  string `json:"version"`
		AgeGroup string `json:"age_group"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Version != TermsVersion {
		writeErr(w, http.StatusConflict, "нөхцөл шинэчлэгдсэн байна — хуудсаа сэргээгээд дахин уншина уу")
		return
	}
	if in.AgeGroup != "" && !validAge(in.AgeGroup) {
		writeErr(w, http.StatusBadRequest, "нас буруу")
		return
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		if u.Role == store.RoleStudent && u.AgeGroup == "" {
			if in.AgeGroup == "" {
				return errAgeRequired
			}
			u.AgeGroup = in.AgeGroup
		}
		consentNow(u)
		return nil
	})
	if errors.Is(err, errAgeRequired) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

// ConsentDemo — демо өгөгдлийн хэрэглэгчид (SEED_DEMO) нөхцөлийг зөвшөөрсөн, насанд хүрсэн гэж тэмдэглэнэ.
func ConsentDemo(u *store.User) {
	consentNow(u)
	if u.Role == store.RoleStudent {
		u.AgeGroup = store.AgeAdult
	}
}

// ---- 18 хүрээгүй суралцагч: эцэг, эх эсвэл асран хамгаалагчийн зөвшөөрөл ----
// Суралцагч өөрийн холбоосыг (/guardian/<токен>) эцэг эхдээ илгээнэ; эцэг эх нээгээд нэр, хамаарал, утсаа бичиж
// зөвшөөрнө (дараа нь мөн холбоосоор цуцалж болно). Зөвшөөрөл ирэх хүртэл биечлэн уулзах цаг захиалахгүй.

// guardianRelations — зөвшөөрөл өгч буй хүний хамаарал (түлхүүр → харагдах нэр).
var guardianRelations = map[string]string{"father": "эцэг", "mother": "эх", "guardian": "асран хамгаалагч"}

func (s *Server) guardianToken(uid string) string { return uid + "." + s.tokens.MAC("guardian", uid) }

func (s *Server) guardianUID(tok string) (string, bool) {
	uid, mac, ok := strings.Cut(tok, ".")
	if !ok || uid == "" || subtle.ConstantTimeCompare([]byte(mac), []byte(s.tokens.MAC("guardian", uid))) != 1 {
		return "", false
	}
	return uid, true
}

// guardianStudent — холбоосны эзэн 18 хүрээгүй суралцагч мөн эсэхийг шалгана.
func (s *Server) guardianStudent(r *http.Request) (*store.User, error) {
	uid, ok := s.guardianUID(r.PathValue("token"))
	if !ok {
		return nil, store.ErrNotFound
	}
	u, err := s.store.UserByID(r.Context(), uid)
	if err != nil {
		return nil, err
	}
	if u.AgeGroup != store.AgeMinor {
		return nil, store.ErrNotFound
	}
	return u, nil
}

// pageGuardian: GET /guardian/{token} — эцэг эх, асран хамгаалагчийн зөвшөөрөл өгөх (цуцлах) хуудас.
func (s *Server) pageGuardian(w http.ResponseWriter, r *http.Request) {
	u, err := s.guardianStudent(r)
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	data := map[string]any{"Student": u.DisplayName, "Token": r.PathValue("token"), "Contact": s.cfg.ContactEmail}
	if data["Student"] == "" {
		data["Student"] = u.Username
	}
	if u.Guardian.At != nil {
		data["Given"] = u.Guardian
		data["Relation"] = guardianRelations[u.Guardian.Relation]
		data["GivenAt"] = u.Guardian.At.In(mnLoc).Format("2006.01.02 15:04")
	}
	if err := s.tmpl.ExecuteTemplate(w, "guardian.html", data); err != nil {
		s.log.Error("template", "err", err, "page", "guardian")
	}
}

// handleGuardianConsent: POST /api/guardian/{token} {name, relation, phone, accept} — эцэг эх зөвшөөрөл өгнө.
func (s *Server) handleGuardianConsent(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	stu, err := s.guardianStudent(r)
	if s.storeErr(w, r, err) {
		return
	}
	var in struct {
		Name     string `json:"name"`
		Relation string `json:"relation"`
		Phone    string `json:"phone"`
		Accept   bool   `json:"accept"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	phone, okPhone := cleanPhone(in.Phone)
	switch n := utf8.RuneCountInString(in.Name); {
	case n < 2 || n > 80:
		writeErr(w, http.StatusBadRequest, "овог нэрээ бичнэ үү")
		return
	case guardianRelations[in.Relation] == "":
		writeErr(w, http.StatusBadRequest, "хамаарлаа сонгоно уу")
		return
	case phone == "" || !okPhone:
		writeErr(w, http.StatusBadRequest, "утасны дугаараа зөв бичнэ үү (8 орон)")
		return
	case !in.Accept:
		writeErr(w, http.StatusBadRequest, "нөхцөл, нууцлалын бодлогыг зөвшөөрнө үү")
		return
	}
	now := time.Now()
	if _, err := s.store.UpdateUser(r.Context(), stu.ID, func(u *store.User) error {
		u.Guardian = store.Guardian{Name: in.Name, Relation: in.Relation, Phone: phone, At: &now}
		return nil
	}); s.storeErr(w, r, err) {
		return
	}
	s.notify(r.Context(), &store.Notification{UserID: stu.ID, Type: NotifGuardian, Link: "/",
		Title: "✅ " + guardianRelations[in.Relation] + " тань зөвшөөрөл өглөө",
		Body:  in.Name + " surgalt.mn-д суралцахыг тань зөвшөөрлөө. Одоо биечлэн уулзах цаг ч захиалж болно."})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "at": now})
}

// handleGuardianRevoke: DELETE /api/guardian/{token} — эцэг эх зөвшөөрлөө цуцална. Удахгүй болох биечлэн
// уулзалттай багш нарт мэдэгдэнэ (цуцлах эсэхийг багш шийднэ).
func (s *Server) handleGuardianRevoke(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	stu, err := s.guardianStudent(r)
	if s.storeErr(w, r, err) {
		return
	}
	if stu.Guardian.At == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	ctx := r.Context()
	if _, err := s.store.UpdateUser(ctx, stu.ID, func(u *store.User) error {
		u.Guardian = store.Guardian{}
		return nil
	}); s.storeErr(w, r, err) {
		return
	}
	ns := []*store.Notification{{UserID: stu.ID, Type: NotifGuardian, Link: "/",
		Title: "Эцэг эх, асран хамгаалагч тань зөвшөөрлөө цуцаллаа", Body: "Биечлэн уулзах шинэ цаг захиалах боломжгүй боллоо."}}
	if slots, err := s.store.StudentSlots(ctx, stu.ID, time.Now()); err == nil {
		told := map[string]bool{}
		for _, sl := range slots {
			if sl.Mode != store.SlotOffline || told[sl.TeacherID] {
				continue
			}
			told[sl.TeacherID] = true
			ns = append(ns, &store.Notification{UserID: sl.TeacherID, Type: NotifBooking, Link: "/me#live",
				Title: "⚠ " + stu.DisplayName + "-ийн эцэг эх зөвшөөрлөө цуцаллаа",
				Body:  "18 хүрээгүй суралцагч. Удахгүй болох биечлэн уулзалтыг цуцлах эсэхээ шийднэ үү."})
		}
	}
	s.notify(ctx, ns...)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// offlineBlocked — 18 хүрээгүй (эсвэл насаа хэлээгүй) суралцагч эцэг эхийн зөвшөөрөлгүйгээр биечлэн уулзах
// цаг захиалахгүй. Хоосон бол зөвшөөрнө.
func offlineBlocked(u *store.User) string {
	switch {
	case u.Role != store.RoleStudent:
		return ""
	case u.AgeGroup == "":
		return "Эхлээд насаа сонгоно уу — хуудсаа сэргээнэ үү"
	case u.NeedsGuardian():
		return "18 хүрээгүй суралцагч биечлэн уулзах цаг захиалахад эцэг эх, асран хамгаалагчийн зөвшөөрөл хэрэгтэй. Нүүр хуудаснаасаа зөвшөөрлийн холбоосоо тэдэнд илгээнэ үү."
	}
	return ""
}
