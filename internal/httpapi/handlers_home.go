package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"surgalt/internal/store"
)

// ---- профайлын нэмэлт талбаруудын шалгалт ----

const (
	maxSubjects    = 8
	maxSubjectLen  = 30
	maxLocationLen = 60
	maxLinkLen     = 300
)

// linkHosts нь холбоосын төрөл бүрт зөвшөөрөгдөх домэйнүүд (хоосон бол дурын).
// Ингэснээр "Facebook" гэсэн товч өөр сайт руу хөтлөх боломжгүй.
var linkHosts = map[string][]string{
	"website":   nil,
	"facebook":  {"facebook.com", "fb.com", "fb.me"},
	"instagram": {"instagram.com"},
	"youtube":   {"youtube.com", "youtu.be"},
}

var linkLabels = map[string]string{"website": "Вэб сайт", "facebook": "Facebook", "instagram": "Instagram", "youtube": "YouTube"}

func hasControl(s string) bool {
	return strings.ContainsFunc(s, unicode.IsControl)
}

// cleanSubjects: тайрч, хоосныг хасаж, давхардлыг (том жижиг үл харгалзан) арилгана.
func cleanSubjects(in []string) ([]string, string) {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, raw := range in {
		v := strings.Join(strings.Fields(raw), " ")
		if v == "" {
			continue
		}
		if utf8.RuneCountInString(v) > maxSubjectLen || hasControl(v) {
			return nil, "чиглэл бүр 30 тэмдэгтээс ихгүй"
		}
		if k := strings.ToLower(v); !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	if len(out) > maxSubjects {
		return nil, "хамгийн ихдээ 8 чиглэл"
	}
	return out, ""
}

// cleanLinks: схемгүй бол https:// нэмж, төрөл бүрийн домэйныг шалгана.
func cleanLinks(in map[string]string) (map[string]string, string) {
	out := map[string]string{}
	for k, raw := range in {
		hosts, known := linkHosts[k]
		if !known {
			return nil, "үл мэдэгдэх холбоосын төрөл: " + k
		}
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if !strings.Contains(v, "://") {
			v = "https://" + v
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil ||
			len(v) > maxLinkLen || hasControl(v) || !strings.Contains(u.Hostname(), ".") {
			return nil, linkLabels[k] + ": холбоос буруу"
		}
		if len(hosts) > 0 && !hostAllowed(strings.ToLower(u.Hostname()), hosts) {
			return nil, linkLabels[k] + ": зөвхөн " + hosts[0] + " холбоос"
		}
		out[k] = u.String()
	}
	return out, ""
}

func hostAllowed(host string, allowed []string) bool {
	for _, a := range allowed {
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// ---- ухаалаг профайл: бүрдлийн оноо ба зөвлөмж ----

type ProfileTip struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Hint  string `json:"hint"`
	Link  string `json:"link"` // студийн аль хэсэгт засах
	Done  bool   `json:"done"`
}

type ProfileInsights struct {
	Score int          `json:"score"` // 0-100
	Level string       `json:"level"` // "Эхлэл" | "Сайн" | "Маш сайн" | "Бүрэн"
	Tips  []ProfileTip `json:"tips"`  // дуусаагүй нь эхэндээ, чухлаараа эрэмбэлсэн
}

// profileInsights нь багшийн профайл суралцагчид хэр итгэл төрүүлэхийг үнэлж,
// дараагийн хийх алхмуудыг чухлаар нь эрэмбэлж өгнө.
func (s *Server) profileInsights(u *store.User, courses []store.Course) ProfileInsights {
	var published, lessons, free, paid int
	for _, c := range courses {
		if !c.Published {
			continue
		}
		published++
		lessons += c.LessonCount
		free += c.FreeLessonCount
		if c.Price > 0 || c.LessonCount > c.FreeLessonCount {
			paid++
		}
	}
	type item struct {
		ProfileTip
		weight int
	}
	items := []item{
		{ProfileTip{"avatar", "Профайл зураг", "Нүүр царайтай профайлд суралцагч илүү итгэдэг.", "#profile", u.AvatarURL != ""}, 15},
		{ProfileTip{"course", "Нийтэлсэн сургалт", "Дор хаяж нэг сургалтаа нийтэлж профайл дээрээ гаргаарай.", "#courses", published > 0}, 15},
		{ProfileTip{"bio", "Дэлгэрэнгүй танилцуулга", "Туршлага, арга барилаа 120+ тэмдэгтээр бичээрэй.", "#profile", utf8.RuneCountInString(strings.TrimSpace(u.Bio)) >= 120}, 15},
		{ProfileTip{"headline", "Мэргэжил / гарчиг", "Нэг өгүүлбэрээр юу заадгаа хэлээрэй.", "#profile", u.Headline != ""}, 10},
		{ProfileTip{"free", "Үнэгүй хичээл", "Үнэгүй хичээл үзсэн хүн худалдан авах магадлал өндөр.", "#courses", free > 0}, 10},
		{ProfileTip{"subjects", "Заадаг чиглэлүүд", "Чиглэлээ нэмбэл суралцагч таныг хурдан ойлгоно.", "#profile", len(u.Subjects) > 0}, 10},
		{ProfileTip{"cover", "Нүүр зураг", "Профайлын дээд талын өргөн зураг хуудсыг амьд болгоно.", "#profile", u.CoverURL != ""}, 5},
		{ProfileTip{"lessons", "3+ хичээл", "Агуулга баялаг байх тусам профайл үнэ цэнтэй харагдана.", "#courses", lessons >= 3}, 5},
		{ProfileTip{"paid", "Төлбөртэй агуулга", "Багц үнэ эсвэл хичээлийн үнэ тавьж орлого олж эхлээрэй.", "#courses", paid > 0}, 5},
		{ProfileTip{"links", "Сошиал холбоос", "Facebook, Instagram, YouTube эсвэл вэб сайтаа холбоорой.", "#profile", len(u.Links) > 0}, 5},
		{ProfileTip{"location", "Байршил", "Хаана байдгаа заавал танхимын сургалтад хэрэгтэй.", "#profile", u.Location != ""}, 5},
	}
	if s.Meet != nil {
		items = append(items, item{ProfileTip{"meet", "Google Meet холболт", "Шууд хичээлийг нэг товчоор товлох боломжтой болно.", "#live", u.MeetConnected}, 5})
	}
	var total, got int
	tips := make([]ProfileTip, 0, len(items))
	for _, it := range items {
		total += it.weight
		if it.Done {
			got += it.weight
		}
		tips = append(tips, it.ProfileTip)
	}
	// Тогтвортой эрэмбэ: дуусаагүй нь эхэнд, жингийн дараалал хэвээр.
	sort.SliceStable(tips, func(i, j int) bool { return !tips[i].Done && tips[j].Done })
	score := got * 100 / total
	level := "Эхлэл"
	switch {
	case score == 100:
		level = "Бүрэн"
	case score >= 80:
		level = "Маш сайн"
	case score >= 50:
		level = "Сайн"
	}
	return ProfileInsights{Score: score, Level: level, Tips: tips}
}

// ---- нүүр хуудас: хэрэглэгчийн бүх зүйл нэг дороос ----

type TeacherBrief struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Headline    string `json:"headline"`
	AvatarURL   string `json:"avatar_url"`
}

// HomeCourse: Access нь "full" (бүх хичээл нээлттэй), "enrolled" (үнэгүй элссэн),
// "lessons" (зөвхөн дангаар авсан хичээлүүд).
type HomeCourse struct {
	Course       store.Course  `json:"course"`
	Teacher      *TeacherBrief `json:"teacher"`
	Access       string        `json:"access"`
	OwnedLessons int           `json:"owned_lessons"`
}

type HomeLesson struct {
	CourseID string     `json:"course_id"`
	LessonID string     `json:"lesson_id"`
	Title    string     `json:"title"`
	PaidAt   *time.Time `json:"paid_at"`
}

type HomeOrder struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	CourseID  string    `json:"course_id"`
	LessonID  string    `json:"lesson_id,omitempty"`
	Title     string    `json:"title"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type HomeMeeting struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CourseID    string    `json:"course_id"`
	CourseTitle string    `json:"course_title"`
	StartsAt    time.Time `json:"starts_at"`
	DurationMin int       `json:"duration_min"`
	MeetURL     string    `json:"meet_url,omitempty"`
}

type HomeChat struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind,omitempty"` // "" хувийн | "group" сургалтын бүлэг
	CourseID      string        `json:"course_id,omitempty"`
	Title         string        `json:"title,omitempty"`
	Teacher       *TeacherBrief `json:"teacher"`
	LastMessage   string        `json:"last_message"`
	LastMessageAt time.Time     `json:"last_message_at"`
}

type HomeTeacherPanel struct {
	Insights     ProfileInsights `json:"insights"`
	Courses      int             `json:"courses"`
	Published    int             `json:"published"`
	Students     int64           `json:"students"`
	ProfileViews int64           `json:"profile_views"`
}

type Home struct {
	User     *store.User       `json:"user"`
	Courses  []HomeCourse      `json:"courses"`
	Lessons  []HomeLesson      `json:"lessons"`
	Pending  []HomeOrder       `json:"pending"`
	Meetings []HomeMeeting     `json:"meetings"`
	Teachers []TeacherBrief    `json:"teachers"`
	Chats    []HomeChat        `json:"chats"`
	Teacher  *HomeTeacherPanel `json:"teacher,omitempty"`
	// Суралцагчийн өөрийн хэсэг: систем олгосон цол, сургалт бүрийн цол, даалгавар/шалгалтын төлөв.
	Rank        *RankInfo        `json:"rank,omitempty"`
	CourseRanks []HomeCourseRank `json:"course_ranks"`
	Tasks       []HomeTask       `json:"tasks"`
	RankAwarded bool             `json:"rank_awarded,omitempty"`
}

const (
	homeMaxOrders   = 200
	homeMaxChats    = 20
	homeMaxMeetings = 20
)

// handleHome нь нэвтэрсэн хэрэглэгчийн нүүр хуудсанд хэрэгтэй бүхнийг нэг хүсэлтээр өгнө:
// сургалтууд, дангаар авсан хичээлүүд, төлөгдөөгүй захиалга, шууд хичээл, багш нар, чат.
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, err := s.store.UserByID(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	enrolled, err := s.store.EnrolledCourses(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	orders, err := s.store.UserOrders(ctx, c.UID, homeMaxOrders)
	if s.storeErr(w, r, err) {
		return
	}
	convs, err := s.store.UserConversations(ctx, c.UID, homeMaxChats)
	if s.storeErr(w, r, err) {
		return
	}
	groups, err := s.store.UserGroupConversations(ctx, c.UID, homeMaxChats)
	if s.storeErr(w, r, err) {
		return
	}
	convs = append(convs, groups...)
	sort.SliceStable(convs, func(i, j int) bool { return convs[i].LastMessageAt.After(convs[j].LastMessageAt) })

	out := Home{User: u, Courses: []HomeCourse{}, Lessons: []HomeLesson{}, Pending: []HomeOrder{},
		Meetings: []HomeMeeting{}, Teachers: []TeacherBrief{}, Chats: []HomeChat{}}

	// Сургалтууд: элссэн + дангаар хичээл авсан (нийтлэгдсэн нь л).
	courses := map[string]store.Course{}
	isEnrolled := map[string]bool{}
	var order []string // харуулах дараалал
	for _, cr := range enrolled {
		courses[cr.ID], isEnrolled[cr.ID] = cr, true
		order = append(order, cr.ID)
	}
	owned := map[string]int{}
	var missing []string
	for _, o := range orders {
		if o.CourseID == "" {
			continue
		}
		if o.Kind == store.OrderKindLesson && o.Status == store.OrderPaid {
			owned[o.CourseID]++
		}
		if _, ok := courses[o.CourseID]; !ok {
			courses[o.CourseID] = store.Course{} // байр эзэлж, давхар асуухгүй
			missing = append(missing, o.CourseID)
		}
	}
	if len(missing) > 0 {
		extra, err := s.store.CoursesByIDs(ctx, missing)
		if s.storeErr(w, r, err) {
			return
		}
		for _, cr := range extra {
			courses[cr.ID] = cr
		}
	}
	visible := func(id string) (store.Course, bool) {
		cr, ok := courses[id]
		return cr, ok && cr.ID != "" && cr.Published
	}
	for _, o := range orders { // дангаар авсан хичээлтэй сургалтууд элссэний дараа
		if o.Kind == store.OrderKindLesson && o.Status == store.OrderPaid && !isEnrolled[o.CourseID] && owned[o.CourseID] > 0 {
			order = append(order, o.CourseID)
			owned[o.CourseID] = -owned[o.CourseID] // нэг л удаа нэмнэ
		}
	}

	// Багш нар (сургалт + чатаас) — нэг багц асуулгаар.
	teacherIDs := map[string]bool{}
	for _, id := range order {
		if cr, ok := visible(id); ok {
			teacherIDs[cr.TeacherID] = true
		}
	}
	for _, cv := range convs {
		teacherIDs[cv.TeacherID] = true
	}
	delete(teacherIDs, c.UID)
	ids := make([]string, 0, len(teacherIDs))
	for id := range teacherIDs {
		ids = append(ids, id)
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if s.storeErr(w, r, err) {
		return
	}
	briefs := make(map[string]*TeacherBrief, len(users))
	for i := range users {
		t := &users[i]
		if t.Role != store.RoleTeacher {
			continue
		}
		briefs[t.ID] = &TeacherBrief{ID: t.ID, Username: t.Username, DisplayName: t.DisplayName, Headline: t.Headline, AvatarURL: t.AvatarURL}
	}

	var courseIDs []string
	for _, id := range order {
		cr, ok := visible(id)
		if !ok {
			continue
		}
		n := owned[id]
		if n < 0 {
			n = -n
		}
		access := "lessons"
		switch {
		case isEnrolled[id] && cr.Price > 0:
			access = "full"
		case isEnrolled[id]:
			access = "enrolled"
		}
		out.Courses = append(out.Courses, HomeCourse{Course: cr, Teacher: briefs[cr.TeacherID], Access: access, OwnedLessons: n})
		courseIDs = append(courseIDs, id)
	}

	for _, o := range orders {
		cr, ok := visible(o.CourseID)
		if !ok {
			continue
		}
		switch {
		case o.Kind == store.OrderKindLesson && o.Status == store.OrderPaid:
			// Багцаар бүгд нээгдсэн бол дангаар авсан хичээлийг тусад нь жагсаах шаардлагагүй.
			if !(isEnrolled[cr.ID] && cr.Price > 0) {
				out.Lessons = append(out.Lessons, HomeLesson{CourseID: o.CourseID, LessonID: o.LessonID, Title: o.Title, PaidAt: o.PaidAt})
			}
		case o.Status == store.OrderPending:
			if o.Kind == store.OrderKindCourse && isEnrolled[cr.ID] {
				continue
			}
			out.Pending = append(out.Pending, HomeOrder{ID: o.ID, Kind: o.Kind, CourseID: o.CourseID, LessonID: o.LessonID,
				Title: o.Title, Amount: o.Amount, CreatedAt: o.CreatedAt})
		}
	}

	// Шууд хичээлүүд: холбоос нь handleCourseMeetings-тэй ижил дүрмээр (элссэн эсвэл үнэгүй сургалт).
	if len(courseIDs) > 0 {
		ms, err := s.store.MeetingsByCourses(ctx, courseIDs, time.Now(), homeMaxMeetings)
		if s.storeErr(w, r, err) {
			return
		}
		for _, m := range ms {
			cr := courses[m.CourseID]
			hm := HomeMeeting{ID: m.ID, Title: m.Title, CourseID: m.CourseID, CourseTitle: cr.Title, StartsAt: m.StartsAt, DurationMin: m.DurationMin}
			if cr.Price == 0 || isEnrolled[cr.ID] {
				hm.MeetURL = m.MeetURL
			}
			out.Meetings = append(out.Meetings, hm)
		}
	}

	for _, cv := range convs {
		if t := briefs[cv.TeacherID]; t != nil {
			hc := HomeChat{ID: cv.ID, Teacher: t, LastMessage: cv.LastMessage, LastMessageAt: cv.LastMessageAt}
			if cv.Kind == store.ConvGroup {
				hc.Kind, hc.CourseID, hc.Title = cv.Kind, cv.CourseID, cv.VisitorName
			}
			out.Chats = append(out.Chats, hc)
		}
	}
	for _, t := range briefs {
		out.Teachers = append(out.Teachers, *t)
	}
	sort.Slice(out.Teachers, func(i, j int) bool { return out.Teachers[i].DisplayName < out.Teachers[j].DisplayName })

	if u.Role == store.RoleTeacher {
		mine, err := s.store.CoursesByTeacher(ctx, u.ID, false)
		if s.storeErr(w, r, err) {
			return
		}
		students, err := s.store.TeacherStudentCount(ctx, u.ID)
		if s.storeErr(w, r, err) {
			return
		}
		p := &HomeTeacherPanel{Insights: s.profileInsights(u, mine), Courses: len(mine), Students: students, ProfileViews: u.ProfileViews}
		for _, cr := range mine {
			if cr.Published {
				p.Published++
			}
		}
		out.Teacher = p
	}

	if u.Role != store.RoleTeacher && len(out.Courses) > 0 {
		rank, byCourse, tasks, awarded := s.myProgress(ctx, c.UID, out.Courses)
		out.Rank, out.CourseRanks, out.Tasks, out.RankAwarded = &rank, byCourse, tasks, awarded
	}
	if out.CourseRanks == nil {
		out.CourseRanks = []HomeCourseRank{}
	}
	if out.Tasks == nil {
		out.Tasks = []HomeTask{}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, out)
}

// handleProfileInsights: багшийн профайлын бүрдлийн оноо, зөвлөмжүүд (студид).
func (s *Server) handleProfileInsights(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	courses, err := s.store.CoursesByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.profileInsights(u, courses))
}

// ---- багшийн суралцагчид ----

type StudentCourse struct {
	CourseID    string     `json:"course_id"`
	CourseTitle string     `json:"course_title"`
	Enrolled    bool       `json:"enrolled"`
	EnrolledAt  *time.Time `json:"enrolled_at,omitempty"`
	Lessons     []string   `json:"lessons"` // дангаар авсан хичээлүүдийн гарчиг
}

type StudentRow struct {
	User    TeacherBrief    `json:"user"` // нэр, зураг, хэрэглэгчийн нэр (имэйл гаргахгүй)
	Courses []StudentCourse `json:"courses"`
	ConvID  string          `json:"conv_id,omitempty"` // өмнө нь чатласан бол тэр яриа
}

// studentRows нь өгөгдсөн сургалтуудын суралцагчдыг нэг хүн нэг мөр болгон нэгтгэнэ.
func (s *Server) studentRows(ctx context.Context, teacherID string, courses []store.Course) ([]StudentRow, error) {
	rows := map[string]*StudentRow{}
	var order []string
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
			sc := StudentCourse{CourseID: c.ID, CourseTitle: c.Title, Enrolled: st.EnrolledAt != nil, EnrolledAt: st.EnrolledAt, Lessons: []string{}}
			for _, lid := range st.LessonIDs {
				if t := titles[lid]; t != "" {
					sc.Lessons = append(sc.Lessons, t)
				}
			}
			row.Courses = append(row.Courses, sc)
		}
	}
	if len(order) == 0 {
		return []StudentRow{}, nil
	}
	users, err := s.store.UsersByIDs(ctx, order)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if row := rows[users[i].ID]; row != nil {
			row.User = TeacherBrief{ID: users[i].ID, Username: users[i].Username, DisplayName: users[i].DisplayName, AvatarURL: users[i].AvatarURL}
		}
	}
	// Өмнө нь чатласан бол яриаг нь холбоно — "Чат" дарахад шууд тэр яриа нээгдэнэ.
	if convs, err := s.store.TeacherConversations(ctx, teacherID, 500); err == nil {
		for _, cv := range convs {
			if row := rows[cv.UserID]; row != nil && cv.Kind != store.ConvGroup {
				row.ConvID = cv.ID
			}
		}
	}
	out := make([]StudentRow, 0, len(order))
	for _, id := range order {
		if rows[id].User.Username != "" { // устсан хэрэглэгчийг алгасна
			out = append(out, *rows[id])
		}
	}
	return out, nil
}

