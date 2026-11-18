package httpapi

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Хичээлийн доорх хэлэлцүүлэг (Facebook маягийн): лайк, сэтгэгдэл, асуулт, хариу. Хичээлд
// "хэлэлцүүлэгтэй" тохиргоо асаалттай үед л. Хичээлд хандах эрхтэй (үнэгүй эсвэл элссэн) хүн оролцоно.

type commentView struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	UserName  string        `json:"user_name"`
	AvatarURL string        `json:"avatar_url,omitempty"`
	Teacher   bool          `json:"teacher"` // багш (сургалтын эзэн)
	Mine      bool          `json:"mine"`
	Body      string        `json:"body"`
	CreatedAt time.Time     `json:"created_at"`
	Likes     int           `json:"likes"`
	Liked     bool          `json:"liked"`
	Replies   []commentView `json:"replies,omitempty"`
	CanDelete bool          `json:"can_delete"`
}

// commentGate — нэг хэрэглэгч 5 секундэд нэг сэтгэгдэл (спамаас хамгаална).
var commentGate = struct {
	sync.Mutex
	last map[string]time.Time
}{last: map[string]time.Time{}}

func (s *Server) discussionLesson(w http.ResponseWriter, r *http.Request, uid string) (*store.Course, *store.Lesson, bool) {
	course, l, ok := s.lessonForUser(w, r, uid, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return nil, nil, false
	}
	if !l.Discussion {
		writeErr(w, http.StatusConflict, "энэ хичээлийн хэлэлцүүлэг хаалттай байна")
		return nil, nil, false
	}
	return course, l, true
}

// handleDiscussion: GET .../discussion — лайк, сэтгэгдлүүд (хариунууд нь дотроо).
func (s *Server) handleDiscussion(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.discussionLesson(w, r, c.UID)
	if !ok {
		return
	}
	out, err := s.discussionData(r.Context(), c.UID, course, l)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// discussionData — хичээлийн хэлэлцүүлэг (лайк, сэтгэгдэл, хариунууд). Хичээл нээх хүсэлтэд ч хамт өгнө (include).
func (s *Server) discussionData(ctx context.Context, uid string, course *store.Course, l *store.Lesson) (map[string]any, error) {
	cms, err := s.store.Comments(ctx, l.ID, 1000)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(cms))
	userIDs := map[string]bool{}
	for i, cm := range cms {
		ids[i] = cm.ID
		userIDs[cm.UserID] = true
	}
	likeCounts, likedMine, err := s.store.Likes(ctx, uid, store.LikeComment, ids)
	if err != nil {
		return nil, err
	}
	lessonLikes, lessonMine, err := s.store.Likes(ctx, uid, store.LikeLesson, []string{l.ID})
	if err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(userIDs))
	for id := range userIDs {
		uids = append(uids, id)
	}
	avatars := map[string]string{}
	if users, err := s.store.UsersByIDs(ctx, uids); err == nil {
		for _, u := range users {
			avatars[u.ID] = u.AvatarURL
		}
	}
	isTeacher := teaches(course, uid)
	view := func(cm store.Comment) commentView {
		return commentView{ID: cm.ID, UserID: cm.UserID, UserName: cm.UserName, AvatarURL: avatars[cm.UserID], Teacher: teaches(course, cm.UserID),
			Mine: cm.UserID == uid, Body: cm.Body, CreatedAt: cm.CreatedAt, Likes: likeCounts[cm.ID], Liked: likedMine[cm.ID], CanDelete: isTeacher || cm.UserID == uid}
	}
	top := []commentView{}
	byID := map[string]int{}
	for _, cm := range cms {
		if cm.ParentID == "" {
			top = append(top, view(cm))
			byID[cm.ID] = len(top) - 1
		}
	}
	for _, cm := range cms {
		if cm.ParentID != "" {
			if i, ok := byID[cm.ParentID]; ok {
				top[i].Replies = append(top[i].Replies, view(cm))
			}
		}
	}
	return map[string]any{"enabled": true, "likes": lessonLikes[l.ID], "liked": lessonMine[l.ID], "count": len(cms), "comments": top, "teacher": isTeacher}, nil
}

