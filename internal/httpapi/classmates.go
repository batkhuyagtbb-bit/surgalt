package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Ангийн найзууд: нэг багшид дагасан (тэр багшийн аль нэг сургалтад элссэн эсвэл хичээл авсан) сурагчид
// чат дээр хоорондоо харагдаж, шууд бичилцэж, дундаасаа бүлэг үүсгэнэ.

// Classmate — чатын "Хүмүүс" жагсаалтын нэг сурагч.
type Classmate struct {
	User    TeacherBrief `json:"user"`
	Via     string       `json:"via"` // хамтдаа дагадаг багшийн нэр
	Courses []string     `json:"courses"`
	ConvID  string       `json:"conv_id,omitempty"` // өмнө нь хувийн яриа нээсэн бол
}

// classmateIDs — uid-тэй нэг багшид дагадаг сурагчдын ID (өөрийг нь оруулахгүй) ба хамтын мэдээлэл.
func (s *Server) classmates(r *http.Request, uid string) ([]Classmate, error) {
	ctx := r.Context()
	enrolled, err := s.store.EnrolledCourses(ctx, uid)
	if err != nil {
		return nil, err
	}
	// Миний багш нар → тэдний бүх сургалт → тэнд байгаа сурагчид.
	teachers := map[string]bool{}
	for _, c := range enrolled {
		teachers[c.TeacherID] = true
	}
	by := map[string]*Classmate{}
	teacherName := map[string]string{}
	for tid := range teachers {
		courses, err := s.store.CoursesByTeacher(ctx, tid, false)
		if err != nil {
			return nil, err
		}
		for _, c := range courses {
			students, err := s.store.CourseStudents(ctx, c.ID)
			if err != nil {
				return nil, err
			}
			for _, st := range students {
				if st.UserID == uid {
					continue
				}
				cm := by[st.UserID]
				if cm == nil {
					cm = &Classmate{User: TeacherBrief{ID: st.UserID}}
					by[st.UserID] = cm
				}
				cm.Courses = append(cm.Courses, c.Title)
				if cm.Via == "" {
					cm.Via = tid
				}
			}
		}
	}
	if len(by) == 0 {
		return []Classmate{}, nil
	}
	ids := make([]string, 0, len(by)+len(teachers))
	for id := range by {
		ids = append(ids, id)
	}
	for tid := range teachers {
		ids = append(ids, tid)
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		teacherName[u.ID] = u.DisplayName
		if cm := by[u.ID]; cm != nil {
			cm.User = TeacherBrief{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Headline: u.Headline, AvatarURL: u.AvatarURL}
		}
	}
	dms, _ := s.store.UserDMConversations(ctx, uid, 500)
	dmWith := map[string]string{}
	for _, d := range dms {
		other := d.UserID
		if other == uid {
			other = d.TeacherID
		}
		dmWith[other] = d.ID
	}
	out := make([]Classmate, 0, len(by))
	for _, cm := range by {
		if cm.User.DisplayName == "" {
			continue
		}
		cm.Via = teacherName[cm.Via]
		cm.ConvID = dmWith[cm.User.ID]
		out = append(out, *cm)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].User.DisplayName < out[j].User.DisplayName })
	return out, nil
}

func (s *Server) isClassmate(r *http.Request, uid, other string) bool {
	cms, err := s.classmates(r, uid)
	if err != nil {
		return false
	}
	for _, c := range cms {
		if c.User.ID == other {
			return true
		}
	}
	return false
}

// handleClassmates: GET /api/me/classmates
func (s *Server) handleClassmates(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	out, err := s.classmates(r, c.UID)
	if s.storeErr(w, r, err) {
		return
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
	if !s.isClassmate(r, c.UID, other) {
		writeErr(w, http.StatusForbidden, "зөвхөн нэг багшид дагадаг ангийн найзтайгаа чатлана")
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
		writeErr(w, http.StatusBadRequest, "бүлгийн нэр 2-60 тэмдэгт")
		return
	}
	if len(in.Members) == 0 || len(in.Members) > 100 {
		writeErr(w, http.StatusBadRequest, "1-100 гишүүн сонгоно уу")
		return
	}
	cms, err := s.classmates(r, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	allowed := map[string]bool{}
	for _, cm := range cms {
		allowed[cm.User.ID] = true
	}
	for _, m := range in.Members {
		if !allowed[m] {
			writeErr(w, http.StatusForbidden, "зөвхөн ангийн найзуудаа бүлэгт нэмнэ")
			return
		}
	}
	conv, err := s.store.CreateTeam(r.Context(), c.UID, in.Title, in.Members)
	if s.storeErr(w, r, err) {
		return
	}
	name := s.displayName(r.Context(), c.UID, c.Name)
	for _, m := range in.Members {
		s.notify(r.Context(), &store.Notification{UserID: m, Type: NotifMessage, Count: 1, Title: "👥 " + name + " таныг «" + in.Title + "» бүлэгт нэмлээ", Link: "/me#chat=" + conv.ID})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"conversation": conv})
}
