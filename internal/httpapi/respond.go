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
		writeErr(w, http.StatusNotFound, "олдсонгүй")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "давхардсан эсвэл зөрчилтэй")
	case r.Context().Err() != nil:
		writeErr(w, http.StatusServiceUnavailable, "хүсэлт цуцлагдсан")
	default:
		s.log.Error("store", "err", err, "path", r.URL.Path)
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
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
	return c, err == nil
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

func (s *Server) requireTeacher(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return c, false
	}
	if c.Role != string(store.RoleTeacher) {
		writeErr(w, http.StatusForbidden, "зөвхөн багш")
		return c, false
	}
	return c, true
}

func constantEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
