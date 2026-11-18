package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"surgalt/internal/auth"
	"surgalt/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decode нь биеийн хэмжээг 1MB-аар хязгаарлаж, үл мэдэгдэх талбарыг татгалзана.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "JSON буруу: "+err.Error())
		return false
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		writeErr(w, http.StatusBadRequest, "JSON-ы ард илүү өгөгдөл байна")
		return false
	}
	return true
}

// storeErr нь store-ын алдааг HTTP хариу болгоно. true буцаавал хариу бичигдсэн.
func (s *Server) storeErr(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "мэдээлэл олдсонгүй — устгагдсан байж магадгүй")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "давхардал эсвэл зөрчил үүслээ — хуудсаа сэргээгээд дахин оролдоно уу")
	case r.Context().Err() != nil:
		writeErr(w, http.StatusServiceUnavailable, "хүсэлт цуцлагдлаа")
	default:
		s.log.Error("store", "err", err, "path", r.URL.Path)
		writeErr(w, http.StatusInternalServerError, "системд алдаа гарлаа — түр хүлээгээд дахин оролдоно уу")
	}
	return true
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

// principal нь нэвтэрсэн хэрэглэгч эсвэл зочны claims-г буцаана.
func (s *Server) principal(r *http.Request) (auth.Claims, bool) {
	tok := bearer(r)
	if tok == "" {
		return auth.Claims{}, false
	}
	c, err := s.tokens.Verify(tok)
	if err != nil || (!c.IsGuest() && !s.liveUser(r.Context(), c.UID, c.Iat)) { // бүртгэлээ устгасан бол хуучин токен хүчингүй
		return auth.Claims{}, false
	}
	return c, true
}

// requireUser нь зөвхөн бүртгэлтэй хэрэглэгч (зочин биш).
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := s.principal(r)
	if !ok || c.IsGuest() {
		writeErr(w, http.StatusUnauthorized, "нэвтэрнэ үү")
		return c, false
	}
	return c, true
}

// requireTeacher — багш, системийн эрх нь хүчинтэй (дууссан бол 402: «Данс» хэсгээс сунгана).
func (s *Server) requireTeacher(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok || !s.planOpen(w, r, c.UID) {
		return c, false
	}
	return c, true
}

// requireTeacherAny — багш (системийн эрх дууссан ч): данс, орлого татах, багц сунгах.
func (s *Server) requireTeacherAny(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return c, false
	}
	if c.Role != string(store.RoleTeacher) {
		// Хянагч багш болгосон бол хуучин токен «суралцагч» хэвээр — өгөгдлийн сангаас шалгана.
		if u, err := s.store.UserByID(r.Context(), c.UID); err == nil && u.Role == store.RoleTeacher {
			c.Role = string(store.RoleTeacher)
			return c, true
		}
		writeErr(w, http.StatusForbidden, "энэ үйлдлийг зөвхөн багш хийх боломжтой")
		return c, false
	}
	return c, true
}

func constantEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
