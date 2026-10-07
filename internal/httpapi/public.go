package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"surgalt/internal/store"
)

// Rendered нь кэшлэгдсэн харагдац: JSON болон HTML-ийг урьдчилан сериалчилсан
// тул халуун зам дээр зөвхөн байт бичнэ (JSON marshal, template execute хийхгүй).
type Rendered[T any] struct {
	Data T
	JSON []byte
	HTML []byte
	ETag string
}

type PublicTeacher struct {
	ID              string       `json:"id"`
	Username        string       `json:"username"`
	DisplayName     string       `json:"display_name"`
	Headline        string       `json:"headline"`
	Bio             string       `json:"bio"`
	AvatarURL       string       `json:"avatar_url"`
	CoverURL        string       `json:"cover_url"`
	Subjects        []string     `json:"subjects"`
	Location        string       `json:"location"`
	Links           []PublicLink `json:"links"`
	JoinedYear      int          `json:"joined_year"`
	StudentCount    int64        `json:"student_count"`
	CourseCount     int          `json:"course_count"`
	LessonCount     int          `json:"lesson_count"`
	FreeLessonCount int          `json:"free_lesson_count"`
}

// PublicLink нь профайл дээрх гадаад холбоос (store.LinkKeys-ийн дарааллаар).
type PublicLink struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

// PublicMeeting нь товлосон шууд хичээлийн НЭЭЛТТЭЙ хэсэг. Meet холбоос энд хэзээ ч орохгүй:
// энэ өгөгдөл кэшлэгдэж, нэвтрээгүй хүнд ч очдог.
type PublicMeeting struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Price       int64     `json:"price,omitempty"` // төлбөртэй шууд хичээл (₮)
	CourseID    string    `json:"course_id"`
	CourseTitle string    `json:"course_title"`
	StartsAt    time.Time `json:"starts_at"`
	DurationMin int       `json:"duration_min"`
}

type PublicProfile struct {
	Teacher PublicTeacher  `json:"teacher"`
	Courses []store.Course `json:"courses"`
	// TopCourseID — хамгийн их үзэлттэй сургалт (2+ сургалттай, үзэлттэй үед).
	TopCourseID string          `json:"top_course_id,omitempty"`
	Meetings    []PublicMeeting `json:"meetings"`
	Books       []PublicBook    `json:"books"`
}

// PublicLesson: үнэгүй хичээлийн агуулга нээлттэй, төлбөртэйн зөвхөн гарчиг.
type PublicLesson struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	IsFree   bool   `json:"is_free"`
	Price    int64  `json:"price"`
	Position int    `json:"position"`
	// Дараалсан нээлтийн тохиргоо (харуулахад): өмнөхийн дараа хэдэн цаг хүлээх, дарааллаас гадуур эсэх.
	UnlockAfterH int               `json:"unlock_after_h"`
	AlwaysOpen   bool              `json:"always_open"`
	Format       string            `json:"format,omitempty"`
	Mode         string            `json:"mode,omitempty"`
	Section      string            `json:"section,omitempty"`
	Content      string            `json:"content,omitempty"`
	Blocks       []store.Block     `json:"blocks,omitempty"`
	ActiveMin    int               `json:"active_min,omitempty"`
	Exam         *store.Exam       `json:"exam,omitempty"`
	Assignment   *store.Assignment `json:"assignment,omitempty"`
	Discussion   bool              `json:"discussion"`
	UnlockRule   string            `json:"unlock_rule,omitempty"`
	VideoURL     string            `json:"video_url,omitempty"`
}

// PublicSection — хөтөлбөрийн нэг бүлэг (модуль). Бүлэггүй хичээлүүд хоосон гарчигтай бүлэгт орно.
type PublicSection struct {
	Title   string         `json:"title"`
	Lessons []PublicLesson `json:"lessons"`
}

type PublicCourse struct {
	Course   store.Course    `json:"course"`
	Teacher  PublicTeacher   `json:"teacher"`
	Lessons  []PublicLesson  `json:"lessons"`
	Sections []PublicSection `json:"sections"`
	// Grouped: дор хаяж нэг хичээл бүлэгтэй — хөтөлбөрийг бүлгээр харуулна.
	Grouped bool `json:"grouped"`
}

// groupLessons нь хичээлүүдийг дарааллаар нь явж, бүлгийн нэрээр (анх таарсан дарааллаар) нэгтгэнэ.
func groupLessons(ls []PublicLesson) ([]PublicSection, bool) {
	var out []PublicSection
	idx := map[string]int{}
	grouped := false
	for _, l := range ls {
		if l.Section != "" {
			grouped = true
		}
		i, ok := idx[l.Section]
		if !ok {
			i = len(out)
			idx[l.Section] = i
			out = append(out, PublicSection{Title: l.Section, Lessons: []PublicLesson{}})
		}
		out[i].Lessons = append(out[i].Lessons, l)
	}
	if out == nil {
		out = []PublicSection{}
	}
	return out, grouped
}

