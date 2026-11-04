package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"surgalt/internal/chat"
	"surgalt/internal/store"
)

// ---- Багшийн суралцагчдын жагсаалт (roster) ----
// Багшийн бүх сургалтын суралцагчид (элссэн + дангаар хичээл авсан), нэг хүн нэг мөр. Багшийн «Суралцагчид»,
// чатын самбар, ангийн найзууд, «найз мөн үү» шалгалт бүгд үүнээс уншина: 30 000 суралцагчтай багшийн жагсаалтыг
// хүсэлт бүрт сангаас (CourseStudents + 30 000 хүний нэр) дахин ачаалахгүй — 60 секунд кэштэй. Жагсаалтын
// хариуг хуудаслана (limit/offset/q); бие нь массив хэвээр, нийт тоо X-Total-Count толгойд.

type roster struct {
	rows     []StudentRow        // шинээс хуучин (сургалт бүрийн CourseStudents-ийн дарааллаар); ConvID хоосон
	index    map[string]int      // uid → rows дахь байр
	byCourse map[string][]string // сургалт → суралцагчид (CourseStudents-ийн дараалал)
	teacher  string              // багшийн нэр (ангийн найзын «via»)
}

// rosterSmall — үүнээс цөөн суралцагчтай жагсаалтыг элсэлт бүрт шууд шинэчилнэ. Том жагсаалтыг (дахин ачаалах нь
// үнэтэй) элсэлтийн оргил үед секунд тутам дахин бүтээхгүйн тулд 60 секундийн хугацаагаар л шинэчилнэ.
const rosterSmall = 2000

func (s *Server) teacherRoster(ctx context.Context, tid string) (*roster, error) {
	return s.rosters.GetOrLoad(ctx, tid, func(ctx context.Context) (*roster, error) {
		courses, err := s.store.CoursesByTeacher(ctx, tid, false)
		if err != nil {
			return nil, err
		}
		return s.buildRoster(ctx, tid, courses)
	})
}

func (s *Server) buildRoster(ctx context.Context, teacherID string, courses []store.Course) (*roster, error) {
	rows := map[string]*StudentRow{}
	var order []string
	byCourse := map[string][]string{}
	for _, c := range courses {
		students, err := s.store.CourseStudents(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		if len(students) == 0 {
			continue
		}
		titles := map[string]string{}
		if ls, err := s.store.LessonsByCourse(ctx, c.ID); err == nil {
			for _, l := range ls {
				titles[l.ID] = l.Title
			}
		}
		for _, st := range students {
			if st.UserID == teacherID {
				continue
			}
			row, ok := rows[st.UserID]
			if !ok {
				row = &StudentRow{User: TeacherBrief{ID: st.UserID}, Courses: []StudentCourse{}}
				rows[st.UserID] = row
				order = append(order, st.UserID)
			}
			byCourse[c.ID] = append(byCourse[c.ID], st.UserID)
			sc := StudentCourse{CourseID: c.ID, CourseTitle: c.Title, Enrolled: st.EnrolledAt != nil, EnrolledAt: st.EnrolledAt, Lessons: []string{}}
			for _, lid := range st.LessonIDs {
				if t := titles[lid]; t != "" {
					sc.Lessons = append(sc.Lessons, t)
				}
			}
			row.Courses = append(row.Courses, sc)
		}
	}
	out := &roster{index: map[string]int{}, byCourse: byCourse}
	users, err := s.store.UsersByIDs(ctx, append(order, teacherID))
	if err != nil {
		return nil, err
	}
	for i := range users {
		u := &users[i]
		if u.ID == teacherID {
			out.teacher = u.DisplayName
		}
		if row := rows[u.ID]; row != nil {
			row.User = TeacherBrief{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Headline: u.Headline, AvatarURL: u.AvatarURL}
		}
	}
	out.rows = make([]StudentRow, 0, len(order))
	for _, id := range order {
		if rows[id].User.Username != "" { // устсан хэрэглэгчийг алгасна
			out.index[id] = len(out.rows)
			out.rows = append(out.rows, *rows[id])
		}
	}
	for cid, ids := range byCourse { // устсан хэрэглэгчийг сургалтын жагсаалтаас ч хасна
		kept := ids[:0]
		for _, id := range ids {
			if _, ok := out.index[id]; ok {
				kept = append(kept, id)
			}
		}
		byCourse[cid] = kept
	}
	return out, nil
}

// rosterChanged — элсэлт, төлбөрийн дараа: жижиг жагсаалтыг шууд, томыг хугацаагаар нь шинэчилнэ (rosterSmall).
func (s *Server) rosterChanged(teacherID string) {
	if r, ok := s.rosters.Peek(teacherID); ok && r != nil && len(r.rows) >= rosterSmall {
		return
	}
	s.rosters.Delete(teacherID)
}

// listPage — ?q= (нэр, хэрэглэгчийн нэрээр), ?offset=, ?limit= (анх ба дээд тал нь maxLimit).
type listPage struct {
	q             string
	offset, limit int
}

func parseListPage(r *http.Request, maxLimit int) listPage {
	q := r.URL.Query()
	p := listPage{q: strings.ToLower(strings.TrimSpace(q.Get("q"))), limit: maxLimit}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n < maxLimit {
		p.limit = n
	}
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		p.offset = n
	}
	return p
}

func (p listPage) match(b TeacherBrief) bool {
	return p.q == "" || strings.Contains(strings.ToLower(b.DisplayName), p.q) || strings.Contains(strings.ToLower(b.Username), p.q)
}

// cut — нийт тоог X-Total-Count толгойд тавьж, хуудасны хэсгийг буцаана.
func cut[T any](w http.ResponseWriter, p listPage, all []T) []T {
	w.Header().Set("X-Total-Count", strconv.Itoa(len(all)))
	if p.offset >= len(all) {
		return []T{}
	}
	all = all[p.offset:]
	if len(all) > p.limit {
		all = all[:p.limit]
	}
	return all
}

// online — хэрэглэгч одоо сайтад байгаа эсэх (нээлттэй WebSocket).
func (s *Server) online(uid string) bool { return s.hub.Online(chat.TeacherKey(uid)) }

// withConvs — багшийн өмнөх яриаг мөр бүрт холбоно ("Чат" дарахад шууд тэр яриа нээгдэнэ).
func (s *Server) withConvs(ctx context.Context, teacherID string, rows []StudentRow) []StudentRow {
	convs, err := s.store.TeacherConversations(ctx, teacherID, 500)
	if err != nil || len(convs) == 0 {
		return rows
	}
	conv := map[string]string{}
	for _, cv := range convs {
		if cv.Kind != store.ConvGroup && cv.UserID != "" {
			if _, ok := conv[cv.UserID]; !ok {
				conv[cv.UserID] = cv.ID
			}
		}
	}
	out := make([]StudentRow, len(rows))
	for i, r := range rows {
		r.ConvID = conv[r.User.ID]
		out[i] = r
	}
	return out
}

// sortForRail — хязгаартай жагсаалтад эхлээд яриатай, дараа нь онлайн хүмүүс (бусад нь анхны дараалалаараа).
func (s *Server) sortForRail(rows []StudentRow, convs map[string]bool) {
	rank := func(r *StudentRow) int {
		switch {
		case convs[r.User.ID]:
			return 0
		case s.online(r.User.ID):
			return 1
		}
		return 2
	}
	sort.SliceStable(rows, func(i, j int) bool { return rank(&rows[i]) < rank(&rows[j]) })
}
