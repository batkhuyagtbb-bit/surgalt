package store

import (
	"context"
	"time"
)

// ExamAttempt — суралцагчийн нэг удаагийн шалгалт.
type ExamAttempt struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	UserName   string          `json:"user_name"`
	CourseID   string          `json:"course_id"`
	LessonID   string          `json:"lesson_id"`
	TeacherID  string          `json:"teacher_id"`
	StartedAt  time.Time       `json:"started_at"`
	DeadlineAt *time.Time      `json:"deadline_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	Status     string          `json:"status"` // active | submitted | terminated | expired
	Reason     string          `json:"reason,omitempty"`
	Score      float64         `json:"score"`
	Max        float64         `json:"max"`
	Pct        int             `json:"pct"`
	Passed     bool            `json:"passed"`
	Violations int             `json:"violations"`
	Results    map[string]bool `json:"results,omitempty"` // асуулт → зөв эсэх
	Order      []string        `json:"order,omitempty"`   // холисон дараалал
}

const (
	AttemptActive     = "active"
	AttemptSubmitted  = "submitted"
	AttemptTerminated = "terminated"
	AttemptExpired    = "expired"
)

// StudySession — суралцагчийн нэг удаагийн үзэлт (хичээл, шалгалт, ном). Идэвхтэй, идэвхгүй,
// өөр цонхонд байсан хугацаа ба анхаарлын оноог зүрхний цохилт (heartbeat) бүрээр нэмнэ.
type StudySession struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	UserName  string         `json:"user_name"`
	CourseID  string         `json:"course_id"`
	LessonID  string         `json:"lesson_id"`
	TeacherID string         `json:"teacher_id"`
	Kind      string         `json:"kind"` // lesson | exam | book
	Title     string         `json:"title"`
	StartedAt time.Time      `json:"started_at"`
	LastAt    time.Time      `json:"last_at"`
	ActiveSec int            `json:"active_sec"`
	IdleSec   int            `json:"idle_sec"`
	AwaySec   int            `json:"away_sec"`
	FocusSum  float64        `json:"focus_sum"`
	FocusN    int            `json:"focus_n"`
	Camera    bool           `json:"camera"`
	IP        string         `json:"ip"`
	Ended     bool           `json:"ended"`
	EndReason string         `json:"end_reason,omitempty"`
	Counts    map[string]int `json:"counts"` // үйл явдлын төрөл → тоо
}

// SessionBeat — нэг зүрхний цохилтын өсөлт.
type SessionBeat struct {
	ActiveSec, IdleSec, AwaySec int
	FocusSum                    float64
	FocusN                      int
	Counts                      map[string]int
	Camera                      bool
	End                         string // хоосон биш бол сесс дуусна (шалтгаан)
}

// ActivityEvent — багшид харагдах лог: зөрчил, анхааруулга, хаалт, шалгалт.
type ActivityEvent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	CourseID  string    `json:"course_id"`
	LessonID  string    `json:"lesson_id"`
	TeacherID string    `json:"teacher_id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	Detail    string    `json:"detail,omitempty"`
	At        time.Time `json:"at"`
}

// ActivityFilter — хоосон талбар шүүхгүй.
type ActivityFilter struct {
	TeacherID, CourseID, UserID, LessonID, Kind string
	Since                                       time.Time
	// BeforeAt/BeforeID — хуудаслалтын курсор: (at, id)-ээс өмнөх мөрүүд (зөвхөн ActivityEvents).
	BeforeAt time.Time
	BeforeID string
}

// QuizLog — хичээл доторх асуултад өгсөн нэг хариулт ба хариулах хурд (таамаглалыг илрүүлэхэд).
type QuizLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	CourseID  string    `json:"course_id"`
	LessonID  string    `json:"lesson_id"`
	TeacherID string    `json:"teacher_id"`
	BlockID   string    `json:"block_id"`
	Question  string    `json:"question"`
	Correct   bool      `json:"correct"`
	Ms        int       `json:"ms"` // асуулт харагдсанаас хариулах хүртэл
	At        time.Time `json:"at"`
}

