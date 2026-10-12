package httpapi

import (
	"net/http"
	"time"

	"surgalt/internal/store"
)

// ---- Хууль, итгэл: үйлчилгээний нөхцөл, нууцлалын бодлого (Хувь хүний мэдээлэл хамгаалах тухай хууль) ----

// TermsVersion — /terms ба /privacy хуудасны одоогийн хувилбар. Агуулгыг өөрчлөх бүрт шинэчилбэл хэрэглэгч бүр
// дараагийн удаа нэвтрэхдээ дахин зөвшөөрнө.
const TermsVersion = "2026-10-12"

// meView: өөрийн мэдээлэл — нээлттэй JSON-д гардаггүй утас, зөвшөөрлийн хувилбарыг эзэмшигчид л нэмж өгнө.
type meView struct {
	*store.User
	Phone          string `json:"phone"`
	ConsentVersion string `json:"consent_version"`
	ConsentNeeded  bool   `json:"consent_needed"`
}

func me(u *store.User) meView {
	return meView{User: u, Phone: u.Phone, ConsentVersion: u.ConsentVersion, ConsentNeeded: u.ConsentVersion != TermsVersion}
}

// consentNow — бүртгүүлэх үед зөвшөөрлийг шууд тэмдэглэнэ.
func consentNow(u *store.User) {
	now := time.Now()
	u.ConsentVersion, u.ConsentAt = TermsVersion, &now
}

// handleConsent: POST /api/me/consent {version} — нөхцөл, нууцлалын бодлогыг зөвшөөрнө (хуучин хуудаснаас ирсэн
// хувилбар таарахгүй бол дахин уншуулна).
func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Version string `json:"version"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Version != TermsVersion {
		writeErr(w, http.StatusConflict, "нөхцөл шинэчлэгдсэн байна — хуудсаа сэргээгээд дахин уншина уу")
		return
	}
	if err := s.store.SetConsent(r.Context(), c.UID, TermsVersion); s.storeErr(w, r, err) {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, me(u))
}

// ConsentDemo — демо өгөгдлийн хэрэглэгчид (SEED_DEMO) нөхцөлийг зөвшөөрсөн гэж тэмдэглэнэ.
func ConsentDemo(u *store.User) { consentNow(u) }
