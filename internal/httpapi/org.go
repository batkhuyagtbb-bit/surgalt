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

type adminOrgRow struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	Pct       float64   `json:"commission_pct"`
	StorageMB int64     `json:"storage_mb"`
	UsedBytes int64     `json:"used_bytes"`
	Programs  int       `json:"programs"`
	Courses   int       `json:"courses"`
	Published int       `json:"courses_published"`
	Teachers  int       `json:"teachers"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) adminOrgRow(u *store.User, courses []store.Course) adminOrgRow {
	row := adminOrgRow{ID: u.ID, Username: u.Username, Name: u.DisplayName, Email: u.Email, AvatarURL: u.AvatarURL,
		Pct: u.Org.Pct, StorageMB: u.Org.StorageMB, Programs: len(u.Org.Programs), CreatedAt: u.CreatedAt}
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
	return row
}

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
			rows = append(rows, s.adminOrgRow(&users[i], courses))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"orgs": rows, "default_pct": store.TeacherCommission(), "free_mb": s.cfg.StorageFreeMB})
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

var errAlreadyOrg = errors.New("аль хэдийн байгууллага")

// handleAdminCreateOrg: POST /api/admin/orgs — байгууллагын бүртгэл үүсгэнэ:
//   - шинэ: {name, username, email, password, commission_pct, storage_mb};
//   - байгаа бүртгэлийг байгууллага болгох: {existing: "@нэр эсвэл имэйл", commission_pct, storage_mb}.
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
		Existing string `json:"existing"`
	}
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	org := &store.Org{Pct: in.Pct, StorageMB: in.StorageMB, Programs: []store.Program{}}
	var u *store.User
	if strings.TrimSpace(in.Existing) != "" {
		cur, err := s.resolveTeacher(ctx, in.Existing)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "ийм бүртгэл олдсонгүй")
			return
		}
		if s.storeErr(w, r, err) {
			return
		}
		u, err = s.store.UpdateUser(ctx, cur.ID, func(x *store.User) error {
			if x.Org != nil {
				return errAlreadyOrg
			}
			x.Role, x.Org = store.RoleTeacher, org
			return nil
		})
		if errors.Is(err, errAlreadyOrg) {
			writeErr(w, http.StatusConflict, "энэ бүртгэл аль хэдийн байгууллага")
			return
		}
		if s.storeErr(w, r, err) {
			return
		}
	} else {
		in.Name = strings.TrimSpace(in.Name)
		in.Username = strings.ToLower(strings.TrimSpace(in.Username))
		in.Email = strings.ToLower(strings.TrimSpace(in.Email))
		switch {
		case in.Name == "" || utf8.RuneCountInString(in.Name) > 80:
			writeErr(w, http.StatusBadRequest, "байгууллагын нэр 1-80 тэмдэгт")
			return
		case !usernameRe.MatchString(in.Username):
			writeErr(w, http.StatusBadRequest, "профайлын нэр: 3-32 тэмдэгт, a-z 0-9 _")
			return
		case func() bool { _, err := mail.ParseAddress(in.Email); return err != nil }():
			writeErr(w, http.StatusBadRequest, "имэйл буруу")
			return
		case len(in.Password) < minPassword || len(in.Password) > 72:
			writeErr(w, http.StatusBadRequest, "нууц үг 6-72 тэмдэгт")
			return
		}
		hash, err := auth.HashPassword(ctx, in.Password)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, "түр ачаалалтай байна")
			return
		}
		u = &store.User{Username: in.Username, Email: in.Email, PasswordHash: hash, Role: store.RoleTeacher, DisplayName: in.Name, Org: org}
		if err := s.store.CreateUser(ctx, u); errors.Is(err, store.ErrConflict) {
			writeErr(w, http.StatusConflict, "энэ имэйл эсвэл профайлын нэр бүртгэлтэй байна")
			return
		} else if s.storeErr(w, r, err) {
			return
		}
	}
	s.onUserCreated(u)
	s.planEnds.Delete(u.ID)
	s.profiles.Delete(u.Username)
	s.adminInvalidate()
	s.log.Info("хянагч байгууллага үүсгэв", "org", u.ID, "admin", admin.ID, "pct", in.Pct, "storage_mb", in.StorageMB)
	courses, _ := s.store.CoursesByTeacher(ctx, u.ID, false)
	writeJSON(w, http.StatusCreated, s.adminOrgRow(u, courses))
}

// handleAdminUpdateOrg: PUT /api/admin/orgs/{id} {commission_pct, storage_mb} — гэрээний нөхцөл.
func (s *Server) handleAdminUpdateOrg(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in orgTerms
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	u, err := s.store.UpdateUser(ctx, r.PathValue("id"), func(x *store.User) error {
		if x.Org == nil {
			return store.ErrNotFound
		}
		x.Org.Pct, x.Org.StorageMB = in.Pct, in.StorageMB
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	s.planEnds.Delete(u.ID)
	s.adminInvalidate()
	s.log.Info("хянагч байгууллагын нөхцөл өөрчлөв", "org", u.ID, "admin", admin.ID, "pct", in.Pct, "storage_mb", in.StorageMB)
	courses, _ := s.store.CoursesByTeacher(ctx, u.ID, false)
	writeJSON(w, http.StatusOK, s.adminOrgRow(u, courses))
}