// Reflection — хичээлийн дараа суралцагчийн бичсэн товч дүгнэлт ("юу сурсан бэ?").
type Reflection struct {
	ID        string    `json:"id"` // user:lesson — нэг хичээлд нэг дүгнэлт
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	CourseID  string    `json:"course_id"`
	LessonID  string    `json:"lesson_id"`
	Lesson    string    `json:"lesson"`
	TeacherID string    `json:"teacher_id"`
	Text      string    `json:"text"`
	Words     int       `json:"words"`
	At        time.Time `json:"at"`
}

// VideoWatch — видеоны хэсэг (10 секунд) бүрийг хэдэн удаа үзсэн (үзэлтийн зураглал).
type VideoWatch struct {
	ID        string      `json:"id"` // user:lesson:block
	UserID    string      `json:"user_id"`
	UserName  string      `json:"user_name"`
	CourseID  string      `json:"course_id"`
	LessonID  string      `json:"lesson_id"`
	TeacherID string      `json:"teacher_id"`
	BlockID   string      `json:"block_id"`
	Duration  int         `json:"duration"` // секунд
	Buckets   map[int]int `json:"buckets"`  // хэсгийн дугаар → үзсэн тоо
	At        time.Time   `json:"at"`
}

// VideoBucketSec — үзэлтийн зураглалын нэг хэсгийн урт.
const VideoBucketSec = 10

// LearningStore — шалгалт, идэвхийн хяналтын хадгалалт.
type LearningStore interface {
	CreateExamAttempt(ctx context.Context, a *ExamAttempt) error
	ExamAttemptByID(ctx context.Context, id string) (*ExamAttempt, error)
	ExamAttempts(ctx context.Context, f ActivityFilter) ([]ExamAttempt, error)
	// FinishExamAttempt нь зөвхөн active төлөвтэйг дуусгана; бусад үед ErrConflict.
	FinishExamAttempt(ctx context.Context, a *ExamAttempt) error
	AddAttemptViolation(ctx context.Context, id string) error

	CreateSession(ctx context.Context, s *StudySession) error
	SessionByID(ctx context.Context, id string) (*StudySession, error)
	AddSessionBeat(ctx context.Context, id string, b SessionBeat) error
	Sessions(ctx context.Context, f ActivityFilter, limit int) ([]StudySession, error)
	AddActivityEvents(ctx context.Context, evs []ActivityEvent) error
	ActivityEvents(ctx context.Context, f ActivityFilter, limit int) ([]ActivityEvent, error)

	// Цэргийн цол: сургалт бүрийн оноог хадгалж нийлбэрийг буцаана; олгосон түвшинг хэрэглэгчээр хадгална
	// (систем автоматаар олгож, түвшин дээшлэхэд мэдэгдэнэ).
	SaveRankPoints(ctx context.Context, userID, courseID string, points int) (total int, err error)
	// RankCourseIDs — суралцагчийн оноо хадгалагдсан сургалтууд (элсээгүй, зөвхөн үнэгүй хичээл үзсэн ч).
	RankCourseIDs(ctx context.Context, userID string) ([]string, error)
	UserRankLevel(ctx context.Context, userID string) (int, error)
	SetUserRankLevel(ctx context.Context, userID string, level int) error
	// Даалгаврын хариу: нэг суралцагч нэг даалгаварт нэг (сүүлийн) хариу.
	SaveSubmission(ctx context.Context, sub *Submission) error
	SubmissionFor(ctx context.Context, userID, lessonID string) (*Submission, error)
	Submissions(ctx context.Context, lessonID string) ([]Submission, error)
	GradeSubmission(ctx context.Context, lessonID, userID string, score int, feedback string) error
	AddQuizLog(ctx context.Context, l QuizLog) error
	QuizLogs(ctx context.Context, f ActivityFilter, limit int) ([]QuizLog, error)
	SaveReflection(ctx context.Context, r *Reflection) error
	Reflections(ctx context.Context, f ActivityFilter, limit int) ([]Reflection, error)
	// AddVideoWatch нь хэсэг бүрийн тоог нэмнэ (байхгүй бол үүсгэнэ).
	AddVideoWatch(ctx context.Context, v VideoWatch) error
	VideoWatches(ctx context.Context, f ActivityFilter) ([]VideoWatch, error)
}
