// Package httpapi нь HTTP давхарга: REST API, WebSocket чат, HTML хуудсууд.
package httpapi

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/cache"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/meet"
	"surgalt/internal/oauth"
	"surgalt/internal/ratelimit"
	"surgalt/internal/store"
)

type Config struct {
	PublicURL     string // жишээ нь https://surgalt.mn — QR болон OG холбоост
	WebhookSecret string
	DevPayments   bool // true бол /api/orders/{id}/dev-pay-ээр төлбөрийг дуурайна (демо)
	TrustProxy    bool // X-Real-IP / X-Forwarded-Proto-г итгэх эсэх (nginx ард)
	LogRequests   bool
	RateLimitRPS  float64 // IP тус бүрт; 0 бол хязгааргүй
	RateBurst     float64
	StorageFreeMB int64         // багш бүрт үнэгүй багтаамж
	StoragePlans  []StoragePlan // худалдаж авах багцууд
}

type Server struct {
	cfg      Config
	store    store.Store
	tokens   *auth.Signer
	hub      *chat.Hub
	files    *files.Store
	log      *slog.Logger
	tmpl     *template.Template
	limiter  *ratelimit.Limiter
	authLim  *ratelimit.Limiter // нэвтрэлт/бүртгэл/зочин токен — илүү чанга
	chatLim  *ratelimit.Limiter // мессеж илгээх
	profiles *cache.Cache[*Rendered[PublicProfile]]
	courses  *cache.Cache[*Rendered[PublicCourse]]
	qrs      *cache.Cache[[]byte]
	notif    *notifier
	search   *searchIndex

	// OAuth нь Google/Microsoft/Facebook/Instagram нэвтрэлт (nil бол идэвхгүй).
	OAuth *oauth.Registry
	// Meet нь Google Meet автомат уулзалт (nil бол идэвхгүй).
	Meet *meet.Client

	// Publish нь шинэ мессежийг бусад серверүүдэд хүргэнэ.
	// nil бол зөвхөн локал hub (ClickHouse store нэг сервер хуулбартай ажилладаг).
	Publish func(store.Message)
	// PublishNotif нь мэдэгдлийг түгээнэ (change stream идэвхтэй бол nil).
	PublishNotif func(store.Notification)
}

func New(cfg Config, st store.Store, tokens *auth.Signer, hub *chat.Hub, fstore *files.Store, log *slog.Logger) *Server {
	isNF := func(err error) bool { return errors.Is(err, store.ErrNotFound) }
	s := &Server{
		cfg: cfg, store: st, tokens: tokens, hub: hub, files: fstore, log: log,
		tmpl:     parseTemplates(),
		limiter:  ratelimit.New(cfg.RateLimitRPS, cfg.RateBurst),
		authLim:  ratelimit.New(1, 10),
		chatLim:  ratelimit.New(2, 10),
		profiles: cache.New[*Rendered[PublicProfile]](30*time.Second, 5*time.Second, isNF),
		courses:  cache.New[*Rendered[PublicCourse]](30*time.Second, 5*time.Second, isNF),
		qrs:      cache.New[[]byte](time.Hour, 5*time.Second, isNF),
	}
	s.notif = newNotifier()
	s.search = &searchIndex{}
	s.Publish, s.PublishNotif = hub.Deliver, hub.Notify
	return s
}

