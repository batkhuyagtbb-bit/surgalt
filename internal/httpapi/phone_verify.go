package httpapi

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/store"
	"surgalt/internal/verifymn"
)

// ---- Утас баталгаажуулах, утсаар нууц үг сэргээх (Verify.MN) ----
// Хэрэглэгч өөрийн утаснаас 144773 руу 6 оронтой кодыг SMS-ээр илгээнэ (төлбөрийг илгээгч төлнө) — Verify.MN тэр
// дугаараас ирснийг баталгаажуулна. Нэвтэрсэн хэрэглэгч дугаараа баталгаажуулж (заавал), мартсан үед тэр дугаараараа
// нууц үгээ сэргээнэ. Сесс, дугаар, зорилгыг гарын үсэгтэй токенд хадгална (сервер дээр төлөв хадгалахгүй).

const (
	phoneKind       = "phone-verify"
	phonePurposeMe  = "me"    // нэвтэрсэн хэрэглэгч дугаараа баталгаажуулна
	phonePurposePwd = "reset" // нууц үгээ мартсан
	phoneTokenTTL   = 6 * time.Minute
)

type phoneClaim struct {
	K       string `json:"k"`
	Purpose string `json:"p"`
	UID     string `json:"u,omitempty"`
	Phone   string `json:"ph"`
	Session string `json:"s"`
}

var (
	errPhoneOff   = errors.New("утсаар баталгаажуулах одоогоор идэвхгүй байна")
	errPhoneToken = errors.New("баталгаажуулалтын хугацаа дууссан — дахин эхлүүлнэ үү")
)

// mnPhone — Verify.MN зөвхөн Монголын 8 оронтой дугаар хүлээн авна.
func mnPhone(s string) (string, bool) {
	p, ok := cleanPhone(s)
	if !ok || len(p) != 8 || strings.HasPrefix(p, "+") {
		return "", false
	}
	return p, true
}

func phoneCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b)
}

// startPhone — Verify.MN-д сесс нээж, хэрэглэгчид харуулах зааврыг буцаана.
func (s *Server) startPhone(ctx context.Context, r *http.Request, purpose, uid, phone string) (map[string]any, error) {
	code := phoneCode()
	reply := "surgalt.mn: Utasny dugaar batalgaajlaa. Bayarlalaa!"
	if purpose == phonePurposePwd {
		reply = "surgalt.mn: Batalgaajlaa. Saitad shine nuuts ugee tavina uu."
	}
	sess, err := s.Verify.Create(ctx, phone, code, s.baseURL(r)+"/api/verify/callback", reply)
	if err != nil {
		return nil, err
	}
	tok, err := s.tokens.Seal(phoneClaim{K: phoneKind, Purpose: purpose, UID: uid, Phone: phone, Session: sess.SessionID}, phoneTokenTTL)
	if err != nil {
		return nil, err
	}
	exp := sess.ExpiresAt
	if exp.IsZero() {
		exp = time.Now().Add(5 * time.Minute)
	}
	return map[string]any{"token": tok, "phone": phone, "shortcode": sess.Shortcode, "text": sess.Text, "sms_uri": sess.SMSURI,
		"instruction": sess.Instruction, "expires_at": exp}, nil
}

// checkPhone — токеныг нээж Verify.MN-ээс жинхэнэ төлвийг асууна.
func (s *Server) checkPhone(ctx context.Context, tok, purpose string) (*phoneClaim, string, error) {
	var pc phoneClaim
	if err := s.tokens.Open(tok, &pc); err != nil || pc.K != phoneKind || pc.Purpose != purpose || pc.Session == "" {
		return nil, "", errPhoneToken
	}
	st, err := s.Verify.Status(ctx, pc.Session)
	if err != nil {
		return nil, "", err
	}
	return &pc, st.SessionStatus, nil
}