func publicTeacher(u *store.User) PublicTeacher {
	t := PublicTeacher{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Headline: u.Headline, Bio: u.Bio, AvatarURL: u.AvatarURL, CoverURL: u.CoverURL,
		Subjects: u.Subjects, Location: u.Location, Links: []PublicLink{}, JoinedYear: u.CreatedAt.Year()}
	if t.Subjects == nil {
		t.Subjects = []string{}
	}
	for _, k := range store.LinkKeys {
		if v := u.Links[k]; v != "" {
			t.Links = append(t.Links, PublicLink{Key: k, Label: linkLabels[k], URL: v})
		}
	}
	return t
}

const profileMaxMeetings = 5

// publicMeetings: зөвхөн нийтлэгдсэн сургалтад хамаарах шууд хичээлүүд. Чатаас үүсгэсэн
// ганцаарчилсан уулзалт (сургалтгүй) нь зочны нэрийг агуулдаг тул нээлттэй гаргахгүй.
func (s *Server) publicMeetings(ctx context.Context, courses []store.Course) ([]PublicMeeting, error) {
	out := []PublicMeeting{}
	if len(courses) == 0 {
		return out, nil
	}
	ids := make([]string, len(courses))
	titles := make(map[string]string, len(courses))
	for i, c := range courses {
		ids[i], titles[c.ID] = c.ID, c.Title
	}
	ms, err := s.store.MeetingsByCourses(ctx, ids, time.Now(), profileMaxMeetings)
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		out = append(out, PublicMeeting{ID: m.ID, Title: m.Title, Price: m.Price, CourseID: m.CourseID, CourseTitle: titles[m.CourseID], StartsAt: m.StartsAt, DurationMin: m.DurationMin})
	}
	return out, nil
}

func topCourse(courses []store.Course) string {
	if len(courses) < 2 {
		return ""
	}
	best := courses[0]
	for _, c := range courses[1:] {
		if c.Views > best.Views {
			best = c
		}
	}
	if best.Views == 0 {
		return ""
	}
	return best.ID
}

func render[T any](s *Server, data T, page string) (*Rendered[T], error) {
	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(j)
	out := &Rendered[T]{Data: data, JSON: j, ETag: `"` + hex.EncodeToString(sum[:8]) + `"`}
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, page, map[string]any{"D": data, "PublicURL": s.cfg.PublicURL}); err != nil {
		return nil, err
	}
	out.HTML = buf.Bytes()
	return out, nil
}

