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

	"golang.org/x/sync/errgroup"

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
			return nil, "чиглэл бүр 30 тэмдэгтээс хэтрэхгүй байх ёстой"
		}
		if k := strings.ToLower(v); !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	if len(out) > maxSubjects {
		return nil, "дээд тал нь 8 чиглэл нэмэх боломжтой"
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
			return nil, linkLabels[k] + ": холбоос буруу байна"
		}
		if len(hosts) > 0 && !hostAllowed(strings.ToLower(u.Hostname()), hosts) {
			return nil, linkLabels[k] + ": зөвхөн " + hosts[0] + " холбоос оруулна уу"
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
		{ProfileTip{"avatar", "Профайл зураг", "Нүүр царай нь харагдах зурагтай профайлд суралцагчид илүү итгэдэг.", "#profile", u.AvatarURL != ""}, 15},
		{ProfileTip{"course", "Нийтэлсэн сургалт", "Дор хаяж нэг сургалтаа нийтэлж профайл дээрээ гаргаарай.", "#courses", published > 0}, 15},
		{ProfileTip{"bio", "Дэлгэрэнгүй танилцуулга", "Туршлага, заах арга барилаа 120-оос дээш тэмдэгтээр бичээрэй.", "#profile", utf8.RuneCountInString(strings.TrimSpace(u.Bio)) >= 120}, 15},
		{ProfileTip{"headline", "Мэргэжил / гарчиг", "Юу заадгаа нэг өгүүлбэрээр бичээрэй.", "#profile", u.Headline != ""}, 10},
		{ProfileTip{"free", "Үнэгүй хичээл", "Үнэгүй хичээл үзсэн хүн сургалтыг худалдан авах магадлал өндөр байдаг.", "#courses", free > 0}, 10},
		{ProfileTip{"subjects", "Заадаг чиглэлүүд", "Заадаг чиглэлээ нэмбэл суралцагчид таныг хурдан олж, ойлгоно.", "#profile", len(u.Subjects) > 0}, 10},
		{ProfileTip{"cover", "Нүүр зураг", "Профайлын дээд талын өргөн зураг хуудсыг амьд болгоно.", "#profile", u.CoverURL != ""}, 5},
		{ProfileTip{"lessons", "3+ хичээл", "Агуулга баялаг байх тусам профайл үнэ цэнтэй харагдана.", "#courses", lessons >= 3}, 5},
		{ProfileTip{"paid", "Төлбөртэй агуулга", "Багц үнэ эсвэл хичээлийн үнэ тавьж орлого олж эхлээрэй.", "#courses", paid > 0}, 5},
		{ProfileTip{"links", "Олон нийтийн сүлжээ", "Facebook, Instagram, YouTube эсвэл вэб сайтаа холбоорой.", "#profile", len(u.Links) > 0}, 5},
		{ProfileTip{"location", "Байршил", "Танхимын сургалт явуулдаг бол байршлаа заах нь чухал.", "#profile", u.Location != ""}, 5},
		{ProfileTip{"phone", "Утасны дугаар", "Зочид нэг товшилтоор залгана; дугаарыг тань хэн харсныг танд мэдэгдэнэ.", "#profile", u.Phone != ""}, 5},
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
	Price       int64     `json:"price,omitempty"` // төлбөртэй шууд хичээл
	MembersFree bool      `json:"members_free,omitempty"`
	Bought      bool      `json:"bought,omitempty"`
	// Багштай захиалсан уулзах цаг (booking.go): онлайн бол Meet холбоос, биечлэн бол хаяг.
	Booking         bool   `json:"booking,omitempty"`
	Mode            string `json:"mode,omitempty"`
	Location        string `json:"location,omitempty"`
	TeacherUsername string `json:"teacher_username,omitempty"`
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
	User     meView            `json:"user"`
	Courses  []HomeCourse      `json:"courses"`
	Lessons  []HomeLesson      `json:"lessons"`
	Pending  []HomeOrder       `json:"pending"`
	Meetings []HomeMeeting     `json:"meetings"`
	Teachers []TeacherBrief    `json:"teachers"`
	Chats    []HomeChat        `json:"chats"`
	Teacher  *HomeTeacherPanel `json:"teacher,omitempty"`
	// Суралцагчийн өөрийн хэсэг: систем олгосон эрэмбэ, сургалт бүрийн эрэмбэ, даалгавар/шалгалтын төлөв.
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
	// lite=1 — чатын самбарт зөвхөн чат, багш нар хэрэгтэй: шууд хичээл, багшийн самбар, эрэмбэ тооцохгүй.
	lite := r.URL.Query().Get("lite") == "1"
	// Бие биеэсээ хамааралгүй асуулгуудыг зэрэг явуулна (алсын ClickHouse руу дараалан 7 удаа хүлээхгүй).
	var (
		u                         *store.User
		enrolled                  []store.Course
		orders                    []store.Order
		convs, groups, dms, teams []store.Conversation
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { u, err = s.store.UserByID(gctx, c.UID); return })
	g.Go(func() (err error) { enrolled, err = s.store.EnrolledCourses(gctx, c.UID); return })
	g.Go(func() (err error) { orders, err = s.store.UserOrders(gctx, c.UID, homeMaxOrders); return })
	g.Go(func() (err error) { convs, err = s.store.UserConversations(gctx, c.UID, homeMaxChats); return })
	g.Go(func() (err error) { groups, err = s.store.UserGroupConversations(gctx, c.UID, homeMaxChats); return })
	g.Go(func() error { dms, _ = s.store.UserDMConversations(gctx, c.UID, homeMaxChats); return nil })
	g.Go(func() error { teams, _ = s.store.UserTeamConversations(gctx, c.UID, homeMaxChats); return nil })
	if s.storeErr(w, r, g.Wait()) {
		return
	}
	convs = append(append(append(convs, groups...), dms...), teams...)
	sort.SliceStable(convs, func(i, j int) bool { return convs[i].LastMessageAt.After(convs[j].LastMessageAt) })
	convs = s.dropHiddenConvs(ctx, c.UID, convs)

	out := Home{User: s.me(u), Courses: []HomeCourse{}, Lessons: []HomeLesson{}, Pending: []HomeOrder{},
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
		if cv.Kind == store.ConvDM {
			teacherIDs[cv.UserID] = true
		}
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
	peers := map[string]*TeacherBrief{} // dm-ийн нөгөө сурагч
	for i := range users {
		t := &users[i]
		b := &TeacherBrief{ID: t.ID, Username: t.Username, DisplayName: t.DisplayName, Headline: t.Headline, AvatarURL: t.AvatarURL}
		if t.Role == store.RoleTeacher {
			briefs[t.ID] = b
		}
		peers[t.ID] = b
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

	if !lite {
		if err := s.homeMeetings(ctx, c.UID, &out, courses, isEnrolled, courseIDs); s.storeErr(w, r, err) {
			return
		}
	}

	for _, cv := range convs {
		switch cv.Kind {
		case store.ConvDM: // нөгөө сурагч "teacher" талбарт (нэр, зураг)
			other := cv.UserID
			if other == c.UID {
				other = cv.TeacherID
			}
			if p := peers[other]; p != nil {
				out.Chats = append(out.Chats, HomeChat{ID: cv.ID, Kind: cv.Kind, Teacher: p, Title: p.DisplayName, LastMessage: cv.LastMessage, LastMessageAt: cv.LastMessageAt})
			}
		case store.ConvTeam:
			out.Chats = append(out.Chats, HomeChat{ID: cv.ID, Kind: cv.Kind, Teacher: peers[cv.TeacherID], Title: cv.VisitorName, LastMessage: cv.LastMessage, LastMessageAt: cv.LastMessageAt})
		default:
			if t := briefs[cv.TeacherID]; t != nil {
				hc := HomeChat{ID: cv.ID, Teacher: t, LastMessage: cv.LastMessage, LastMessageAt: cv.LastMessageAt}
				if cv.Kind == store.ConvGroup {
					hc.Kind, hc.CourseID, hc.Title = cv.Kind, cv.CourseID, cv.VisitorName
				}
				out.Chats = append(out.Chats, hc)
			}
		}
	}
	for _, t := range briefs {
		out.Teachers = append(out.Teachers, *t)
	}
	sort.Slice(out.Teachers, func(i, j int) bool { return out.Teachers[i].DisplayName < out.Teachers[j].DisplayName })

	if u.Role == store.RoleTeacher && !lite {
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

	// Эрэмбэ: элссэн сургалтууд + элсээгүй ч үнэгүй хичээл үзэж оноо авсан сургалтууд (myProgress → RankCourseIDs).
	// Багш ч бусдын сургалтад суралцагчтай яг адил суралцана (өөрийн сургалт тооцогдохгүй).
	if !lite {
		rank, byCourse, tasks, awarded := s.myProgress(ctx, c.UID, out.Courses)
		out.CourseRanks, out.Tasks, out.RankAwarded = byCourse, tasks, awarded
		if len(byCourse) > 0 {
			out.Rank = &rank
		}
	}
	if out.CourseRanks == nil {
		out.CourseRanks = []HomeCourseRank{}
	}
	if out.Tasks == nil {
		out.Tasks = []HomeTask{}
	}
	// Чатын жагсаалтын хүмүүс (багш нар, ярианы нөгөө тал) — онлайн төлөвийг нь харж болно.
	s.knowConvs(c.UID, convs)
	for _, t := range out.Teachers {
		s.seen.book.add(c.UID, t.ID)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, out)
}

// homeMeetings — нүүрийн «Шууд хичээл»: элссэн сургалтуудын, худалдаж авсан, багштай захиалсан уулзалтууд.
func (s *Server) homeMeetings(ctx context.Context, uid string, out *Home, courses map[string]store.Course, isEnrolled map[string]bool, courseIDs []string) error {
	// Шууд хичээлүүд: холбоос нь handleCourseMeetings-тэй ижил дүрмээр (meetingOpen): үнэгүй нь элссэн
	// эсвэл үнэгүй сургалтад, төлбөртэй нь худалдаж авсан хүнд.
	bought, _ := s.store.MeetingAccess(ctx, uid)
	listed := map[string]bool{}
	if len(courseIDs) > 0 {
		ms, err := s.store.MeetingsByCourses(ctx, courseIDs, time.Now(), homeMaxMeetings)
		if err != nil {
			return err
		}
		for i := range ms {
			m := &ms[i]
			cr := courses[m.CourseID]
			hm := HomeMeeting{ID: m.ID, Title: m.Title, CourseID: m.CourseID, CourseTitle: cr.Title, StartsAt: m.StartsAt, DurationMin: m.DurationMin,
				Price: m.Price, MembersFree: m.MembersFree, Bought: bought[m.ID]}
			if meetingOpen(m, &cr, uid, isEnrolled[cr.ID], bought) {
				hm.MeetURL = m.MeetURL
			}
			listed[m.ID] = true
			out.Meetings = append(out.Meetings, hm)
		}
	}
	// Сургалтад элсээгүй ч худалдаж авсан төлбөртэй шууд хичээлүүд.
	for id := range bought {
		if listed[id] {
			continue
		}
		m, err := s.store.MeetingByID(ctx, id)
		if err != nil || m.StartsAt.Add(time.Duration(m.DurationMin)*time.Minute).Before(time.Now()) {
			continue
		}
		title := ""
		if cr, err := s.store.CourseByID(ctx, m.CourseID); err == nil {
			title = cr.Title
		}
		out.Meetings = append(out.Meetings, HomeMeeting{ID: m.ID, Title: m.Title, CourseID: m.CourseID, CourseTitle: title, StartsAt: m.StartsAt,
			DurationMin: m.DurationMin, MeetURL: m.MeetURL, Price: m.Price, Bought: true})
	}
	// Багштай захиалсан уулзах цагууд — «Шууд хичээл»-тэй нэг жагсаалтад.
	if bs, err := s.store.StudentSlots(ctx, uid, time.Now().Add(-4*time.Hour)); err == nil {
		for i := range bs {
			b := &bs[i]
			if b.End().Before(time.Now()) {
				continue
			}
			hm := HomeMeeting{ID: b.ID, Title: "Уулзалт", StartsAt: b.StartsAt, DurationMin: b.DurationMin, MeetURL: b.MeetURL,
				Price: b.Price, Bought: b.Price > 0, Booking: true, Mode: b.Mode, Location: b.Location}
			if t, err := s.store.UserByID(ctx, b.TeacherID); err == nil {
				hm.Title, hm.TeacherUsername = "Уулзалт: "+t.DisplayName, t.Username
			}
			out.Meetings = append(out.Meetings, hm)
		}
	}
	sort.SliceStable(out.Meetings, func(i, j int) bool { return out.Meetings[i].StartsAt.Before(out.Meetings[j].StartsAt) })
	return nil
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

// studentsLimit — багшийн суралцагчдын жагсаалтын нэг хуудасны дээд хэмжээ (хуучин клиент limit өгөхгүй бол ч).
const studentsLimit = 1000

// handleCourseStudents: GET /api/me/courses/{id}/students?q=&offset=&limit= — нэг сургалтын суралцагчид.
func (s *Server) handleCourseStudents(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	ro, err := s.teacherRoster(r.Context(), course.TeacherID) // байгууллагын сургалтад эзэмшигчийнх
	if s.storeErr(w, r, err) {
		return
	}
	p := parseListPage(r, studentsLimit)
	rows := []StudentRow{}
	for _, id := range ro.byCourse[course.ID] {
		row := ro.rows[ro.index[id]]
		if !p.match(row.User) {
			continue
		}
		for _, sc := range row.Courses { // зөвхөн энэ сургалтын мэдээлэл
			if sc.CourseID == course.ID {
				row.Courses = []StudentCourse{sc}
				break
			}
		}
		rows = append(rows, row)
	}
	rows = s.withConvs(r.Context(), c.UID, cut(w, p, rows))
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, rows)
}

// handleMyStudents: GET /api/me/students?q=&offset=&limit=&sort=rail — бүх сургалтын суралцагчид нэг жагсаалтаар.
// sort=rail (чатын самбар): яриатай, онлайн хүмүүс эхэнд.
func (s *Server) handleMyStudents(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	ro, err := s.teacherRoster(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	p := parseListPage(r, studentsLimit)
	rows := make([]StudentRow, 0, len(ro.rows))
	for _, row := range ro.rows {
		if p.match(row.User) {
			rows = append(rows, row)
		}
	}
	if r.URL.Query().Get("sort") == "rail" {
		rows = s.withConvs(r.Context(), c.UID, rows)
		has := map[string]bool{}
		for _, row := range rows {
			if row.ConvID != "" {
				has[row.User.ID] = true
			}
		}
		s.sortForRail(rows, has)
		rows = cut(w, p, rows)
	} else {
		rows = s.withConvs(r.Context(), c.UID, cut(w, p, rows))
	}
	for _, row := range rows {
		s.seen.book.add(c.UID, row.User.ID)
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
	msgs = s.dropHiddenMsgs(r.Context(), c.UID, conv.ID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": store.SenderTeacher})
}