// handleAddComment: POST .../discussion {body, parent_id}
func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.discussionLesson(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Body     string `json:"body"`
		ParentID string `json:"parent_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(in.Body); n == 0 || n > 2000 {
		writeErr(w, http.StatusBadRequest, "сэтгэгдэл 1-2000 тэмдэгт байх ёстой")
		return
	}
	var parent *store.Comment
	if in.ParentID != "" {
		p, err := s.store.CommentByID(r.Context(), in.ParentID)
		if err != nil || p.LessonID != l.ID {
			writeErr(w, http.StatusNotFound, "хариулах сэтгэгдэл олдсонгүй")
			return
		}
		if p.ParentID != "" { // нэг түвшний хариу: хариуны хариу нь эх сэтгэгдэлд очно
			in.ParentID = p.ParentID
		}
		parent = p
	}
	commentGate.Lock()
	if t, ok := commentGate.last[c.UID]; ok && time.Since(t) < 5*time.Second {
		commentGate.Unlock()
		writeErr(w, http.StatusTooManyRequests, "хэт хурдан бичиж байна — түр хүлээгээд дахин бичнэ үү")
		return
	}
	commentGate.last[c.UID] = time.Now()
	commentGate.Unlock()
	cm := &store.Comment{CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, UserID: c.UID,
		UserName: s.displayName(r.Context(), c.UID, c.Name), ParentID: in.ParentID, Body: in.Body}
	if err := s.store.AddComment(r.Context(), cm); s.storeErr(w, r, err) {
		return
	}
	link := "/c/" + course.ID + "#l=" + l.ID
	if c.UID != course.TeacherID || len(course.Teachers) > 0 { // багш нарт (бичсэн хүнээс бусад): шинэ сэтгэгдэл/асуулт
		s.notifyCourse(r.Context(), course, c.UID, store.Notification{Type: "comment", ActorID: c.UID, Title: "💬 " + cm.UserName + ": «" + l.Title + "»", Body: short(in.Body), Link: link})
	}
	if parent != nil && parent.UserID != c.UID && parent.UserID != course.TeacherID { // хариу авсан суралцагчид
		s.notify(r.Context(), &store.Notification{UserID: parent.UserID, Type: "reply", ActorID: c.UID, Title: "↩ " + cm.UserName + " танд хариуллаа", Body: short(in.Body), Link: link})
	}
	s.courses.Delete(course.ID)
	writeJSON(w, http.StatusCreated, cm)
}

// handleDeleteComment: DELETE .../discussion/{cid} — өөрийн эсвэл багш.
func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	cm, err := s.store.CommentByID(r.Context(), r.PathValue("cid"))
	if err != nil || cm.LessonID != l.ID {
		writeErr(w, http.StatusNotFound, "сэтгэгдэл олдсонгүй")
		return
	}
	if cm.UserID != c.UID && !teaches(course, c.UID) {
		writeErr(w, http.StatusForbidden, "та зөвхөн өөрийн сэтгэгдлийг устгах боломжтой")
		return
	}
	if err := s.store.DeleteComment(r.Context(), cm.ID); s.storeErr(w, r, err) {
		return
	}
	s.courses.Delete(course.ID)
	w.WriteHeader(http.StatusNoContent)
}

// handleLike: POST .../like {target: lesson|comment, id}
func (s *Server) handleLike(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	_, l, ok := s.discussionLesson(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Target string `json:"target"`
		ID     string `json:"id"`
	}
	if !decode(w, r, &in) {
		return
	}
	switch in.Target {
	case store.LikeLesson:
		in.ID = l.ID
	case store.LikeComment:
		cm, err := s.store.CommentByID(r.Context(), in.ID)
		if err != nil || cm.LessonID != l.ID {
			writeErr(w, http.StatusNotFound, "сэтгэгдэл олдсонгүй")
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, "target: lesson эсвэл comment")
		return
	}
	liked, total, err := s.store.ToggleLike(r.Context(), c.UID, in.Target, in.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"liked": liked, "likes": total})
}
