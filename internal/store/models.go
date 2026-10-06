// Package store нь өгөгдлийн давхарга: загварууд болон Store интерфэйс.
// ID бүр 24 тэмдэгттэй hex мөр (цаг хугацаагаар эрэмбэлэгддэг, store.NewID).
package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("олдсонгүй")
	ErrConflict = errors.New("давхардсан эсвэл зөрчилтэй")
)

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email,omitempty"`
	PasswordHash string `json:"-"`
	Role         Role   `json:"role"`
	DisplayName  string `json:"display_name"`
	Headline     string `json:"headline"`
	Bio          string `json:"bio"`
	AvatarURL    string `json:"avatar_url"`
	CoverURL     string `json:"cover_url"` // профайлын дээд талын өргөн зураг
	// Нээлттэй профайлын нэмэлт мэдээлэл (багш).
	Subjects  []string          `json:"subjects"` // заадаг чиглэлүүд
	Location  string            `json:"location"` // хот / байршил
	Links     map[string]string `json:"links"`    // LinkKeys-ийн аль нэг -> https холбоос
	CreatedAt time.Time         `json:"created_at"`
	// Худалдан авсан нэмэлт багтаамж (байт) ба дуусах хугацаа.
	StorageExtraBytes int64      `json:"storage_extra_bytes"`
	StorageExpiresAt  *time.Time `json:"storage_expires_at,omitempty"`
	// Google Meet: шифрлэгдсэн refresh token (JSON-д гарахгүй).
	GoogleToken   string `json:"-"`
	MeetConnected bool   `json:"meet_connected"`
	ProfileViews  int64  `json:"profile_views"`
}

// LinkKeys — профайл дээр зөвшөөрөгдсөн гадаад холбоосын төрлүүд (харуулах дарааллаар).
var LinkKeys = []string{"website", "facebook", "instagram", "youtube"}

// ProfileUpdate нь хэрэглэгч өөрөө засаж болох профайлын талбарууд.
type ProfileUpdate struct {
	DisplayName string
	Headline    string
	Bio         string
	AvatarURL   string
	CoverURL    string
	Subjects    []string
	Location    string
	Links       map[string]string
}

