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
	ConsentNeeded  bool   `json:"consent_needed"`     // нөхцөлийг дахин асууна
	IsAdmin        bool   `json:"is_admin,omitempty"` // чатын мэдүүлэг шийдвэрлэх хянагч (moderation.go)
}

func (s *Server) me(u *store.User) meView {
	return meView{User: u, Phone: u.Phone, ConsentVersion: u.ConsentVersion, ConsentNeeded: u.ConsentVersion != TermsVersion, IsAdmin: s.isAdmin(u)}
}

// consentNow — бүртгүүлэх (эсвэл зөвшөөрөх) үед зөвшөөрлийг тэмдэглэнэ.
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
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		consentNow(u)
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

// ConsentDemo — демо өгөгдлийн хэрэглэгчид (SEED_DEMO) нөхцөлийг зөвшөөрсөн гэж тэмдэглэнэ.
func ConsentDemo(u *store.User) { consentNow(u) }
