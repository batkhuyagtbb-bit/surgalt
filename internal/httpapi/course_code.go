package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"surgalt/internal/store"
)

// Нууц үгтэй сургалт: багш сургалтын тохиргоонд нууц үг тавина; суралцагч тэр үгийг оруулбал сургалтад
// элсэж (үнэ төлөхгүй) бүх хичээлийг үзнэ. Өмнө нь элссэн хүн ч, багш үгээ сольсон үед ч заавал одоогийн
// нууц үгийг багшаас авч оруулна. Нууц үг нийтэд хэзээ ч гарахгүй, таах оролдлого хязгаартай.

const minCodeLen, maxCodeLen = 4, 64

// normCode — том жижиг үсэг, илүүдэл зайг тооцохгүй (утсаар бичихэд амар).
func normCode(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// codeHash — нууц үгийн хэш (хэвийн болгосны дараа): хэрэглэгч аль үгээр нээснийг харьцуулна.
func codeHash(code string) string {
	sum := sha256.Sum256([]byte("course-code:" + normCode(code)))
	return hex.EncodeToString(sum[:12])
}

// codeUnlocked — багш эсвэл одоогийн нууц үгийг оруулсан хүн (багш үгээ солибол дахин оруулна).
func (s *Server) codeUnlocked(ctx context.Context, uid string, course *store.Course) (bool, error) {
	switch {
	case uid == "":
		return false, nil
	case uid == course.TeacherID:
		return true, nil
	case !course.Locked:
		return false, nil
	}
	h, err := s.store.CourseUnlock(ctx, uid, course.ID)
	return err == nil && h != "" && h == codeHash(course.AccessCode), err
}

// handleUnlockCourse: POST /api/courses/{id}/unlock {"code":"…"}
func (s *Server) handleUnlockCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	course, err := s.store.CourseByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if !course.Published || !course.Locked {
		writeErr(w, http.StatusBadRequest, "энэ сургалт нууц үггүй")
		return
	}
	if course.TeacherID == c.UID {
		writeJSON(w, http.StatusOK, map[string]any{"enrolled": true})
		return
	}
	if ok, err := s.codeUnlocked(ctx, c.UID, course); s.storeErr(w, r, err) {
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{"enrolled": true})
		return
	}
	if !s.codeLim.Allow(c.UID+"|"+course.ID) || !s.codeLim.Allow("ip|"+s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Хэт олон удаа оролдлоо — хагас минутын дараа дахин оролдоно уу")
		return
	}
	if subtle.ConstantTimeCompare([]byte(normCode(in.Code)), []byte(normCode(course.AccessCode))) != 1 {
		writeErr(w, http.StatusForbidden, "Нууц үг буруу байна — багшаасаа дахин асууна уу")
		return
	}
	if err := s.store.SetCourseUnlock(ctx, c.UID, course.ID, codeHash(course.AccessCode)); s.storeErr(w, r, err) {
		return
	}
	enrolled, err := s.store.IsEnrolled(ctx, c.UID, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if !enrolled {
		if err := s.store.Enroll(ctx, c.UID, course); s.storeErr(w, r, err) {
			return
		}
	}
	s.notify(ctx, &store.Notification{UserID: course.TeacherID, Type: NotifEnroll, Count: 1,
		Title: "🔑 Нууц үгээр элслээ", Body: s.displayName(ctx, c.UID, c.Name) + " — " + course.Title, Link: "/c/" + course.ID})
	writeJSON(w, http.StatusCreated, map[string]any{"enrolled": true})
}