func (s *Server) verifyErr(w http.ResponseWriter, err error) {
	var ae *verifymn.APIError
	if errors.As(err, &ae) && ae.Status >= 400 && ae.Status < 500 && ae.Status != 401 && ae.Status != 403 {
		writeErr(w, http.StatusBadRequest, "Verify.MN: "+ae.Message)
		return
	}
	s.log.Warn("verify.mn", "err", err)
	writeErr(w, http.StatusBadGateway, "Verify.MN үйлчилгээтэй холбогдож чадсангүй — түр хүлээгээд дахин оролдоно уу")
}

// phoneStatusView — PENDING/EXPIRED үеийн хариу.
func phoneStatusView(status string) map[string]any {
	switch status {
	case verifymn.Expired:
		return map[string]any{"status": "expired", "error": "Хугацаа дууссан — дахин эхлүүлнэ үү"}
	case verifymn.Verified:
		return map[string]any{"status": "verified"}
	}
	return map[string]any{"status": "pending"}
}

// handleRecovery: GET /api/auth/recovery — нууц үг сэргээх ямар арга идэвхтэй.
func (s *Server) handleRecovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, map[string]any{"email": s.cfg.Mail != nil || s.cfg.DevPayments, "phone": s.Verify != nil})
}

// handleMyPhoneSave: PUT /api/me/phone {phone} — сэргээх утсыг түр хадгална (SMS шаардахгүй). Утсаар сэргээх үед
// тэр дугаараас SMS ирснээр баталгаажна.
func (s *Server) handleMyPhoneSave(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Phone string `json:"phone"`
	}
	if !decode(w, r, &in) {
		return
	}
	phone, ok := mnPhone(in.Phone)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Монголын 8 оронтой гар утасны дугаараа бичнэ үү")
		return
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		if phone == u.PhoneVerified {
			u.PhonePending = ""
		} else {
			u.PhonePending = phone
		}
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

