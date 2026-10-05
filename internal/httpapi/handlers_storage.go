package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// StoragePlan — сарын төлбөртэй багтаамжийн багц (жишээ нь 300MB, 500MB, 1GB, 2GB).
type StoragePlan struct {
	MB    int64  `json:"mb"`
	Label string `json:"label"`
	Price int64  `json:"price_month"` // ₮ / сар
}

func sizeLabel(mb int64) string {
	if mb >= 1024 && mb%1024 == 0 {
		return fmt.Sprintf("%dGB", mb/1024)
	}
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGB", float64(mb)/1024)
	}
	return fmt.Sprintf("%dMB", mb)
}

func (s *Server) plans() []StoragePlan {
	out := make([]StoragePlan, len(s.cfg.StoragePlans))
	for i, p := range s.cfg.StoragePlans {
		p.Label = sizeLabel(p.MB)
		out[i] = p
	}
	return out
}

func (s *Server) quotaOf(u *store.User) int64 {
	return u.StorageQuota(s.cfg.StorageFreeMB<<20, time.Now())
}

func (s *Server) handleStoragePlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{
		"free_mb": s.cfg.StorageFreeMB, "free_label": sizeLabel(s.cfg.StorageFreeMB),
		"plans": s.plans(), "months": []int{1, 3, 6, 12}, "max_image_side": files.MaxImageSide,
	})
}

func (s *Server) handleMyStorage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	used, err := s.files.Usage(c.UID)
	if err != nil {
		s.filesErr(w, err)
		return
	}
	active := u.StorageExpiresAt != nil && time.Now().Before(*u.StorageExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"used": used, "quota": s.quotaOf(u), "free_bytes": s.cfg.StorageFreeMB << 20,
		"plan_mb": u.StorageExtraBytes >> 20, "plan_active": active, "expires_at": u.StorageExpiresAt,
	})
}

// handleBuyStorage: {mb, months} → захиалга. Төлөгдмөгц багтаамж нэмэгдэнэ
// (идэвхтэй багц байвал хугацаа нь үргэлжилж сунгагдана).
func (s *Server) handleBuyStorage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in struct {
		MB     int64 `json:"mb"`
		Months int   `json:"months"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Months < 1 || in.Months > 12 {
		writeErr(w, http.StatusBadRequest, "хугацаа 1-12 сар")
		return
	}
	var plan *StoragePlan
	for _, p := range s.cfg.StoragePlans {
		if p.MB == in.MB {
			plan = &p
			break
		}
	}
	if plan == nil {
		writeErr(w, http.StatusBadRequest, "ийм багц байхгүй")
		return
	}
	o, err := s.store.CreateStorageOrder(r.Context(), c.UID, plan.MB, in.Months, plan.Price*int64(in.Months))
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"order":   o,
		"payment": map[string]any{"amount": o.Amount, "currency": "MNT", "dev_pay": s.cfg.DevPayments},
	})
}