func (s *Server) publicProfile(ctx context.Context, username string) (*Rendered[PublicProfile], error) {
	return s.profiles.GetOrLoad(ctx, username, func(ctx context.Context) (*Rendered[PublicProfile], error) {
		u, err := s.store.UserByUsername(ctx, username)
		if err != nil {
			return nil, err
		}
		if u.Role != store.RoleTeacher {
			return nil, store.ErrNotFound
		}
		courses, err := s.store.CoursesByTeacher(ctx, u.ID, true)
		if err != nil {
			return nil, err
		}
		students, err := s.store.TeacherStudentCount(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		t := publicTeacher(u)
		t.StudentCount, t.CourseCount = students, len(courses)
		for _, c := range courses {
			t.LessonCount += c.LessonCount
			t.FreeLessonCount += c.FreeLessonCount
		}
		meetings, err := s.publicMeetings(ctx, courses)
		if err != nil {
			return nil, err
		}
		books := []PublicBook{}
		if bs, err := s.store.BooksByTeacher(ctx, t.ID, true); err == nil {
			for i := range bs {
				books = append(books, publicBook(&bs[i], false))
			}
		}
		return render(s, PublicProfile{Teacher: t, Courses: courses, TopCourseID: topCourse(courses), Meetings: meetings, Books: books}, "profile.html")
	})
}

func (s *Server) publicCourse(ctx context.Context, id string) (*Rendered[PublicCourse], error) {
	return s.courses.GetOrLoad(ctx, id, func(ctx context.Context) (*Rendered[PublicCourse], error) {
		c, err := s.store.CourseByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !c.Published {
			return nil, store.ErrNotFound
		}
		u, err := s.store.UserByID(ctx, c.TeacherID)
		if err != nil {
			return nil, err
		}
		lessons, err := s.store.LessonsByCourse(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		pls := make([]PublicLesson, len(lessons))
		for i, l := range lessons {
			pls[i] = PublicLesson{ID: l.ID, Title: l.Title, IsFree: l.IsFree, Price: l.Price, Position: l.Position, UnlockAfterH: l.UnlockAfterH, AlwaysOpen: l.AlwaysOpen, Format: l.Format, Mode: l.Mode, Section: l.Section, ActiveMin: l.ActiveMin, Exam: l.Exam, Assignment: l.Assignment, Discussion: l.Discussion, UnlockRule: l.UnlockRule}
			if l.IsFree {
				// Кэш 30 секунд тул 6 цагийн гарын үсэг үргэлж хүчинтэй байна.
				pls[i].Content, pls[i].VideoURL, pls[i].Blocks = l.Content, s.viewerMedia(l.VideoURL, ""), s.viewerBlocks(examIntro(&l))
			}
		}
		secs, grouped := groupLessons(pls)
		return render(s, PublicCourse{Course: *c, Teacher: publicTeacher(u), Lessons: pls, Sections: secs, Grouped: grouped}, "course.html")
	})
}

// serveRendered нь ETag-аар 304 буцааж, CDN/браузерт богино кэш зөвшөөрнө.
func serveRendered(w http.ResponseWriter, r *http.Request, etag string, body []byte, ctype string) {
	h := w.Header()
	h.Set("ETag", etag)
	h.Set("Cache-Control", "public, max-age=10, stale-while-revalidate=30")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", ctype)
	h.Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}

func (s *Server) handlePublicProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.publicProfile(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	serveRendered(w, r, p.ETag, p.JSON, "application/json; charset=utf-8")
}

func (s *Server) handlePublicCourse(w http.ResponseWriter, r *http.Request) {
	c, err := s.publicCourse(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	serveRendered(w, r, c.ETag, c.JSON, "application/json; charset=utf-8")
}

// handleQR нь профайлын холбоосыг QR PNG болгож өгнө (?size=256..1024, ?download=1).
func (s *Server) handleQR(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if _, err := s.publicProfile(r.Context(), username); s.storeErr(w, r, err) {
		return
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	size = min(max(size, 256), 1024)
	url := s.baseURL(r) + "/t/" + username
	png, err := s.qrs.GetOrLoad(r.Context(), url+"|"+strconv.Itoa(size), func(context.Context) ([]byte, error) {
		return qrcode.Encode(url, qrcode.High, size)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "QR үүсгэж чадсангүй")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+username+`-qr.png"`)
	}
	_, _ = w.Write(png)
}

// ---- HTML хуудсууд ----

func (s *Server) pageProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.publicProfile(r.Context(), r.PathValue("username"))
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	serveRendered(w, r, p.ETag, p.HTML, "text/html; charset=utf-8") // үзэлтийг хөтөч /api/views-ээр мэдээлнэ
}

func (s *Server) pageCourse(w http.ResponseWriter, r *http.Request) {
	c, err := s.publicCourse(r.Context(), r.PathValue("id"))
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	serveRendered(w, r, c.ETag, c.HTML, "text/html; charset=utf-8")
}

// handleTrackView: POST /api/views {kind: profile|course, id} — профайл, сургалтын хуудас нээгдэхэд хөтөч
// (нэвтэрсэн бол токентой) мэдээлнэ. HTML хуудас токенгүй ирдэг тул үзэгчийг энд л танина: багш өөрийнхөө
// профайл, сургалтыг үзвэл тоолохгүй, мэдэгдэл явуулахгүй.
func (s *Server) handleTrackView(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	if !decode(w, r, &in) {
		return
	}
	uid := ""
	if p, ok := s.principal(r); ok && !p.IsGuest() {
		uid = p.UID
	}
	switch in.Kind {
	case "profile":
		if p, err := s.publicProfile(r.Context(), in.ID); err == nil && p.Data.Teacher.ID != uid {
			t := p.Data.Teacher
			s.trackView(r, t.ID, NotifProfileView, t.ID, "", "/t/"+t.Username)
		}
	case "course":
		if c, err := s.publicCourse(r.Context(), in.ID); err == nil && c.Data.Course.TeacherID != uid {
			s.trackView(r, c.Data.Course.TeacherID, NotifCourseView, c.Data.Course.ID, c.Data.Course.Title, "/c/"+c.Data.Course.ID)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pageError(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := http.StatusInternalServerError, "Алдаа гарлаа"
	if errors.Is(err, store.ErrNotFound) {
		status, msg = http.StatusNotFound, "Хуудас олдсонгүй"
	} else {
		s.log.Error("page", "err", err, "path", r.URL.Path)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.tmpl.ExecuteTemplate(w, "error.html", map[string]any{"Status": status, "Message": msg})
}

func (s *Server) pageHome(w http.ResponseWriter, r *http.Request) {
	s.pageStatic("home")(w, r)
}

func (s *Server) pageStatic(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		if err := s.tmpl.ExecuteTemplate(w, name+".html", map[string]any{"PublicURL": s.cfg.PublicURL}); err != nil {
			s.log.Error("template", "err", err, "page", name)
		}
	}
}