// handleMyPhoneStart: POST /api/me/phone/verify {phone}
func (s *Server) handleMyPhoneStart(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if s.Verify == nil {
		writeErr(w, http.StatusServiceUnavailable, errPhoneOff.Error())
		return
	}
	var in struct {
		Phone string `json:"phone"`
	}
	if !decode(w, r, &in) {
		return
	}
	phone, ok := mnPhone(in.Phone)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Монголын 8 оронтой гар утасны дугаараа бичнэ үү")
		return
	}
	if !s.authLim.Allow("ph:" + c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон удаа оролдлоо — түр хүлээгээд дахин оролдоно уу")
		return
	}
	v, err := s.startPhone(r.Context(), r, phonePurposeMe, c.UID, phone)
	if err != nil {
		s.verifyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// handleMyPhoneCheck: POST /api/me/phone/check {token} — баталгаажмагц дугаарыг бүртгэлд хадгална.
func (s *Server) handleMyPhoneCheck(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if s.Verify == nil {
		writeErr(w, http.StatusServiceUnavailable, errPhoneOff.Error())
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	pc, status, err := s.checkPhone(r.Context(), in.Token, phonePurposeMe)
	if err != nil {
		if errors.Is(err, errPhoneToken) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		s.verifyErr(w, err)
		return
	}
	if pc.UID != c.UID {
		writeErr(w, http.StatusForbidden, "танд хандах эрх байхгүй")
		return
	}
	if status != verifymn.Verified {
		writeJSON(w, http.StatusOK, phoneStatusView(status))
		return
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		u.PhoneVerified, u.PhonePending = pc.Phone, ""
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	s.log.Info("утас баталгаажлаа", "user", u.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "verified", "user": s.me(u)})
}

// handleForgotPhone: POST /api/auth/forgot/phone {phone, captcha_id, captcha} — бүртгэл байгаа эсэхийг задлахгүй:
// SMS-ээр дугаараа баталгаажуулсны дараа л тэр дугаартай бүртгэлийг харуулна.
func (s *Server) handleForgotPhone(w http.ResponseWriter, r *http.Request) {
	if !s.authLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон удаа оролдлоо — түр хүлээгээд дахин оролдоно уу")
		return
	}
	if s.Verify == nil {
		writeErr(w, http.StatusServiceUnavailable, errPhoneOff.Error())
		return
	}
	var in struct {
		Phone     string `json:"phone"`
		CaptchaID string `json:"captcha_id"`
		Captcha   string `json:"captcha"`
	}
	if !decode(w, r, &in) {
		return
	}
	phone, ok := mnPhone(in.Phone)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Монголын 8 оронтой гар утасны дугаараа бичнэ үү")
		return
	}
	if !captchas.check(in.CaptchaID, in.Captcha, time.Now()) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Зурган дээрх тоо таарсангүй — шинэ кодыг харж дахин бичнэ үү", "code": "captcha"})
		return
	}
	if !s.resetLim.Allow("ph:" + phone) {
		writeErr(w, http.StatusTooManyRequests, "Энэ дугаарт саяхан код илгээсэн — 2 минутын дараа дахин оролдоно уу")
		return
	}
	v, err := s.startPhone(r.Context(), r, phonePurposePwd, "", phone)
	if err != nil {
		s.verifyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// handleForgotPhoneCheck: POST /api/auth/forgot/phone/check {token} — баталгаажсан бол тэр дугаартай бүртгэл бүрт
// нууц үг сэргээх холбоос (имэйлийнхтэй ижил, 15 минут).
func (s *Server) handleForgotPhoneCheck(w http.ResponseWriter, r *http.Request) {
	if s.Verify == nil {
		writeErr(w, http.StatusServiceUnavailable, errPhoneOff.Error())
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	pc, status, err := s.checkPhone(r.Context(), in.Token, phonePurposePwd)
	if err != nil {
		if errors.Is(err, errPhoneToken) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		s.verifyErr(w, err)
		return
	}
	if status != verifymn.Verified {
		writeJSON(w, http.StatusOK, phoneStatusView(status))
		return
	}
	us, err := s.store.UsersByRecoveryPhone(r.Context(), pc.Phone)
	if s.storeErr(w, r, err) {
		return
	}
	if len(us) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"status": "none", "error": "Энэ дугаартай бүртгэл олдсонгүй. Нууц үгээ имэйлээр сэргээнэ үү."})
		return
	}
	accts := make([]map[string]any, 0, len(us))
	for i := range us {
		u := &us[i]
		if u.PhonePending == pc.Phone { // SMS энэ дугаараас ирсэн — түр хадгалсан дугаар баталгаажлаа
			if nu, err := s.store.UpdateUser(r.Context(), u.ID, func(x *store.User) error {
				if x.PhonePending == pc.Phone {
					x.PhoneVerified, x.PhonePending = pc.Phone, ""
				}
				return nil
			}); err == nil {
				u = nu
			}
		}
		tok, err := s.tokens.Seal(resetClaim{K: resetKind, UID: u.ID, FP: s.resetFP(u)}, 15*time.Minute)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "системд алдаа гарлаа — түр хүлээгээд дахин оролдоно уу")
			return
		}
		accts = append(accts, map[string]any{"name": u.DisplayName, "username": u.Username, "email": maskEmail(u.Email), "role": u.Role, "token": tok})
	}
	s.log.Info("утсаар нууц үг сэргээх", "accounts", len(accts))
	writeJSON(w, http.StatusOK, map[string]any{"status": "verified", "accounts": accts})
}

// maskEmail — «ba***@gmail.com».
func maskEmail(e string) string {
	at := strings.LastIndex(e, "@")
	if at < 0 || strings.HasSuffix(e, "@users.noreply.surgalt") {
		return ""
	}
	name := []rune(e[:at])
	keep := min(2, len(name))
	return string(name[:keep]) + "***" + e[at:]
}

// handleVerifyCallback: GET /api/verify/callback — Verify.MN-ийн мэдэгдэл (агуулгагүй). Төлвийг хэрэглэгчийн
// хуудас өөрөө /check-ээр шалгадаг тул энд зөвхөн хүлээн авснаа хэлнэ.
func (s *Server) handleVerifyCallback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
