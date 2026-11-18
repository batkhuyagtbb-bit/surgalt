package httpapi

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Ангийн найзууд: нэг багшид дагасан (тэр багшийн аль нэг сургалтад элссэн эсвэл хичээл авсан) сурагчид
// чат дээр хоорондоо харагдаж, шууд бичилцэж, дундаасаа бүлэг үүсгэнэ. Багш бүрийн жагсаалтыг roster-оос
// (roster.go, кэштэй) уншина: 30 000 суралцагчтай багшийн найзуудыг хүсэлт бүрт сангаас ачаалахгүй.

// Classmate — чатын "Хүмүүс" жагсаалтын нэг сурагч.
type Classmate struct {
	User    TeacherBrief `json:"user"`
	Via     string       `json:"via"` // хамтдаа дагадаг багшийн нэр
	Courses []string     `json:"courses"`
	ConvID  string       `json:"conv_id,omitempty"` // өмнө нь хувийн яриа нээсэн бол
}

// classmatesLimit — нэг хариуд өгөх найзын дээд тоо (бусдыг нь нэрээр хайж олно).
const classmatesLimit = 300

// myRosters — uid-ийн элссэн сургалтуудын багш нарын суралцагчдын жагсаалт.
func (s *Server) myRosters(ctx context.Context, uid string) ([]*roster, error) {
	enrolled, err := s.store.EnrolledCourses(ctx, uid)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []*roster
	for _, c := range enrolled {
		if seen[c.TeacherID] || c.TeacherID == uid {
			continue
		}
		seen[c.TeacherID] = true
		ro, err := s.teacherRoster(ctx, c.TeacherID)
		if err != nil {
			return nil, err
		}
		out = append(out, ro)
	}
	return out, nil
}

func (s *Server) isClassmate(ctx context.Context, uid, other string) bool {
	if other == "" || other == uid {
		return false
	}
	rs, err := s.myRosters(ctx, uid)
	if err != nil {
		return false
	}
	for _, ro := range rs {
		if _, ok := ro.index[other]; ok {
			return true
		}
	}
	return false
}

func courseTitles(row *StudentRow) []string {
	out := make([]string, 0, len(row.Courses))
	for _, sc := range row.Courses {
		out = append(out, sc.CourseTitle)
	}
	return out
}

// handleClassmates: GET /api/me/classmates?q=&limit=&offset= — эхлээд яриатай, дараа нь онлайн, бусад нь
// шинээр элссэнээс. Нийт тоо X-Total-Count толгойд.
func (s *Server) handleClassmates(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	rs, err := s.myRosters(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	dmWith := map[string]string{}
	if dms, err := s.store.UserDMConversations(ctx, c.UID, 500); err == nil {
		for _, d := range dms {
			other := d.UserID
			if other == c.UID {
				other = d.TeacherID
			}
			dmWith[other] = d.ID
		}
	}
	p := parseListPage(r, classmatesLimit)
	need := p.offset + p.limit
	var dup map[string]bool // олон багштай бол давхардлыг арилгана
	if len(rs) > 1 {
		dup = map[string]bool{}
	}
	var withConv, onNow, rest []Classmate
	total := 0
	for _, ro := range rs {
		for i := range ro.rows {
			row := &ro.rows[i]
			id := row.User.ID
			if id == c.UID || !p.match(row.User) {
				continue
			}
			if dup != nil {
				if dup[id] {
					continue
				}
				dup[id] = true
			}
			total++
			conv := dmWith[id]
			switch {
			case conv != "":
				withConv = append(withConv, Classmate{User: row.User, Via: ro.teacher, Courses: courseTitles(row), ConvID: conv})
			case s.online(id):
				onNow = append(onNow, Classmate{User: row.User, Via: ro.teacher, Courses: courseTitles(row)})
			case len(rest) < need:
				rest = append(rest, Classmate{User: row.User, Via: ro.teacher, Courses: courseTitles(row)})
			}
		}
	}
	all := append(append(withConv, onNow...), rest...)
	out := cut(w, p, all)
	w.Header().Set("X-Total-Count", itoa(total))
	for _, cm := range out {
		s.seen.book.add(c.UID, cm.User.ID)
	}
	if out == nil {
		out = []Classmate{}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleStartDM: POST /api/me/dm/{uid} — ангийн найзтайгаа хувийн яриа.
func (s *Server) handleStartDM(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	other := r.PathValue("uid")
	if other == c.UID {
		writeErr(w, http.StatusBadRequest, "өөртэйгөө чатлах боломжгүй")
		return
	}
	if !s.isClassmate(r.Context(), c.UID, other) {
		writeErr(w, http.StatusForbidden, "зөвхөн нэг багшийн сургалтад суралцдаг ангийн найзтайгаа чатлах боломжтой")
		return
	}
	u, err := s.store.UserByID(r.Context(), other)
	if s.storeErr(w, r, err) {
		return
	}
	conv, err := s.store.CreateDM(r.Context(), c.UID, other, u.DisplayName)
	if s.storeErr(w, r, err) {
		return
	}
	msgs, _ := s.store.Messages(r.Context(), conv.ID, "", 50)
	msgs = s.dropHiddenMsgs(r.Context(), c.UID, conv.ID, msgs)
	role := store.SenderTeacher
	if conv.TeacherID != c.UID {
		role = store.SenderVisitor
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": role})
}

// handleCreateTeam: POST /api/me/teams {title, members[]} — сурагчдын бүлэг (гишүүд ангийн найзууд байна).
func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Title   string   `json:"title"`
		Members []string `json:"members"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if n := utf8.RuneCountInString(in.Title); n < 2 || n > 60 {
		writeErr(w, http.StatusBadRequest, "бүлгийн нэр 2-60 тэмдэгт байх ёстой")
		return
	}
	if len(in.Members) == 0 || len(in.Members) > 100 {
		writeErr(w, http.StatusBadRequest, "1-100 гишүүн сонгоно уу")
		return
	}
	rs, err := s.myRosters(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	allowed := func(id string) bool {
		for _, ro := range rs {
			if _, ok := ro.index[id]; ok && id != c.UID {
				return true
			}
		}
		return false
	}
	for _, m := range in.Members {
		if !allowed(m) {
			writeErr(w, http.StatusForbidden, "бүлэгт зөвхөн ангийн найзуудаа нэмэх боломжтой")
			return
		}
	}
	conv, err := s.store.CreateTeam(r.Context(), c.UID, in.Title, in.Members)
	if s.storeErr(w, r, err) {
		return
	}
	name := s.displayName(r.Context(), c.UID, c.Name)
	for _, m := range in.Members {
		s.notify(r.Context(), &store.Notification{UserID: m, Type: NotifMessage, Count: 1, ActorID: c.UID, Title: "👥 " + name + " таныг «" + in.Title + "» бүлэгт нэмлээ", Link: "/me#chat=" + conv.ID})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"conversation": conv})
}