// Notification — хэрэглэгчид (ихэвчлэн багшид) очих мэдэгдэл.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // profile_view, course_view, lesson_view, enroll, purchase, message, storage
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	Count     int64     `json:"count"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// Meeting нь Google Meet-ээр явагдах шууд хичээл/уулзалт.
type Meeting struct {
	ID          string    `json:"id"`
	TeacherID   string    `json:"teacher_id"`
	CourseID    string    `json:"course_id,omitempty"`
	Title       string    `json:"title"`
	StartsAt    time.Time `json:"starts_at"`
	DurationMin int       `json:"duration_min"`
	MeetURL     string    `json:"meet_url,omitempty"`
	EventID     string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

// StorageQuota нь үнэгүй хэмжээ + хүчинтэй худалдан авсан багтаамж.
func (u *User) StorageQuota(freeBytes int64, now time.Time) int64 {
	if u.StorageExpiresAt != nil && now.Before(*u.StorageExpiresAt) {
		return freeBytes + u.StorageExtraBytes
	}
	return freeBytes
}

// Course бол багшийн явуулдаг сургалт. Price нь төгрөгөөр, 0 бол үнэгүй.
type Course struct {
	ID          string `json:"id"`
	TeacherID   string `json:"teacher_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Published   bool   `json:"published"`
	// Дараалсан нээлт (drip): хичээлүүд өмнөхөө үзсэний дараа л нээгдэнэ.
	Drip bool `json:"drip"`
	// UnlockAllPaid: багцын төлбөр төлсөн суралцагчид дарааллаас үл хамааран бүгд нээлттэй.
	UnlockAllPaid bool `json:"unlock_all_paid"`
	// Camera — камертай анхаарлын хяналт: off | optional | required.
	Camera string `json:"camera"`
	// Certificate — сургалтыг дүүргэсэн суралцагчид сертификат олгоно (хайлтад шошго болно).
	Certificate     bool      `json:"certificate"`
	LessonCount     int       `json:"lesson_count"`
	FreeLessonCount int       `json:"free_lesson_count"`
	Views           int64     `json:"views"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Lesson нь сургалтын нэг хичээл. IsFree=true бол хэн ч үзэж болно (preview).
type Lesson struct {
	ID       string `json:"id"`
	CourseID string `json:"course_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	VideoURL string `json:"video_url"`
	IsFree   bool   `json:"is_free"`
	// Price — төлбөртэй хичээлийг дангаар нь худалдах үнэ (₮).
	// 0 бол зөвхөн сургалтын багцаар нээгдэнэ.
	Price int64 `json:"price"`
	// Дараалсан нээлтийн тохиргоо: өмнөх хичээлийг үзснээс хойш хэдэн цагийн дараа нээгдэх (0 = шууд).
	UnlockAfterH int `json:"unlock_after_h"`
	// AlwaysOpen: дарааллаас үл хамааран нээлттэй (төлбөрийн шалгалт хэвээр).
	AlwaysOpen bool `json:"always_open"`
	// Format — сургалтын хэлбэр (LessonFormats), Mode — заах аргын төрөл (LessonModes). Хоосон бол заагаагүй.
	Format string `json:"format"`
	Mode   string `json:"mode"`
	// Section — хичээлийн бүлэг (модуль): хөтөлбөр дээр ижил нэртэй хичээлүүд нэг бүлэгт харагдана. Хоосон = бүлэггүй.
	Section string `json:"section"`
	// Blocks — хичээлийн дэлгэрэнгүй агуулга: текст, гарчиг, зураг, дуу, видео, файл, асуулт дарааллаараа.
	Blocks []Block `json:"blocks"`
	// ActiveMin — суралцагч идэвхтэй үзэх ёстой хамгийн бага хугацаа (минут). 0 = шаардлагагүй.
	ActiveMin int `json:"active_min"`
	// Exam — хоосон биш бол энэ хичээл шалгалт (хугацаатай, хамгаалалт хатуу).
	Exam *Exam `json:"exam,omitempty"`
	// Assignment — хоосон биш бол энэ хичээл даалгавар (хугацаатай, хариу илгээж дүгнүүлнэ).
	Assignment *Assignment `json:"assignment,omitempty"`
	Position   int         `json:"position"`
	CreatedAt  time.Time   `json:"created_at"`
}

// Block — хичээлийн агуулгын нэг хэсэг. Text нь text төрөлд хэлбэржүүлсэн текст (аюулгүй
// энгийн тэмдэглэгээ: **тод**, *налуу*, __доогуур__, ==тодруулга==, [холбоос](https://…), "- " жагсаалт, "> " ишлэл),
// heading-д гарчиг, медиа төрөлд тайлбар.
type Block struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size,omitempty"`
	Quiz *Quiz  `json:"quiz,omitempty"`
	// Download — файлын хэсэгт: суралцагч татаж авахыг багш зөвшөөрсөн эсэх (анхдагч: зөвхөн үзнэ).
	Download bool `json:"download,omitempty"`
}

// Quiz — нэг асуулт. Зөв хариулт (Correct, Answers, Spot, Right-ийн дараалал) ба Explain нь
// суралцагчид хариулахаас өмнө илгээгдэхгүй.
type Quiz struct {
	// Kind: single (нэг сонголт), multi (олон сонголт), text (бичгээр), match (харгалзуулах), image (зурган дээр заах).
	Kind     string   `json:"kind,omitempty"`
	Question string   `json:"question"`
	Image    string   `json:"image,omitempty"` // асуултын зураг; image төрөлд заавал
	Options  []string `json:"options,omitempty"`
	Correct  []int    `json:"correct,omitempty"`
	Multi    bool     `json:"multi,omitempty"`
	Answers  []string `json:"answers,omitempty"` // text: зөвд тооцох хариултууд
	Left     []string `json:"left,omitempty"`    // match: зүүн багана
	Right    []string `json:"right,omitempty"`   // match: Right[i] нь Left[i]-ийн хос
	Spot     *Spot    `json:"spot,omitempty"`    // image: зөв хэсэг (хувиар)
	Points   int      `json:"points,omitempty"`  // шалгалтын оноо (0 = 1)
	Explain  string   `json:"explain,omitempty"`
}

// Spot — зураг дээрх зөв хэсэг: төв (X, Y) ба радиус R, зургийн өргөн/өндрийн хувиар.
type Spot struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r"`
}