// Background нь кэш болон хязгаарлагчийн цэвэрлэгээг эхлүүлнэ.
func (s *Server) Background(ctx context.Context) {
	go s.profiles.Janitor(ctx, time.Minute)
	go s.courses.Janitor(ctx, time.Minute)
	go s.qrs.Janitor(ctx, 10*time.Minute)
	go s.limiter.Janitor(ctx)
	go s.authLim.Janitor(ctx)
	go s.chatLim.Janitor(ctx)
	go s.notifyLoop(ctx)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ws_connections": s.hub.Connections()})
	})

	// Нэвтрэлт
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/guest", s.handleGuest)
	mux.HandleFunc("GET /api/auth/providers", s.handleProviders)
	mux.HandleFunc("GET /auth/{provider}/start", s.handleOAuthStart)
	mux.HandleFunc("GET /auth/{provider}/callback", s.handleOAuthCallback)

	// Өөрийн мэдээлэл
	mux.HandleFunc("GET /api/me", s.handleMe)
	mux.HandleFunc("GET /api/me/home", s.handleHome)
	mux.HandleFunc("GET /api/me/profile/insights", s.handleProfileInsights)
	mux.HandleFunc("PUT /api/me/profile", s.handleUpdateProfile)
	mux.HandleFunc("PUT /api/me/username", s.handleUpdateUsername)
	mux.HandleFunc("GET /api/me/courses", s.handleMyCourses)
	mux.HandleFunc("GET /api/me/courses/{id}", s.handleMyCourse)
	mux.HandleFunc("GET /api/me/enrollments", s.handleMyEnrollments)
	mux.HandleFunc("GET /api/me/sales", s.handleMySales)
	mux.HandleFunc("GET /api/me/students", s.handleMyStudents)
	mux.HandleFunc("GET /api/me/courses/{id}/students", s.handleCourseStudents)
	mux.HandleFunc("POST /api/me/students/{uid}/chat", s.handleStudentChat)
	mux.HandleFunc("GET /api/me/conversations", s.handleMyConversations)
	mux.HandleFunc("GET /api/me/notifications", s.handleNotifications)
	mux.HandleFunc("POST /api/me/notifications/read", s.handleNotificationsRead)

	// Сургалт
	mux.HandleFunc("POST /api/courses", s.handleCreateCourse)
	mux.HandleFunc("PUT /api/courses/{id}", s.handleUpdateCourse)
	mux.HandleFunc("POST /api/courses/{id}/lessons", s.handleCreateLesson)
	mux.HandleFunc("GET /api/courses/{id}", s.handlePublicCourse)
	mux.HandleFunc("GET /api/courses/{id}/access", s.handleCourseAccess)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}", s.handleGetLesson)
	mux.HandleFunc("POST /api/courses/{id}/enroll", s.handleEnroll)
	mux.HandleFunc("PUT /api/courses/{id}/lessons/{lid}", s.handleUpdateLesson)
	mux.HandleFunc("DELETE /api/courses/{id}/lessons/{lid}", s.handleDeleteLesson)
	mux.HandleFunc("PUT /api/courses/{id}/lesson-order", s.handleReorderLessons)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/buy", s.handleBuyLesson)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/complete", s.handleCompleteLesson)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/quiz/{bid}", s.handleAnswerQuiz)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/watched/{bid}", s.handleVideoWatched)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/reflect", s.handleSaveReflection)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/progress/{bid}", s.handleVideoProgress)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/discussion", s.handleDiscussion)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/discussion", s.handleAddComment)
	mux.HandleFunc("DELETE /api/courses/{id}/lessons/{lid}/discussion/{cid}", s.handleDeleteComment)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/like", s.handleLike)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/assignment", s.handleAssignmentInfo)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/submit", s.handleSubmit)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/submissions", s.handleSubmissions)
	mux.HandleFunc("PUT /api/courses/{id}/lessons/{lid}/submissions/{uid}", s.handleGrade)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/late-pay", s.handleLatePay)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/exam", s.handleExamInfo)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/exam/start", s.handleExamStart)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/exam/submit", s.handleExamSubmit)
	mux.HandleFunc("POST /api/activity/start", s.handleActivityStart)
	mux.HandleFunc("POST /api/activity/beat", s.handleActivityBeat)
	mux.HandleFunc("POST /api/me/students/{uid}/remind", s.handleRemindStudent)
	mux.HandleFunc("GET /api/me/analytics", s.handleAnalytics)
	mux.HandleFunc("GET /api/me/analytics/students/{uid}", s.handleStudentAnalytics)
	mux.HandleFunc("GET /api/me/analytics/export", s.handleAnalyticsExport)
	mux.HandleFunc("GET /api/quiz-template.xlsx", s.handleQuizTemplate)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/me/books", s.handleMyBooks)
	mux.HandleFunc("POST /api/me/books", s.handleCreateBook)
	mux.HandleFunc("PUT /api/me/books/{id}", s.handleUpdateBook)
	mux.HandleFunc("DELETE /api/me/books/{id}", s.handleDeleteBook)
	mux.HandleFunc("POST /api/me/books/{id}/pages/{n}", s.handleUploadBookPage)
	mux.HandleFunc("POST /api/me/books/{id}/pages-done", s.handleBookPagesDone)
	mux.HandleFunc("GET /api/me/book-events", s.handleBookEvents)
	mux.HandleFunc("GET /api/books/{id}", s.handleBook)
	mux.HandleFunc("GET /api/books/{id}/pages/{n}", s.handleBookPage)
	mux.HandleFunc("GET /api/books/{id}/cover", s.handleBookCover)
	mux.HandleFunc("POST /api/books/{id}/buy", s.handleBuyBook)
	mux.HandleFunc("POST /api/books/{id}/interest", s.handleBookInterest)
	mux.HandleFunc("POST /api/me/quiz-import", s.handleQuizImport)

	// Төлбөр
	mux.HandleFunc("GET /api/orders/{id}", s.handleGetOrder)
	mux.HandleFunc("POST /api/orders/{id}/dev-pay", s.handleDevPay)
	mux.HandleFunc("POST /api/payments/webhook", s.handlePaymentWebhook)

	// Багшийн файлын сан (багш бүр тусдаа хавтастай)
	mux.HandleFunc("POST /api/me/files", s.handleUpload)
	mux.HandleFunc("GET /api/me/files", s.handleListFiles)
	mux.HandleFunc("GET /api/me/files/usage", s.handleFileUsage)
	mux.HandleFunc("DELETE /api/me/files/{visibility}/{name}", s.handleDeleteFile)
	mux.HandleFunc("GET /files/{teacher}/{visibility}/{name}", s.handleServeFile)
	mux.HandleFunc("GET /api/media/{ticket}/{name}", s.handleMedia)
	mux.HandleFunc("GET /api/storage/plans", s.handleStoragePlans)
	mux.HandleFunc("GET /api/me/storage", s.handleMyStorage)
	mux.HandleFunc("POST /api/me/storage/purchase", s.handleBuyStorage)

	// Google Meet
	mux.HandleFunc("POST /api/me/meet/connect", s.handleMeetConnect)
	mux.HandleFunc("DELETE /api/me/meet", s.handleMeetDisconnect)
	mux.HandleFunc("GET /auth/google-meet/callback", s.handleMeetCallback)
	mux.HandleFunc("GET /api/me/meetings", s.handleMyMeetings)
	mux.HandleFunc("POST /api/me/meetings", s.handleCreateMeeting)
	mux.HandleFunc("GET /api/courses/{id}/meetings", s.handleCourseMeetings)
	mux.HandleFunc("POST /api/chat/{id}/meet", s.handleChatMeet)

	// Нээлттэй профайл
	mux.HandleFunc("GET /api/teachers/{username}", s.handlePublicProfile)
	mux.HandleFunc("GET /t/{username}/qr.png", s.handleQR)

	// Чат
	mux.HandleFunc("POST /api/teachers/{username}/chat", s.handleStartChat)
	mux.HandleFunc("POST /api/courses/{id}/chat", s.handleCourseChat)
	mux.HandleFunc("GET /api/chat/{id}/messages", s.handleListMessages)
	mux.HandleFunc("POST /api/chat/{id}/messages", s.handleSendMessage)
	mux.HandleFunc("PUT /api/chat/{id}/messages/{mid}", s.handleEditMessage)
	mux.HandleFunc("DELETE /api/chat/{id}/messages/{mid}", s.handleDeleteMessage)
	mux.HandleFunc("POST /api/chat/{id}/react", s.handleReact)
	mux.HandleFunc("POST /api/chat/{id}/read", s.handleRead)
	mux.HandleFunc("POST /api/chat/{id}/typing", s.handleTyping)
	mux.HandleFunc("POST /api/chat/{id}/upload", s.handleChatUpload)
	mux.HandleFunc("GET /api/me/classmates", s.handleClassmates)
	mux.HandleFunc("POST /api/me/dm/{uid}", s.handleStartDM)
	mux.HandleFunc("POST /api/me/teams", s.handleCreateTeam)
	mux.HandleFunc("GET /api/chat/ws", s.handleWS)

	// Хуудсууд
	mux.HandleFunc("GET /{$}", s.pageHome)
	mux.HandleFunc("GET /t/{username}", s.pageProfile)
	mux.HandleFunc("GET /c/{id}", s.pageCourse)
	mux.HandleFunc("GET /b/{id}", s.pageBook)
	mux.HandleFunc("GET /login", s.pageStatic("login"))
	// /me — нэвтэрсэн хүний өөрийн хуудас руу шилжүүлнэ (багш: /t/<нэр>, суралцагч: /). Хуучин /studio холбоос ч мөн.
	mux.HandleFunc("GET /me", s.pageStatic("me"))
	mux.HandleFunc("GET /studio", s.pageStatic("me"))
	mux.Handle("GET /static/", staticHandler())

	return s.recoverMW(s.logMW(s.limitMW(mux)))
}

// ---- middleware ----

func (s *Server) recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("panic", "err", v, "path", r.URL.Path, "stack", string(debug.Stack()))
				writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int)        { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) logMW(next http.Handler) http.Handler {
	if !s.cfg.LogRequests {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		s.log.Info("req", "m", r.Method, "p", r.URL.Path, "s", sw.status, "d", time.Since(start))
	})
}

func (s *Server) limitMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static/") && !s.limiter.Allow(s.clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт, түр хүлээнэ үү")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) baseURL(r *http.Request) string {
	if s.cfg.PublicURL != "" {
		return strings.TrimRight(s.cfg.PublicURL, "/")
	}
	scheme := "http"
	if r.TLS != nil || (s.cfg.TrustProxy && r.Header.Get("X-Forwarded-Proto") == "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
