package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/auth"
	"surgalt/internal/store"
)

// ---- Байгууллага ----
// Хянагч байгууллагын бүртгэл үүсгэж, борлуулалтын шимтгэл, файлын санг гэрээгээр тогтооно. Байгууллага өөрөө нэвтэрч
// «Сургалтууд» хэсэгт мэргэжлийн хөтөлбөр, сургалтаа үүсгээд сургалт бүрт багш холбоно. Холбосон багш тухайн сургалтад
// хичээл оруулж, зохион байгуулна (хичээл, дүгнэлт, хэлэлцүүлэг, журнал); сургалтын үнэ, нийтлэх, устгах, орлого, файлын
// сан — байгууллагынх. Сургалт багшийн профайл дээр гарахгүй: байгууллагын хуудаснаас «хэн заадаг»-аар нь багшийн профайл
// руу холбогдоно.

const (
	maxPrograms       = 50
	maxCourseTeachers = 10
	maxOrgStorageMB   = 1 << 20 // 1 TB
)

// teaches — сургалтыг удирдах эрх: эзэмшигч (багш, байгууллага) эсвэл байгууллагын холбосон багш.
func teaches(c *store.Course, uid string) bool {
	return uid != "" && c != nil && (c.TeacherID == uid || slices.Contains(c.Teachers, uid))
}

// teachCourse — ownCourse шиг, гэхдээ байгууллагын холбосон багшийг ч зөвшөөрнө (хичээл, журнал, дүгнэлт). Системийн
// эрхийг сургалтын эзэмшигчээр шалгана: байгууллагын багш өөрийн багцгүй ч байгууллагын сургалтаа заана.
func (s *Server) teachCourse(w http.ResponseWriter, r *http.Request, uid string) (*store.Course, bool) {
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return nil, false
	}
	if !teaches(course, uid) {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return nil, false
	}
	if !s.planOpen(w, r, course.TeacherID) {
		return nil, false
	}
	return course, true
}

// analyticsOwner — аналитикийн хүрээ: ?course= нь байгууллагаас холбосон сургалт бол эзэмшигчийнх (үйл явдал, сесс
// эзэмшигчийн ID-гаар бичигддэг), бусад үед өөрийнх. Системийн эрхийг тэр хүрээний эзнээр шалгана.
func (s *Server) analyticsOwner(w http.ResponseWriter, r *http.Request, uid string) (string, bool) {
	if cid := r.URL.Query().Get("course"); cid != "" {
		if co, err := s.store.CourseByID(r.Context(), cid); err == nil && co.TeacherID != uid && teaches(co, uid) {
			return co.TeacherID, s.planOpen(w, r, co.TeacherID)
		}
	}
	return uid, s.planOpen(w, r, uid)
}

// courseEdited — хичээл нэмэх, засах, устгах, дараалал солиход нийтийн кэшийг (сургалт, хайлт, эзэмшигчийн профайл) цэвэрлэнэ.
func (s *Server) courseEdited(ctx context.Context, course *store.Course) {
	s.courses.Delete(course.ID)
	s.search.markDirty()
	if u, err := s.store.UserByID(ctx, course.TeacherID); err == nil {
		s.profiles.Delete(u.Username)
	}
}

// sharedGroups — байгууллагаас холбосон сургалтуудын бүлэг чатууд (холбосон багшийн чатын жагсаалт, WebSocket-д).
func (s *Server) sharedGroups(ctx context.Context, uid string) []store.Conversation {
	cs, err := s.store.CoursesTaughtBy(ctx, uid)
	if err != nil || len(cs) == 0 {
		return nil
	}
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].ID
	}
	convs, _ := s.store.GroupConversationsByCourses(ctx, ids)
	return convs
}