// QuizKind нь хуучин (Kind хоосон) асуултыг Multi-аар нь ангилна.
func (q *Quiz) QuizKind() string {
	if q.Kind != "" {
		return q.Kind
	}
	if q.Multi {
		return "multi"
	}
	return "single"
}

// Exam — хичээлийг шалгалт болгоно: асуулт блокууд нь шалгалтын даалгавар.
type Exam struct {
	TimeMin     int  `json:"time_min"`     // хугацаа (минут), 0 = хязгааргүй
	Attempts    int  `json:"attempts"`     // оролдлогын тоо, 0 = хязгааргүй
	PassPct     int  `json:"pass_pct"`     // тэнцэх хувь
	Shuffle     bool `json:"shuffle"`      // асуултын дарааллыг холих
	ShowAnswers bool `json:"show_answers"` // дууссаны дараа зөв хариултыг харуулах
	Due         Due  `json:"due"`          // дуусах хугацаа ба хоцорсон тохиолдлын бодлого
}

// Due — шалгалт, даалгаврын дуусах хугацаа. At хоосон бол хугацаагүй. Хугацаа өнгөрсний дараа:
// Late = "free" (төлбөргүй үргэлжлүүлнэ), "paid" (LateFee төлж нээнэ), "closed" (хаалттай).
type Due struct {
	At      *time.Time `json:"at,omitempty"`
	Late    string     `json:"late,omitempty"`
	LateFee int64      `json:"late_fee,omitempty"`
}

// LatePassKey — хоцорсон шалгалт/даалгаврын төлбөр төлсөн тэмдэг (LessonProgress.Quiz дотор).
const LatePassKey = "late_pass"

// LatePolicies — хоцорсон тохиолдлын бодлогын түлхүүр → монгол нэр.
var LatePolicies = map[string]string{"": "төлбөргүй", "free": "төлбөргүй", "paid": "төлбөртэй", "closed": "хаалттай"}

// Assignment — даалгавар: суралцагч текст, файлаар хариугаа илгээж, багш оноо, тайлбар өгнө.
type Assignment struct {
	Due        Due  `json:"due"`
	MaxScore   int  `json:"max_score"`   // дээд оноо (анхдагч 100)
	AllowFiles bool `json:"allow_files"` // файл хавсаргахыг зөвшөөрөх
}

// Submission — нэг суралцагчийн нэг даалгаварт илгээсэн хариу (дахин илгээвэл шинэчлэгдэнэ).
type Submission struct {
	ID          string     `json:"id"` // user:lesson
	UserID      string     `json:"user_id"`
	UserName    string     `json:"user_name"`
	CourseID    string     `json:"course_id"`
	LessonID    string     `json:"lesson_id"`
	TeacherID   string     `json:"teacher_id"`
	Text        string     `json:"text"`
	Files       []string   `json:"files"` // багшийн сан дахь /files/... замууд
	SubmittedAt time.Time  `json:"submitted_at"`
	Late        bool       `json:"late"`
	Score       *int       `json:"score,omitempty"`
	Feedback    string     `json:"feedback,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
}

// BlockTypes — зөвшөөрөгдсөн блокийн төрлүүд.
var BlockTypes = map[string]bool{"text": true, "heading": true, "image": true, "audio": true, "video": true, "file": true, "quiz": true}

// LessonOrder — чирж зөөсний дараах нэг хичээлийн байр: ID ба бүлэг.
type LessonOrder struct {
	ID      string `json:"id"`
	Section string `json:"section"`
}

// Хичээлийн хэлбэр ба заах аргын төрөл: түлхүүр → монгол нэр.
var LessonFormats = map[string]string{"lecture": "Лекц", "seminar": "Семинар", "practice": "Дадлага", "lab": "Лаборатори"}
var LessonModes = map[string]string{"classroom": "Танхимын", "online": "Цахим", "blended": "Холимог"}

// LessonProgress нь суралцагчийн нэг хичээл дээрх явц: анх үзсэн ба дууссан цаг.
type LessonProgress struct {
	LessonID    string     `json:"lesson_id"`
	ViewedAt    time.Time  `json:"viewed_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	// Quiz — асуулт (блокийн ID) → сүүлийн хариулт зөв эсэх.
	Quiz map[string]bool `json:"quiz,omitempty"`
	// QuizDoneAt — хичээлийн бүх асуултад зөв хариулж дуусгасан мөч (дараагийн хичээлийн таймер эндээс эхэлнэ).
	QuizDoneAt *time.Time `json:"quiz_done_at,omitempty"`
}

