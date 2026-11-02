package httpapi

import (
	"errors"
	"net/http"
	"time"

	"surgalt/internal/store"
)

var errAlreadyTeacher = errors.New("аль хэдийн багш")

// handleAdminMakeTeacher: POST /api/admin/students/{id}/make-teacher — хянагч суралцагчийг багш болгоно.
// Суралцсан түүх, элсэлт, худалдан авалт хэвээр; багшийн студи, профайл, файлын сан нээгдэнэ. Хуучин токен
// «суралцагч» гэж бичигдсэн ч requireTeacher өгөгдлийн сангаас шалгадаг тул дахин нэвтрэх шаардлагагүй.
func (s *Server) handleAdminMakeTeacher(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, err := s.store.UpdateUser(ctx, r.PathValue("id"), func(x *store.User) error {
		if x.Role == store.RoleTeacher {
			return errAlreadyTeacher
		}
		x.Role = store.RoleTeacher
		if x.Plan.TrialFrom == nil { // үнэгүй 3 сар багш болсон өдрөөс
			now := time.Now()
			x.Plan.TrialFrom = &now
		}
		return nil
	})
	if errors.Is(err, errAlreadyTeacher) {
		writeErr(w, http.StatusConflict, "энэ хэрэглэгч аль хэдийн багш")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	s.onUserCreated(u) // багшийн файлын сан
	s.profiles.Delete(u.Username)
	s.notify(ctx, &store.Notification{UserID: u.ID, Type: NotifAccount, Count: 1, Title: "🎓 Таны бүртгэл багшийн бүртгэл боллоо",
		Body: "Одоо сургалт, ном нийтэлж, суралцагчидтайгаа ажиллана. Дээд цэсний «Багшлах» хэсэгт орно уу.", Link: "/me"})
	s.log.Info("хянагч суралцагчийг багш болгов", "user", u.ID, "admin", admin.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "role": u.Role, "username": u.Username})
}