// notifyCourse — сургалтын эзэмшигч ба холбосон багш нарт (үйлдэл хийсэн хүнээс бусад) мэдэгдэл: даалгаврын хариу,
// хэлэлцүүлэг, шалгалт — зохион байгуулж буй багш шууд мэднэ.
func (s *Server) notifyCourse(ctx context.Context, course *store.Course, actor string, n store.Notification) {
	for _, uid := range append([]string{course.TeacherID}, course.Teachers...) {
		if uid == actor {
			continue
		}
		m := n
		m.UserID = uid
		s.notify(ctx, &m)
	}
}

// TeacherRef — багш, байгууллагын товч мэдээлэл (карт, жагсаалтад).
type TeacherRef struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Headline    string `json:"headline,omitempty"`
}

func refOf(u *store.User) TeacherRef {
	return TeacherRef{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, AvatarURL: u.AvatarURL, Headline: u.Headline}
}

// peopleRefs — ID-нуудын дарааллаар (устсан хэрэглэгчийг алгасна).
func (s *Server) peopleRefs(ctx context.Context, ids []string) map[string]TeacherRef {
	out := map[string]TeacherRef{}
	if len(ids) == 0 {
		return out
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if err != nil {
		return out
	}
	for i := range users {
		out[users[i].ID] = refOf(&users[i])
	}
	return out
}

func pickRefs(refs map[string]TeacherRef, ids []string) []TeacherRef {
	out := make([]TeacherRef, 0, len(ids))
	for _, id := range ids {
		if p, ok := refs[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

// orgOf — дуудагч байгууллага бол түүний бүртгэл.
func (s *Server) orgOf(ctx context.Context, uid string) (*store.User, bool) {
	u, err := s.store.UserByID(ctx, uid)
	if err != nil || u.Org == nil {
		return nil, false
	}
	return u, true
}

// orgView — байгууллагын өөрийн мэдээлэл (/api/me, /api/me/org).
type orgView struct {
	Pct       float64         `json:"commission_pct"`
	StorageMB int64           `json:"storage_mb"`
	Programs  []store.Program `json:"programs"`
}

func viewOrg(o *store.Org) *orgView {
	if o == nil {
		return nil
	}
	ps := o.Programs
	if ps == nil {
		ps = []store.Program{}
	}
	return &orgView{Pct: o.Pct, StorageMB: o.StorageMB, Programs: ps}
}

// ---- Байгууллагын студи ----

// handleMyOrg: GET /api/me/org — гэрээний нөхцөл, хөтөлбөрүүд.
func (s *Server) handleMyOrg(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	u, ok := s.orgOf(r.Context(), c.UID)
	if !ok {
		writeErr(w, http.StatusNotFound, "байгууллагын бүртгэл биш")
		return
	}
	writeJSON(w, http.StatusOK, viewOrg(u.Org))
}

// handleOrgPrograms: PUT /api/me/org/programs {programs:[{id?, name, description}]} — мэргэжлийн хөтөлбөрүүдийг
// (дарааллаар нь) бүхэлд нь хадгална. Шинэд ID олгоно; хассан хөтөлбөрийн сургалтууд «хөтөлбөргүй» болно.
func (s *Server) handleOrgPrograms(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	var in struct {
		Programs []store.Program `json:"programs"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Programs) > maxPrograms {
		writeErr(w, http.StatusBadRequest, "хөтөлбөр 50-аас ихгүй")
		return
	}
	names := map[string]bool{}
	for i := range in.Programs {
		p := &in.Programs[i]
		p.Name = strings.Join(strings.Fields(p.Name), " ")
		p.Description = strings.TrimSpace(p.Description)
		switch {
		case p.Name == "" || utf8.RuneCountInString(p.Name) > 120:
			writeErr(w, http.StatusBadRequest, "хөтөлбөрийн нэр 1-120 тэмдэгт")
			return
		case utf8.RuneCountInString(p.Description) > 1000:
			writeErr(w, http.StatusBadRequest, "хөтөлбөрийн тайлбар 1000 тэмдэгтээс ихгүй")
			return
		case names[strings.ToLower(p.Name)]:
			writeErr(w, http.StatusBadRequest, "«"+p.Name+"» нэртэй хөтөлбөр давхардаж байна")
			return
		}
		names[strings.ToLower(p.Name)] = true
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		if u.Org == nil {
			return store.ErrNotFound
		}
		known := map[string]bool{}
		for _, p := range u.Org.Programs {
			known[p.ID] = true
		}
		seen := map[string]bool{}
		for i := range in.Programs {
			if p := &in.Programs[i]; !known[p.ID] || seen[p.ID] {
				p.ID = randHex(6)
			}
			seen[in.Programs[i].ID] = true
		}
		u.Org.Programs = in.Programs
		return nil
	})
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "байгууллагын бүртгэл биш")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	s.profiles.Delete(u.Username)
	writeJSON(w, http.StatusOK, viewOrg(u.Org))
}

// validProgram — байгууллагын хөтөлбөрийн ID (хоосон = хөтөлбөргүй); байгууллага биш бол үргэлж хоосон.
func validProgram(u *store.User, id string) (string, bool) {
	if id == "" {
		return "", true
	}
	if u == nil || u.Org.ProgramByID(id) == nil {
		return "", false
	}
	return id, true
}

// resolveTeacher — «@нэр», нэр, имэйл эсвэл профайлын холбоосоор (…/t/нэр) багшийг олно.
func (s *Server) resolveTeacher(ctx context.Context, who string) (*store.User, error) {
	who = strings.TrimSpace(who)
	if i := strings.LastIndex(who, "/t/"); i >= 0 {
		who = who[i+3:]
		if j := strings.IndexAny(who, "/?#"); j >= 0 {
			who = who[:j]
		}
	}
	who = strings.ToLower(strings.TrimPrefix(who, "@"))
	if who == "" {
		return nil, store.ErrNotFound
	}
	if strings.Contains(who, "@") {
		if _, err := mail.ParseAddress(who); err != nil {
			return nil, store.ErrNotFound
		}
		return s.store.UserByEmail(ctx, who)
	}
	return s.store.UserByUsername(ctx, who)
}

// handleAddCourseTeacher: POST /api/courses/{id}/teachers {who} — байгууллага сургалтдаа багш холбоно.
func (s *Server) handleAddCourseTeacher(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	org, ok := s.orgOf(ctx, c.UID)
	if !ok {
		writeErr(w, http.StatusForbidden, "багш холбох нь зөвхөн байгууллагын сургалтад")
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Who string `json:"who"`
	}
	if !decode(w, r, &in) {
		return
	}
	t, err := s.resolveTeacher(ctx, in.Who)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "ийм хэрэглэгч олдсонгүй — багшийн @нэр, имэйл эсвэл профайлын холбоосыг оруулна уу")
		return
	case s.storeErr(w, r, err):
		return
	case t.ID == c.UID:
		writeErr(w, http.StatusBadRequest, "байгууллага өөрөө сургалтын эзэмшигч — багшаа холбоно уу")
		return
	case t.Role != store.RoleTeacher:
		writeErr(w, http.StatusBadRequest, "«"+t.DisplayName+"» багшийн бүртгэлгүй байна — эхлээд багшаар бүртгүүлэх шаардлагатай")
		return
	case t.Org != nil:
		writeErr(w, http.StatusBadRequest, "байгууллагын бүртгэлийг багшаар холбохгүй")
		return
	case slices.Contains(course.Teachers, t.ID):
		writeErr(w, http.StatusConflict, "«"+t.DisplayName+"» энэ сургалтад аль хэдийн холбогдсон")
		return
	case len(course.Teachers) >= maxCourseTeachers:
		writeErr(w, http.StatusBadRequest, "нэг сургалтад 10 хүртэл багш холбоно")
		return
	}
	updated, err := s.store.SetCourseOrg(ctx, c.UID, course.ID, course.ProgramID, append(slices.Clone(course.Teachers), t.ID))
	if s.storeErr(w, r, err) {
		return
	}
	s.courseEdited(ctx, updated)
	s.notify(ctx, &store.Notification{UserID: t.ID, Type: NotifAccount, Count: 1, ActorID: c.UID,
		Title: "🏫 " + org.DisplayName + " таныг багшаар холболоо",
		Body:  "«" + course.Title + "» сургалтад хичээл оруулж, зохион байгуулна. Студийн «Сургалтууд» хэсэгт харагдана.", Link: "/me#course=" + course.ID})
	s.log.Info("байгууллага багш холбов", "org", c.UID, "course", course.ID, "teacher", t.ID)
	writeJSON(w, http.StatusOK, map[string]any{"teachers": updated.Teachers, "teacher_list": pickRefs(s.peopleRefs(ctx, updated.Teachers), updated.Teachers)})
}

// handleRemoveCourseTeacher: DELETE /api/courses/{id}/teachers/{uid} — байгууллага багшийг салгана, эсвэл багш өөрөө
// сургалтаас гарна.
func (s *Server) handleRemoveCourseTeacher(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	course, err := s.store.CourseByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	tid := r.PathValue("uid")
	self := tid == c.UID && slices.Contains(course.Teachers, c.UID)
	if course.TeacherID != c.UID && !self {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if !slices.Contains(course.Teachers, tid) {
		writeErr(w, http.StatusNotFound, "энэ багш сургалтад холбогдоогүй")
		return
	}
	updated, err := s.store.SetCourseOrg(ctx, course.TeacherID, course.ID, course.ProgramID, slices.DeleteFunc(slices.Clone(course.Teachers), func(x string) bool { return x == tid }))
	if s.storeErr(w, r, err) {
		return
	}
	s.courseEdited(ctx, updated)
	if self {
		if me, err := s.store.UserByID(ctx, c.UID); err == nil {
			s.notify(ctx, &store.Notification{UserID: course.TeacherID, Type: NotifAccount, Count: 1, ActorID: c.UID,
				Title: me.DisplayName + " «" + course.Title + "» сургалтаас гарлаа", Link: "/me#course=" + course.ID})
		}
	} else {
		s.notify(ctx, &store.Notification{UserID: tid, Type: NotifAccount, Count: 1, ActorID: c.UID,
			Title: "«" + course.Title + "» сургалтын багшийн эрх хаагдлаа", Body: "Байгууллага таныг энэ сургалтаас салгалаа."})
	}
	writeJSON(w, http.StatusOK, map[string]any{"teachers": updated.Teachers, "teacher_list": pickRefs(s.peopleRefs(ctx, updated.Teachers), updated.Teachers)})
}

// ---- Хянагч: байгууллагын бүртгэл ----

const maxOrgManagers = 10

// adminPerson — хянагчид харагдах хэрэглэгчийн товч мэдээлэл (имэйлтэй).
type adminPerson struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role,omitempty"`
}

func adminPersonOf(u *store.User) adminPerson {
	return adminPerson{ID: u.ID, Name: u.DisplayName, Username: u.Username, Email: u.Email, AvatarURL: u.AvatarURL, Role: string(u.Role)}
}

type adminOrgRow struct {
	ID        string        `json:"id"`
	Username  string        `json:"username"`
	Name      string        `json:"name"`
	Email     string        `json:"email"`
	Login     bool          `json:"login"` // тусдаа имэйл, нууц үгээр нэвтэрч болно
	AvatarURL string        `json:"avatar_url"`
	Pct       float64       `json:"commission_pct"`
	StorageMB int64         `json:"storage_mb"`
	UsedBytes int64         `json:"used_bytes"`
	Programs  int           `json:"programs"`
	Courses   int           `json:"courses"`
	Published int           `json:"courses_published"`
	Teachers  int           `json:"teachers"`
	Managers  []adminPerson `json:"managers"`
	CreatedAt time.Time     `json:"created_at"`
	// Системийн эрх: үнэгүй хугацаа эсвэл байгууллагын багц, дуусах огноо.
	Plan       string    `json:"plan"`
	PlanName   string    `json:"plan_name"`
	PlanUntil  time.Time `json:"plan_until"`
	PlanActive bool      `json:"plan_active"`
}

func (s *Server) adminOrgRow(ctx context.Context, u *store.User, courses []store.Course) adminOrgRow {
	row := adminOrgRow{ID: u.ID, Username: u.Username, Name: u.DisplayName, Email: u.Email, Login: u.PasswordHash != "" && !orgPlaceholderEmail(u.Email),
		AvatarURL: u.AvatarURL, Pct: u.Org.Pct, StorageMB: u.Org.StorageMB, Programs: len(u.Org.Programs), CreatedAt: u.CreatedAt, Managers: []adminPerson{}}
	if row.Login == false && orgPlaceholderEmail(u.Email) {
		row.Email = ""
	}
	row.Plan, _, row.PlanActive = u.PlanAt(time.Now())
	row.PlanName, row.PlanUntil = planName(row.Plan), u.PlanUntil()
	row.UsedBytes, _ = s.files.Usage(u.ID)
	teachers := map[string]bool{}
	for _, c := range courses {
		if c.TeacherID != u.ID {
			continue
		}
		row.Courses++
		if c.Published {
			row.Published++
		}
		for _, t := range c.Teachers {
			teachers[t] = true
		}
	}
	row.Teachers = len(teachers)
	if len(u.Org.Managers) > 0 {
		if ms, err := s.store.UsersByIDs(ctx, u.Org.Managers); err == nil {
			for i := range ms {
				row.Managers = append(row.Managers, adminPersonOf(&ms[i]))
			}
		}
	}
	return row
}

func (s *Server) adminOrgReply(w http.ResponseWriter, r *http.Request, status int, u *store.User) {
	courses, _ := s.store.CoursesByTeacher(r.Context(), u.ID, false)
	writeJSON(w, status, s.adminOrgRow(r.Context(), u, courses))
}

// orgPlaceholderEmail — тусдаа нэвтрэлтгүй (зөвхөн удирдагчаар) байгууллагын дотоод имэйл.
const orgEmailDomain = "@org.surgalt.invalid"

func orgPlaceholderEmail(e string) bool { return strings.HasSuffix(e, orgEmailDomain) }

// handleAdminOrgs: GET /api/admin/orgs — байгууллагууд.
func (s *Server) handleAdminOrgs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	ctx := r.Context()
	users, err := s.store.AdminUsers(ctx)
	if s.storeErr(w, r, err) {
		return
	}
	courses, err := s.store.AdminCourses(ctx)
	if s.storeErr(w, r, err) {
		return
	}
	rows := []adminOrgRow{}
	for i := range users {
		if users[i].Org != nil {
			rows = append(rows, s.adminOrgRow(ctx, &users[i], courses))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"orgs": rows, "default_pct": store.OrgFeePct, "free_mb": s.cfg.StorageFreeMB})
}

// handleAdminUserSearch: GET /api/admin/users?q= — удирдагч сонгоход хэрэглэгч хайх (нэр, @нэр, имэйл; байгууллагаас бусад).
func (s *Server) handleAdminUserSearch(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	q := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("q")), "@"))
	out := []adminPerson{}
	if utf8.RuneCountInString(q) < 2 {
		writeJSON(w, http.StatusOK, out)
		return
	}
	users, err := s.store.AdminUsers(r.Context())
	if s.storeErr(w, r, err) {
		return
	}
	for i := range users {
		u := &users[i]
		if u.Org != nil {
			continue
		}
		if strings.Contains(strings.ToLower(u.DisplayName), q) || strings.Contains(u.Username, q) || strings.Contains(strings.ToLower(u.Email), q) {
			out = append(out, adminPersonOf(u))
			if len(out) == 20 {
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

type orgTerms struct {
	Pct       float64 `json:"commission_pct"`
	StorageMB int64   `json:"storage_mb"`
}

func (t orgTerms) validate() string {
	switch {
	case t.Pct < 0 || t.Pct > 100:
		return "шимтгэл 0-100%"
	case t.StorageMB < 0 || t.StorageMB > maxOrgStorageMB:
		return "файлын сан 0-1,048,576 MB (1 TB)"
	}
	return ""
}

func validOrgName(name string) string {
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return "байгууллагын нэр 1-80 тэмдэгт"
	}
	return ""
}

// managerCandidate — удирдагч болох хэрэглэгч (байгууллагын бүртгэл биш).
func (s *Server) managerCandidate(w http.ResponseWriter, r *http.Request, id string) (*store.User, bool) {
	m, err := s.store.UserByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "удирдах хэрэглэгч олдсонгүй")
		return nil, false
	}
	if s.storeErr(w, r, err) {
		return nil, false
	}
	if m.Org != nil {
		writeErr(w, http.StatusBadRequest, "байгууллагын бүртгэлийг удирдагчаар холбохгүй — хүний бүртгэл сонгоно уу")
		return nil, false
	}
	return m, true
}

func (s *Server) notifyManager(ctx context.Context, uid string, org *store.User) {
	s.managed.Delete(uid)
	s.notify(ctx, &store.Notification{UserID: uid, Type: NotifAccount, Count: 1, ActorID: org.ID,
		Title: "🏫 Та «" + org.DisplayName + "» байгууллагын удирдагч боллоо",
		Body:  "Дээд цэсний «" + org.DisplayName + "» товчоор байгууллагын профайл руу шилжиж сургалт, багш, хөтөлбөрөө удирдана.", Link: "/"})
}

// handleAdminCreateOrg: POST /api/admin/orgs {name, username, commission_pct, storage_mb, manager?, email?, password?} —
// байгууллагын тусдаа профайл үүсгэнэ. Удирдагч (байгаа хэрэглэгч) өөрийн бүртгэлээр нэвтэрч шилжинэ; имэйл, нууц үг
// өгвөл байгууллага өөрөө ч нэвтэрнэ. Аль нэг нь заавал.
func (s *Server) handleAdminCreateOrg(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		orgTerms
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Manager  string `json:"manager"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	msg := in.validate()
	if msg == "" {
		msg = validOrgName(in.Name)
	}
	switch {
	case msg != "":
	case !usernameRe.MatchString(in.Username):
		msg = "профайлын нэр: 3-32 тэмдэгт, a-z 0-9 _"
	case in.Manager == "" && in.Email == "":
		msg = "удирдах хүнээ сонгоно уу (эсвэл байгууллагын нэвтрэх имэйл, нууц үг өгнө)"
	case in.Email != "" && func() bool { _, err := mail.ParseAddress(in.Email); return err != nil }():
		msg = "имэйл буруу"
	case in.Email != "" && (len(in.Password) < minPassword || len(in.Password) > 72):
		msg = "нууц үг 6-72 тэмдэгт"
	}
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	var manager *store.User
	if in.Manager != "" {
		if manager, ok = s.managerCandidate(w, r, in.Manager); !ok {
			return
		}
	}
	u := &store.User{Username: in.Username, Email: in.Email, Role: store.RoleTeacher, DisplayName: in.Name,
		Org: &store.Org{Pct: in.Pct, StorageMB: in.StorageMB, Programs: []store.Program{}}}
	if manager != nil {
		u.Org.Managers = []string{manager.ID}
	}
	if in.Email == "" {
		u.Email = in.Username + orgEmailDomain // нууц үггүй — зөвхөн удирдагчаар нэвтэрнэ
	} else {
		hash, err := auth.HashPassword(ctx, in.Password)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, "түр ачаалалтай байна")
			return
		}
		u.PasswordHash = hash
	}
	consentNow(u) // нөхцөлийг удирдагч өөрийн бүртгэлээр зөвшөөрсөн
	if err := s.store.CreateUser(ctx, u); errors.Is(err, store.ErrConflict) {
		writeErr(w, http.StatusConflict, "энэ профайлын нэр эсвэл имэйл бүртгэлтэй байна")
		return
	} else if s.storeErr(w, r, err) {
		return
	}
	s.onUserCreated(u)
	s.planEnds.Delete(u.ID)
	s.profiles.Delete(u.Username)
	s.adminInvalidate()
	if manager != nil {
		s.notifyManager(ctx, manager.ID, u)
	}
	s.log.Info("хянагч байгууллага үүсгэв", "org", u.ID, "admin", admin.ID, "pct", in.Pct, "storage_mb", in.StorageMB, "manager", in.Manager)
	s.adminOrgReply(w, r, http.StatusCreated, u)
}

// handleAdminUpdateOrg: PUT /api/admin/orgs/{id} {name?, username?, commission_pct, storage_mb} — нэр, профайлын
// холбоос, гэрээний нөхцөл.
func (s *Server) handleAdminUpdateOrg(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		orgTerms
		Name     *string `json:"name"`
		Username *string `json:"username"`
	}
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	cur, err := s.store.UserByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if cur.Org == nil {
		writeErr(w, http.StatusNotFound, "байгууллага олдсонгүй")
		return
	}
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if msg := validOrgName(*in.Name); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
	}
	if in.Username != nil {
		un := strings.ToLower(strings.TrimSpace(*in.Username))
		if !usernameRe.MatchString(un) {
			writeErr(w, http.StatusBadRequest, "профайлын нэр: 3-32 тэмдэгт, a-z 0-9 _")
			return
		}
		if un != cur.Username {
			if err := s.store.UpdateUsername(ctx, cur.ID, un); errors.Is(err, store.ErrConflict) {
				writeErr(w, http.StatusConflict, "энэ профайлын нэр бүртгэлтэй байна")
				return
			} else if s.storeErr(w, r, err) {
				return
			}
			s.profiles.Delete(cur.Username)
		}
	}
	u, err := s.store.UpdateUser(ctx, cur.ID, func(x *store.User) error {
		if x.Org == nil {
			return store.ErrNotFound
		}
		x.Org.Pct, x.Org.StorageMB = in.Pct, in.StorageMB
		if in.Name != nil {
			x.DisplayName = *in.Name
		}
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	s.planEnds.Delete(u.ID)
	s.hub.SendUsers([]string{u.ID}, map[string]any{"type": "plan_changed"}) // гэрээт ↔ багцтай болбол студи шинэчлэгдэнэ
	s.invalidateTeacher(r, u.ID, u.Username)
	for _, m := range u.Org.Managers {
		s.managed.Delete(m)
	}
	s.adminInvalidate()
	s.log.Info("хянагч байгууллагыг засав", "org", u.ID, "admin", admin.ID, "pct", in.Pct, "storage_mb", in.StorageMB)
	s.adminOrgReply(w, r, http.StatusOK, u)
}

// handleAdminOrgManager: POST /api/admin/orgs/{id}/managers {user_id} — удирдагч нэмнэ;
// DELETE /api/admin/orgs/{id}/managers/{uid} — хасна (байгууллагын нээлттэй сессүүд хүчингүй болно).
func (s *Server) handleAdminOrgManager(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	add := r.Method == http.MethodPost
	uid := r.PathValue("uid")
	if add {
		var in struct {
			UserID string `json:"user_id"`
		}
		if !decode(w, r, &in) {
			return
		}
		m, ok := s.managerCandidate(w, r, in.UserID)
		if !ok {
			return
		}
		uid = m.ID
	}
	errFull, errHas := errors.New("full"), errors.New("has")
	u, err := s.store.UpdateUser(ctx, r.PathValue("id"), func(x *store.User) error {
		if x.Org == nil {
			return store.ErrNotFound
		}
		has := slices.Contains(x.Org.Managers, uid)
		switch {
		case add && has:
			return errHas
		case add && len(x.Org.Managers) >= maxOrgManagers:
			return errFull
		case add:
			x.Org.Managers = append(x.Org.Managers, uid)
		case !has:
			return store.ErrNotFound
		default:
			x.Org.Managers = slices.DeleteFunc(x.Org.Managers, func(v string) bool { return v == uid })
			now := time.Now()
			x.TokensAfter = &now // хассан удирдагчийн шилжиж авсан эрх (байгууллагын токен) шууд хүчингүй
		}
		return nil
	})
	switch {
	case errors.Is(err, errHas):
		writeErr(w, http.StatusConflict, "энэ хүн аль хэдийн удирдагч")
		return
	case errors.Is(err, errFull):
		writeErr(w, http.StatusBadRequest, "нэг байгууллагад 10 хүртэл удирдагч")
		return
	case s.storeErr(w, r, err):
		return
	}
	s.alive.Delete(u.ID)
	s.managed.Delete(uid)
	if add {
		s.notifyManager(ctx, uid, u)
	} else {
		s.notify(ctx, &store.Notification{UserID: uid, Type: NotifAccount, Count: 1, ActorID: u.ID, Title: "«" + u.DisplayName + "» байгууллагын удирдах эрх хаагдлаа"})
	}
	s.log.Info("хянагч байгууллагын удирдагч өөрчлөв", "org", u.ID, "admin", admin.ID, "user", uid, "add", add)
	s.adminOrgReply(w, r, http.StatusOK, u)
}

// handleAdminDeleteOrg: DELETE /api/admin/orgs/{id} — байгууллагын бүртгэл, профайлыг устгана. Сургалт, номтой бол
// устгахгүй (суралцагчдын төлбөр, хандалт хамааралтай) — эхлээд сургалтуудаа устгана.
func (s *Server) handleAdminDeleteOrg(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, err := s.store.UserByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if u.Org == nil {
		writeErr(w, http.StatusNotFound, "байгууллага олдсонгүй")
		return
	}
	blocked, err := s.eraseAccount(ctx, u)
	if s.storeErr(w, r, err) {
		return
	}
	if blocked != "" {
		writeErr(w, http.StatusConflict, "Сургалт, номтой байгууллагыг устгахгүй — суралцагчдын төлбөр, хандалт хамааралтай. Эхлээд байгууллагын сургалтуудыг устгана уу (суралцагчтай бол ноорог болгож нууна).")
		return
	}
	for _, m := range u.Org.Managers {
		s.managed.Delete(m)
	}
	s.adminInvalidate()
	s.log.Info("хянагч байгууллагыг устгав", "org", u.ID, "admin", admin.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Удирдагч: байгууллага руу шилжих ----

// managedOrgs — хэрэглэгчийн удирддаг байгууллагууд (2 минут кэштэй; удирдагч өөрчлөгдөхөд цэвэрлэнэ).
func (s *Server) managedOrgs(ctx context.Context, uid string) []TeacherRef {
	refs, err := s.managed.GetOrLoad(ctx, uid, func(ctx context.Context) ([]TeacherRef, error) {
		us, err := s.store.OrgsManagedBy(ctx, uid)
		if err != nil {
			return nil, err
		}
		out := make([]TeacherRef, 0, len(us))
		for i := range us {
			out = append(out, refOf(&us[i]))
		}
		return out, nil
	})
	if err != nil {
		return nil
	}
	return refs
}

// handleMyOrgs: GET /api/me/orgs — миний удирддаг байгууллагууд.
func (s *Server) handleMyOrgs(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.managedOrgs(r.Context(), c.UID))
}

// handleSwitchOrg: POST /api/me/orgs/{id}/switch — удирдагч байгууллагын профайл руу шилжинэ: байгууллагын нэрийн
// өмнөөс ажиллах токен авна (клиент хувийн токеноо хадгалж «буцах»-д ашиглана).
func (s *Server) handleSwitchOrg(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	org, err := s.store.UserByID(r.Context(), r.PathValue("id"))
	if err != nil || org.Org == nil || !slices.Contains(org.Org.Managers, c.UID) {
		writeErr(w, http.StatusNotFound, "байгууллага олдсонгүй")
		return
	}
	s.log.Info("удирдагч байгууллага руу шилжив", "user", c.UID, "org", org.ID)
	s.issue(w, http.StatusOK, org)
}