// handleCourseStudents: GET /api/me/courses/{id}/students
func (s *Server) handleCourseStudents(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	rows, err := s.studentRows(r.Context(), c.UID, []store.Course{*course})
	if s.storeErr(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, rows)
}

// handleMyStudents: GET /api/me/students — бүх сургалтын суралцагчид нэг жагсаалтаар.
func (s *Server) handleMyStudents(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	courses, err := s.store.CoursesByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	rows, err := s.studentRows(r.Context(), c.UID, courses)
	if s.storeErr(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, rows)
}

// handleStudentChat: POST /api/me/students/{uid}/chat — багш суралцагчтайгаа шууд яриа эхлүүлнэ.
// Зөвхөн өөрийн сургалтын суралцагчтай (элссэн эсвэл хичээл авсан).
func (s *Server) handleStudentChat(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	uid := r.PathValue("uid")
	student, err := s.store.UserByID(r.Context(), uid)
	if s.storeErr(w, r, err) {
		return
	}
	courses, err := s.store.CoursesByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	member := false
	for _, course := range courses {
		if ok, err := s.courseMember(r.Context(), uid, course.ID); err == nil && ok {
			member = true
			break
		}
	}
	if !member {
		writeErr(w, http.StatusForbidden, "энэ хүн таны сургалтын суралцагч биш байна")
		return
	}
	conv, err := s.store.GetOrCreateConversation(r.Context(), c.UID, "u:"+uid, student.DisplayName, uid)
	if s.storeErr(w, r, err) {
		return
	}
	msgs, err := s.store.Messages(r.Context(), conv.ID, "", 50)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": store.SenderTeacher})
}