type OrderStatus string

const (
	OrderPending OrderStatus = "pending"
	OrderPaid    OrderStatus = "paid"
)

const (
	OrderKindCourse  = "course"  // сургалтын багц (бүх хичээл)
	OrderKindLesson  = "lesson"  // нэг хичээл
	OrderKindStorage = "storage" // файлын сангийн багтаамж
	OrderKindLate    = "late"    // хугацаа хоцорсон шалгалт/даалгаврын төлбөр (lesson_progress-д late_pass)
)

// StorageMonth — багтаамжийн нэг сарын үргэлжлэх хугацаа.
const StorageMonth = 30 * 24 * time.Hour

type Order struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	StorageMB int64       `json:"storage_mb,omitempty"`
	Months    int         `json:"months,omitempty"`
	UserID    string      `json:"user_id"`
	CourseID  string      `json:"course_id,omitempty"`
	LessonID  string      `json:"lesson_id,omitempty"`
	BookID    string      `json:"book_id,omitempty"`
	TeacherID string      `json:"teacher_id,omitempty"`
	Title     string      `json:"title,omitempty"`
	Amount    int64       `json:"amount"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	PaidAt    *time.Time  `json:"paid_at,omitempty"`
}

type Sale struct {
	OrderID       string    `json:"order_id"`
	CourseID      string    `json:"course_id"`
	CourseTitle   string    `json:"course_title"`
	BuyerUsername string    `json:"buyer_username"`
	Amount        int64     `json:"amount"`
	PaidAt        time.Time `json:"paid_at"`
}

// CourseStudent нь сургалтад элссэн эсвэл хичээл худалдаж авсан нэг суралцагч.
type CourseStudent struct {
	UserID     string     `json:"user_id"`
	EnrolledAt *time.Time `json:"enrolled_at,omitempty"`
	LessonIDs  []string   `json:"lesson_ids"`
}

type SalesSummary struct {
	TotalAmount int64  `json:"total_amount"`
	Count       int64  `json:"count"`
	Recent      []Sale `json:"recent"`
}

// Conversation нь багш ба зочин (нэвтэрсэн эсвэл нэвтрээгүй) хоорондын яриа,
// эсвэл сургалтын бүлэг чат (Kind == ConvGroup). VisitorKey: "u:<userID>", "g:<guestID>",
// бүлэгт "course:<courseID>" — ингэснээр нэг сургалтад нэг л бүлэг байна.
type Conversation struct {
	ID          string `json:"id"`
	TeacherID   string `json:"teacher_id"`
	VisitorKey  string `json:"-"`
	VisitorName string `json:"visitor_name"` // бүлэгт: бүлгийн нэр
	UserID      string `json:"user_id,omitempty"`
	// Бүлэг чат
	Kind          string    `json:"kind,omitempty"` // "" (хувийн) | ConvGroup
	CourseID      string    `json:"course_id,omitempty"`
	LastMessage   string    `json:"last_message"`
	CreatedAt     time.Time `json:"created_at"`
	LastMessageAt time.Time `json:"last_message_at"`
}

const (
	SenderTeacher = "teacher"
	SenderVisitor = "visitor"

	ConvGroup = "group"
)

// GroupVisitorKey нь сургалтын бүлэг чатын түлхүүр.
func GroupVisitorKey(courseID string) string { return "course:" + courseID }

type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	TeacherID      string `json:"-"`
	VisitorKey     string `json:"-"`
	Sender         string `json:"sender"`
	// Бүлэг чатад хэн бичсэнийг ялгана (хувийн яриад ч бөглөгдөнө).
	SenderID   string    `json:"sender_id,omitempty"`
	SenderName string    `json:"sender_name,omitempty"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store interface {
	LearningStore
	BookStore
	CreateUser(ctx context.Context, u *User) error
	UserByID(ctx context.Context, id string) (*User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	UserByUsername(ctx context.Context, username string) (*User, error)
	UpdateProfile(ctx context.Context, id string, p ProfileUpdate) error
	// UpdateUsername нь хэрэглэгчийн нэрийг (профайлын холбоос) солино. Нэр эзэнтэй бол ErrConflict.
	UpdateUsername(ctx context.Context, id, username string) error
	// UsersByIDs нь өгөгдсөн хэрэглэгчдийг буцаана (олдоогүйг алгасна, дараалал баталгаагүй).
	UsersByIDs(ctx context.Context, ids []string) ([]User, error)
	TeacherStudentCount(ctx context.Context, teacherID string) (int64, error)
	SetGoogleToken(ctx context.Context, userID, encrypted string) error
	// Үзэлтийн тоолуурууд (нэгтгэж, багцаар нэмэгдүүлнэ).
	IncViews(ctx context.Context, profile map[string]int64, course map[string]int64) error
	// Мэдэгдэл
	AddNotifications(ctx context.Context, ns []*Notification) error
	Notifications(ctx context.Context, userID string, limit int) ([]Notification, int64, error)
	MarkNotificationsRead(ctx context.Context, userID string) error
	CreateMeeting(ctx context.Context, m *Meeting) error
	// Meetings нь from-оос хойшхи уулзалтууд (courseID хоосон бол багшийн бүх).
	Meetings(ctx context.Context, teacherID, courseID string, from time.Time, limit int) ([]Meeting, error)
	// Гадаад нэвтрэлт (Google, Facebook ...): provider+subject -> хэрэглэгч.
	UserByIdentity(ctx context.Context, provider, subject string) (*User, error)
	LinkIdentity(ctx context.Context, userID, provider, subject string) error

	CreateCourse(ctx context.Context, c *Course) error
	// UpdateCourse нь зөвхөн c.TeacherID эзэмшдэг сургалтыг шинэчилнэ.
	UpdateCourse(ctx context.Context, c *Course) error
	CourseByID(ctx context.Context, id string) (*Course, error)
	CoursesByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Course, error)
	// PublishedCourses нь хайлтын индексэд: бүх нийтлэгдсэн сургалт (limit хүртэл).
	PublishedCourses(ctx context.Context, limit int) ([]Course, error)
	// LessonOutlines нь сургалтуудын хичээлийн гарчиг, бүлэг, хэлбэр (агуулга, блокгүй — хөнгөн).
	LessonOutlines(ctx context.Context, courseIDs []string) ([]Lesson, error)

	CreateLesson(ctx context.Context, l *Lesson) error
	LessonsByCourse(ctx context.Context, courseID string) ([]Lesson, error)
	LessonByID(ctx context.Context, courseID, lessonID string) (*Lesson, error)
	// UpdateLesson нь гарчиг, агуулга, медиа, үнэгүй/төлбөртэй, үнэ, дараалсан нээлтийн тохиргоог шинэчилнэ.
	UpdateLesson(ctx context.Context, l *Lesson) error
	// DeleteLesson нь хичээлийг устгана (явц, худалдан авалтын түүх хэвээр үлдэнэ).
	DeleteLesson(ctx context.Context, courseID, lessonID string) error
	// ReorderLessons нь хичээлүүдийн дараалал (1..n) ба бүлгийг нэг дор шинэчилнэ. items нь сургалтын бүх хичээлийг агуулна.
	ReorderLessons(ctx context.Context, courseID string, items []LessonOrder) error
	// SaveQuizResult нь суралцагчийн асуултын сүүлийн хариултыг (зөв/буруу) явцад хадгална.
	SaveQuizResult(ctx context.Context, userID, courseID, lessonID, blockID string, correct bool) error
	// Явц: MarkLessonViewed идемпотент (анхны үзэлтийн цаг хадгална), MarkLessonCompleted дууссаныг тэмдэглэнэ.
	MarkLessonViewed(ctx context.Context, userID, courseID, lessonID string) error
	MarkLessonCompleted(ctx context.Context, userID, courseID, lessonID string) error
	// MarkQuizDone нь хичээлийн бүх асуултад зөв хариулсныг нэг удаа тэмдэглэнэ (идемпотент).
	MarkQuizDone(ctx context.Context, userID, courseID, lessonID string) error
	// CourseProgress нь сургалтын бүх суралцагчийн явц: user_id → lesson_id → явц (багшийн статистикт).
	CourseProgress(ctx context.Context, courseID string) (map[string]map[string]LessonProgress, error)
	LessonProgress(ctx context.Context, userID, courseID string) (map[string]LessonProgress, error)

	IsEnrolled(ctx context.Context, userID, courseID string) (bool, error)
	Enroll(ctx context.Context, userID string, c *Course) error
	EnrolledCourses(ctx context.Context, userID string) ([]Course, error)
	// CoursesByIDs нь өгөгдсөн сургалтуудыг буцаана (олдоогүйг алгасна, дараалал баталгаагүй).
	CoursesByIDs(ctx context.Context, ids []string) ([]Course, error)
	// UserOrders нь хэрэглэгчийн сургалт/хичээлийн захиалгууд (багтаамжийнх орохгүй), шинээс хуучин.
	UserOrders(ctx context.Context, userID string, limit int) ([]Order, error)
	// UserConversations нь бүртгэлтэй хэрэглэгчийн багш нартай хийсэн яриа, сүүлийн мессежээр эрэмбэлсэн.
	UserConversations(ctx context.Context, userID string, limit int) ([]Conversation, error)
	// MeetingsByCourses нь өгөгдсөн сургалтуудын from-оос хойш дуусах шууд хичээлүүд, эхлэх цагаар.
	MeetingsByCourses(ctx context.Context, courseIDs []string, from time.Time, limit int) ([]Meeting, error)

	// CreateOrGetPendingOrder нь хэрэглэгч+сургалт дээр хүлээгдэж буй нэг л
	// захиалга байлгана (давхар дарсан ч нэг захиалга).
	CreateOrGetPendingOrder(ctx context.Context, userID string, c *Course) (*Order, error)
	// CreateOrGetPendingLessonOrder — нэг хичээл худалдан авах захиалга.
	CreateOrGetPendingLessonOrder(ctx context.Context, userID string, c *Course, l *Lesson) (*Order, error)
	// CreateOrGetPendingLateOrder — хоцорсон шалгалт/даалгаврыг нээх төлбөрийн захиалга (төлөгдвөл late_pass).
	CreateOrGetPendingLateOrder(ctx context.Context, userID string, c *Course, l *Lesson, fee int64) (*Order, error)
	// PurchasedLessons нь хэрэглэгчийн тухайн сургалтаас дангаар худалдаж авсан хичээлүүд.
	PurchasedLessons(ctx context.Context, userID, courseID string) ([]string, error)
	// CreateStorageOrder нь багтаамж худалдан авах захиалга.
	CreateStorageOrder(ctx context.Context, userID string, mb int64, months int, amount int64) (*Order, error)
	OrderByID(ctx context.Context, id string) (*Order, error)
	// MarkOrderPaid нь идемпотент: дахин дуудахад алдаагүй, элсэлт/багтаамж давхардахгүй.
	MarkOrderPaid(ctx context.Context, orderID string, amount int64) (*Order, error)
	TeacherSales(ctx context.Context, teacherID string, limit int) (*SalesSummary, error)
	// CourseStudents нь сургалтын суралцагчид (элссэн + дангаар хичээл авсан), шинээс хуучин.
	CourseStudents(ctx context.Context, courseID string) ([]CourseStudent, error)

	// Чат
	GetOrCreateConversation(ctx context.Context, teacherID, visitorKey, visitorName, userID string) (*Conversation, error)
	// GetOrCreateGroupConversation нь сургалтын бүлэг чат (нэг сургалтад нэг).
	GetOrCreateGroupConversation(ctx context.Context, teacherID, courseID, title string) (*Conversation, error)
	// UserGroupConversations нь хэрэглэгчийн элссэн эсвэл хичээл худалдаж авсан сургалтуудын бүлэг чатууд.
	UserGroupConversations(ctx context.Context, userID string, limit int) ([]Conversation, error)
	ConversationByID(ctx context.Context, id string) (*Conversation, error)
	// AddMessage нь m.TeacherID, m.VisitorKey-г яриагаас бөглөнө.
	AddMessage(ctx context.Context, m *Message) error
	// Messages нь beforeID-с өмнөх ("" бол хамгийн сүүлийн) limit мессежийг хуучнаас шинэ рүү буцаана.
	Messages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error)
	TeacherConversations(ctx context.Context, teacherID string, limit int) ([]Conversation, error)

	Close()
}
