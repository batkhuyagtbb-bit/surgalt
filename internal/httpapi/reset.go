package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/mail"
	"surgalt/internal/store"
)

// ---- Нууц үг сэргээх: имэйлээр 1 цагийн, нэг удаагийн холбоос ----
// Холбоос нь гарын үсэгтэй (хадгалах шаардлагагүй) бөгөөд одоогийн нууц үгийн хэшийн хурууны хээг агуулна:
// нууц үг солигдмогц бүх холбоос хүчингүй. Сэргээсний дараа өмнөх бүх нэвтрэлт (токен) хүчингүй болно.

const resetTTL = time.Hour

// resetClaim — сэргээх холбоосны агуулга.
type resetClaim struct {
	K   string `json:"k"` // зориулалт (бусад гарын үсэгтэй утгатай андуурахгүй)
	UID string `json:"u"`
	FP  string `json:"f"` // нууц үгийн хэшийн хурууны хээ
}

const resetKind = "pw-reset"

var errResetUsed = errors.New("холбоосын хугацаа дууссан эсвэл ашиглагдсан байна — дахин сэргээх хүсэлт илгээнэ үү")

func (s *Server) resetFP(u *store.User) string {
	return s.tokens.MAC("pw-reset", u.ID+"|"+u.PasswordHash)
}

// handleForgot: POST /api/auth/forgot {email} — бүртгэл байвал сэргээх холбоос илгээнэ. Бүртгэл байгаа эсэхийг
// задлахгүйн тулд үргэлж ижил хариу өгч, захидлыг ар талд илгээнэ (хариуны хугацаа ч ижил).
func (s *Server) handleForgot(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if len(email) < 3 || len(email) > 254 || !strings.Contains(email, "@") {
		writeErr(w, http.StatusBadRequest, "имэйлээ зөв бичнэ үү")
		return
	}
	if s.cfg.Mail == nil && !s.cfg.DevPayments {
		writeErr(w, http.StatusServiceUnavailable, "Нууц үг сэргээх имэйл одоогоор ажиллахгүй байна — Google-ээр нэвтрэх эсвэл бидэнтэй холбогдоно уу")
		return
	}
	if s.resetLim.Allow(email) { // нэг хаяг руу богино хугацаанд олон захидал илгээхгүй
		go s.sendReset(email, s.baseURL(r))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) sendReset(email, base string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	u, err := s.store.UserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			s.log.Warn("нууц үг сэргээх", "err", err)
		}
		return
	}
	tok, err := s.tokens.Seal(resetClaim{K: resetKind, UID: u.ID, FP: s.resetFP(u)}, resetTTL)
	if err != nil {
		s.log.Error("нууц үг сэргээх токен", "err", err)
		return
	}
	link := base + "/reset#t=" + tok
	if s.cfg.Mail == nil { // демо горим: SMTP тохируулаагүй — холбоосыг логт харуулна
		s.log.Info("нууц үг сэргээх холбоос (демо — SMTP тохируулаагүй)", "to", email, "link", link)
		return
	}
	name := u.DisplayName
	if name == "" {
		name = "Сайн байна уу"
	} else {
		name = "Сайн байна уу, " + name
	}
	m := mail.Message{To: u.Email, Subject: "surgalt.mn — нууц үг сэргээх",
		Text: name + ",\n\nsurgalt.mn дээрх бүртгэлийн тань нууц үгийг сэргээх хүсэлт ирлээ. Доорх холбоосоор орж шинэ нууц үг тавина уу (1 цагийн дотор):\n\n" +
			link + "\n\nХэрэв та хүсэлт илгээгээгүй бол энэ захидлыг үл тоомсорлоно уу — нууц үг тань өөрчлөгдөхгүй.\n\nsurgalt.mn",
		HTML: `<div style="font-family:Arial,sans-serif;max-width:520px;margin:0 auto;color:#131b31;line-height:1.6">` +
			`<p style="font-size:22px;font-weight:800;color:#1f3c8f;margin:0 0 16px">surgalt<span style="color:#eaa02e">.mn</span></p>` +
			`<p>` + html.EscapeString(name) + `,</p><p>surgalt.mn дээрх бүртгэлийн тань нууц үгийг сэргээх хүсэлт ирлээ. Доорх товчоор орж шинэ нууц үг тавина уу. Холбоос 1 цагийн дотор хүчинтэй.</p>` +
			`<p style="margin:24px 0"><a href="` + html.EscapeString(link) + `" style="background:#1f3c8f;color:#fff;padding:12px 22px;border-radius:999px;text-decoration:none;font-weight:700">Шинэ нууц үг тавих</a></p>` +
			`<p style="color:#647089;font-size:13px">Хэрэв та хүсэлт илгээгээгүй бол энэ захидлыг үл тоомсорлоно уу — нууц үг тань өөрчлөгдөхгүй.</p></div>`}
	if err := s.cfg.Mail.Send(ctx, m); err != nil {
		s.log.Error("нууц үг сэргээх имэйл илгээж чадсангүй", "err", err, "to", email)
	}
}

// handleReset: POST /api/auth/reset {token, password} — шинэ нууц үг тавьж, өмнөх бүх нэвтрэлтийг хүчингүй болгоод
// энэ төхөөрөмж дээр шууд нэвтрүүлнэ.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	var rc resetClaim
	if err := s.tokens.Open(in.Token, &rc); err != nil || rc.K != resetKind || rc.UID == "" {
		writeErr(w, http.StatusBadRequest, errResetUsed.Error())
		return
	}
	if len(in.Password) < minPassword || len(in.Password) > 72 {
		writeErr(w, http.StatusBadRequest, "нууц үг 6-72 тэмдэгт")
		return
	}
	ctx := r.Context()
	hash, err := auth.HashPassword(ctx, in.Password)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "түр завгүй байна, дахин оролдоно уу")
		return
	}
	now := time.Now()
	u, err := s.store.UpdateUser(ctx, rc.UID, func(u *store.User) error {
		if subtle.ConstantTimeCompare([]byte(s.resetFP(u)), []byte(rc.FP)) != 1 {
			return errResetUsed
		}
		u.PasswordHash, u.TokensAfter = hash, &now
		return nil
	})
	if errors.Is(err, errResetUsed) || errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusBadRequest, errResetUsed.Error())
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	s.alive.Delete(u.ID)
	s.notify(ctx, &store.Notification{UserID: u.ID, Type: NotifAccount, Link: "/privacy#my-data",
		Title: "🔒 Нууц үг тань солигдлоо", Body: "Бусад төхөөрөмж дээрх нэвтрэлт хаагдлаа. Хэрэв та биш бол бидэнтэй яаралтай холбогдоно уу."})
	s.issue(w, http.StatusOK, u)
}
