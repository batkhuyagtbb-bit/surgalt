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
	"surgalt/internal/mail"
	"surgalt/internal/meet"
	"surgalt/internal/oauth"
	"surgalt/internal/qpay"
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
	ContactEmail  string        // нууцлал, нөхцөлийн хуудсанд харагдах холбоо барих имэйл
	AdminEmails   []string      // чатын мэдүүлэг шийдвэрлэх хянагчид (жижиг үсгээр; moderation.go)
	Mail          mail.Sender   // нууц үг сэргээх имэйл (nil бол илгээхгүй; демо горимд холбоосыг логт гаргана)
	PayoutMin     int64         // багшийн орлого татах доод дүн (₮; 0 бол 1000)
	// ServiceFeePct — төлбөр бүрийн үнэ дээр нэмж авах үйлчилгээний хураамж (%, жишээ нь 1). Багшийн орлогод орохгүй.
	ServiceFeePct float64
	// PixabayKey — цагаан самбарын зураг хайлтад Pixabay (SafeSearch) ашиглах түлхүүр (хоосон бол Openverse + Wikimedia).
	PixabayKey string
	// Хянагчийн самбар (admin.go): тусдаа зам ба нууц үгээр нэвтрэх (хоосон бол /admin, зөвхөн ADMIN_EMAILS).
	AdminPath     string
	AdminLogin    string
	AdminPassHash string // bcrypt

	// Хайлтын системд сайтаа баталгаажуулах код (Google Search Console, Bing Webmaster) — seo.go.
	GoogleSiteVerification, BingSiteVerification string
}

type Server struct {
	cfg       Config
	store     store.Store
	tokens    *auth.Signer
	hub       *chat.Hub
	files     *files.Store
	log       *slog.Logger
	tmpl      *template.Template
	limiter   *ratelimit.Limiter
	authLim   *ratelimit.Limiter // нэвтрэлт/бүртгэл/зочин токен — илүү чанга
	codeLim   *ratelimit.Limiter // сургалтын нууц үг таах оролдлого (course_code.go)
	chatLim   *ratelimit.Limiter // мессеж илгээх
	phoneLim  *ratelimit.Limiter // багшийн утасны дугаар нээх — дугаар хусуурдахаас сэргийлнэ
	resetLim  *ratelimit.Limiter // нэг имэйл рүү нууц үг сэргээх захидал (2 минутад 1 орчим)
	admLim    *ratelimit.Limiter // хянагчийн нууц үгээр нэвтрэх оролдлого (IP тус бүрт)
	boardLim  *ratelimit.Limiter // цагаан самбарын өөрчлөлт (хэрэглэгч тус бүрт; анги нэг IP-аас орж болно)
	boards    boardHub           // цагаан самбарын сессүүд (board.go)
	searchLim *ratelimit.Limiter // самбарын зураг хайлт (хэрэглэгч тус бүрт)
	imgs      imgSearch          // зураг хайлтын кэш, орчуулга (board_images.go)
	hits      hitBuf             // сайтын хандалтын түр сан (admin.go)
	adm       adminCache         // хянагчийн самбарын агшин зураг
	sys       sysCache           // хянагчийн «Систем»: файлын сангийн хэмжээ (60 сек кэш)
	started   time.Time          // сервер асаасан мөч
	profiles  *cache.Cache[*Rendered[PublicProfile]]
	courses   *cache.Cache[*Rendered[PublicCourse]]
	qrs       *cache.Cache[[]byte]
	ogs       *cache.Cache[[]byte] // профайлыг хуваалцах зураг (og.go)
	alive     *cache.Cache[int64]  // хэрэглэгч бүрийн «энэ цагаас өмнөх токен хүчингүй» (миллисекунд; account.go)
	seen      presence             // багшийн онлайн төлөв, сүүлд идэвхтэй байсан мөч (presence.go)
	smap      sitemapCache         // /sitemap.xml (seo.go)
	notif     *notifier
	search    *searchIndex

	// OAuth нь Google/Microsoft/Facebook/Instagram нэвтрэлт (nil бол идэвхгүй).
	OAuth *oauth.Registry
	// Meet нь Google Meet автомат уулзалт (nil бол идэвхгүй).
	Meet *meet.Client
	// QPay нь QR төлбөр (nil бол идэвхгүй; DEV_PAYMENTS үед демо QR).
	QPay *qpay.Client

	// Publish нь шинэ мессежийг бусад серверүүдэд хүргэнэ.
	// nil бол зөвхөн локал hub (ClickHouse store нэг сервер хуулбартай ажилладаг).
	Publish func(store.Message)
	// PublishNotif нь мэдэгдлийг түгээнэ (change stream идэвхтэй бол nil).
	PublishNotif func(store.Notification)
}

func New(cfg Config, st store.Store, tokens *auth.Signer, hub *chat.Hub, fstore *files.Store, log *slog.Logger) *Server {
	isNF := func(err error) bool { return errors.Is(err, store.ErrNotFound) }
	siteVerifyGoogle, siteVerifyBing = cfg.GoogleSiteVerification, cfg.BingSiteVerification
	s := &Server{
		cfg: cfg, store: st, tokens: tokens, hub: hub, files: fstore, log: log,
		tmpl:      parseTemplates(),
		limiter:   ratelimit.New(cfg.RateLimitRPS, cfg.RateBurst),
		authLim:   ratelimit.New(1, 10),
		codeLim:   ratelimit.New(1.0/30, 6),
		chatLim:   ratelimit.New(2, 10),
		phoneLim:  ratelimit.New(0.05, 8),
		resetLim:  ratelimit.New(1.0/120, 2),
		admLim:    ratelimit.New(1.0/30, 5),
		boardLim:  ratelimit.New(25, 100),
		searchLim: ratelimit.New(0.5, 12),
		profiles:  cache.New[*Rendered[PublicProfile]](30*time.Second, 5*time.Second, isNF),
		courses:   cache.New[*Rendered[PublicCourse]](30*time.Second, 5*time.Second, isNF),
		qrs:       cache.New[[]byte](time.Hour, 5*time.Second, isNF),
		ogs:       cache.New[[]byte](time.Hour, 5*time.Second, isNF),
		alive:     cache.New[int64](5*time.Minute, 5*time.Minute, isNF),
	}
	s.notif = newNotifier()
	s.started = time.Now()
	s.search = &searchIndex{}
	s.Publish, s.PublishNotif = hub.Deliver, hub.Notify
	return s
}

// Background нь кэш болон хязгаарлагчийн цэвэрлэгээг эхлүүлнэ.
func (s *Server) Background(ctx context.Context) {
	go s.profiles.Janitor(ctx, time.Minute)
	go s.courses.Janitor(ctx, time.Minute)
	go s.qrs.Janitor(ctx, 10*time.Minute)
	go s.alive.Janitor(ctx, 10*time.Minute)
	go s.limiter.Janitor(ctx)
	go s.authLim.Janitor(ctx)
	go s.codeLim.Janitor(ctx)
	go s.chatLim.Janitor(ctx)
	go s.phoneLim.Janitor(ctx)
	go s.resetLim.Janitor(ctx)
	go s.admLim.Janitor(ctx)
	go s.boardLim.Janitor(ctx)
	go s.searchLim.Janitor(ctx)
	go s.boardLoop(ctx)
	go s.hitLoop(ctx)
	go s.notifyLoop(ctx)
	go s.payoutLoop(ctx)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ws_connections": s.hub.Connections()})
	})

	// Нэвтрэлт
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/auth/providers", s.handleProviders)
	mux.HandleFunc("GET /auth/{provider}/start", s.handleOAuthStart)
	mux.HandleFunc("GET /auth/{provider}/callback", s.handleOAuthCallback)
	mux.HandleFunc("POST /auth/facebook/data-deletion", s.handleFBDataDeletion)

	// Өөрийн мэдээлэл
	mux.HandleFunc("GET /api/me", s.handleMe)
	mux.HandleFunc("GET /api/me/home", s.handleHome)
	mux.HandleFunc("GET /api/me/profile/insights", s.handleProfileInsights)
	mux.HandleFunc("PUT /api/me/profile", s.handleUpdateProfile)
	mux.HandleFunc("PUT /api/me/name", s.handleUpdateName)
	mux.HandleFunc("POST /api/me/consent", s.handleConsent)
	mux.HandleFunc("POST /api/me/role", s.handleSetRole)
	mux.HandleFunc("POST /api/auth/forgot", s.handleForgot)
	mux.HandleFunc("POST /api/auth/reset", s.handleReset)
	mux.HandleFunc("GET /reset", s.pageStatic("reset"))
	mux.HandleFunc("GET /api/me/export", s.handleExport)
	mux.HandleFunc("POST /api/me/delete", s.handleDeleteAccount)
	mux.HandleFunc("PUT /api/me/username", s.handleUpdateUsername)
	mux.HandleFunc("GET /api/me/courses", s.handleMyCourses)
	mux.HandleFunc("GET /api/me/courses/{id}", s.handleMyCourse)
	mux.HandleFunc("GET /api/me/enrollments", s.handleMyEnrollments)
	mux.HandleFunc("GET /api/me/sales", s.handleMySales)
	mux.HandleFunc("GET /api/me/students", s.handleMyStudents)
	mux.HandleFunc("GET /api/me/courses/{id}/students", s.handleCourseStudents)
	mux.HandleFunc("GET /api/me/courses/{id}/journal", s.handleCourseJournal)
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
	mux.HandleFunc("PUT /api/courses/{id}/lessons/{lid}/visibility", s.handleLessonVisibility)
	mux.HandleFunc("PUT /api/courses/{id}/lesson-order", s.handleReorderLessons)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/buy", s.handleBuyLesson)
	mux.HandleFunc("POST /api/courses/{id}/unlock", s.handleUnlockCourse)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/complete", s.handleCompleteLesson)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/quiz/{bid}", s.handleAnswerQuiz)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/watched/{bid}", s.handleVideoWatched)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/reflect", s.handleSaveReflection)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/highlights", s.handleGetHighlights)
	mux.HandleFunc("PUT /api/courses/{id}/lessons/{lid}/highlights", s.handleSetHighlights)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/cloud/{bid}", s.handleGetCloud)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/cloud/{bid}", s.handleSendCloud)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/board/{bid}", s.handleGetBoard)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/ops", s.handleBoardOps)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/presence", s.handleBoardPresence)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/state", s.handleBoardState)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/image", s.handleBoardImage)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/fetch", s.handleBoardFetch)
	mux.HandleFunc("GET /api/courses/{id}/lessons/{lid}/board/{bid}/chat", s.handleBoardChat)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/chat", s.handleBoardChatPost)
	mux.HandleFunc("POST /api/courses/{id}/lessons/{lid}/board/{bid}/voice", s.handleBoardVoice)
	mux.HandleFunc("GET /board-img/{lid}/{name}", s.handleBoardImg)
	mux.HandleFunc("GET /api/board/images", s.handleBoardImages)
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
	mux.HandleFunc("POST /api/me/students/{uid}/unblock", s.handleUnblockStudent)
	mux.HandleFunc("GET /api/me/analytics", s.handleAnalytics)
	mux.HandleFunc("GET /api/me/analytics/students/{uid}", s.handleStudentAnalytics)
	mux.HandleFunc("GET /api/me/analytics/export", s.handleAnalyticsExport)
	mux.HandleFunc("GET /api/me/analytics/events", s.handleAnalyticsEvents)
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
	mux.HandleFunc("GET /api/books/{id}/download", s.handleBookDownload)
	mux.HandleFunc("POST /api/books/{id}/interest", s.handleBookInterest)
	mux.HandleFunc("GET /api/me/notes", s.handleMyNotes)
	mux.HandleFunc("POST /api/me/story-view", s.handleStoryView)
	mux.HandleFunc("GET /api/me/courses/{id}/notes-stats", s.handleNotesStats)
	mux.HandleFunc("POST /api/me/note-images", s.handleNoteImage)
	mux.HandleFunc("GET /note-img/{uid}/{name}", s.handleNoteImg)
	mux.HandleFunc("GET /api/books/{id}/notes", s.handleGetBookNotes)
	mux.HandleFunc("PUT /api/books/{id}/notes", s.handleSetBookNotes)
	mux.HandleFunc("POST /api/me/quiz-import", s.handleQuizImport)

	// Төлбөр
	mux.HandleFunc("GET /api/orders/{id}", s.handleGetOrder)
	mux.HandleFunc("POST /api/orders/{id}/dev-pay", s.handleDevPay)
	mux.HandleFunc("POST /api/payments/webhook", s.handlePaymentWebhook)
	mux.HandleFunc("GET /api/payments/qpay", s.handleQPayCallback)
	mux.HandleFunc("POST /api/payments/qpay", s.handleQPayCallback)
	mux.HandleFunc("GET /pay/{id}", s.handlePayPage)
	// Багшийн данс: орлого татах, статистик (payouts.go)
	mux.HandleFunc("GET /api/me/payout", s.handleMyPayout)
	mux.HandleFunc("PUT /api/me/payout/account", s.handleSetPayoutAccount)
	mux.HandleFunc("POST /api/me/payout/withdraw", s.handleWithdraw)
	mux.HandleFunc("GET /api/me/earnings", s.handleMyEarnings)
	mux.HandleFunc("GET /api/admin/payouts", s.handleAdminPayouts)
	mux.HandleFunc("POST /api/admin/payouts/{id}", s.handleAdminPayout)

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
	mux.HandleFunc("PUT /api/me/meetings/{id}", s.handleMeetingPrice)
	mux.HandleFunc("POST /api/meetings/{id}/buy", s.handleBuyMeeting)
	mux.HandleFunc("GET /api/courses/{id}/meetings", s.handleCourseMeetings)
	mux.HandleFunc("POST /api/chat/{id}/meet", s.handleChatMeet)

	// Нээлттэй профайл
	mux.HandleFunc("GET /api/teachers/{username}", s.handlePublicProfile)
	mux.HandleFunc("POST /api/teachers/{username}/phone", s.handleRevealPhone)
	// Цаг захиалга: багшийн календарь ба захиалга
	mux.HandleFunc("GET /api/teachers/{username}/slots", s.handleTeacherSlots)
	mux.HandleFunc("POST /api/slots/{id}/book", s.handleBookSlot)
	mux.HandleFunc("GET /api/me/slots", s.handleMySlots)
	mux.HandleFunc("POST /api/me/slots", s.handleAddSlots)
	mux.HandleFunc("DELETE /api/me/slots/{id}", s.handleDeleteSlot)
	mux.HandleFunc("GET /api/me/bookings", s.handleMyBookings)
	mux.HandleFunc("PUT /api/me/slots/{id}/meet", s.handleSlotMeet)
	mux.HandleFunc("POST /api/me/slots/cancel", s.handleCancelSlots)
	mux.HandleFunc("POST /api/views", s.handleTrackView)
	mux.HandleFunc("POST /api/views/{id}/end", s.handleVisitEnd)
	mux.HandleFunc("GET /api/me/visits", s.handleMyVisits)
	mux.HandleFunc("GET /t/{username}/qr.png", s.handleQR)
	mux.HandleFunc("GET /t/{username}/og.jpg", s.handleProfileOG)

	// Чат
	mux.HandleFunc("POST /api/teachers/{username}/chat", s.handleStartChat)
	mux.HandleFunc("GET /api/teachers/{username}/presence", s.handleTeacherPresence)
	mux.HandleFunc("GET /api/me/presence", s.handleMyPresence)
	mux.HandleFunc("DELETE /api/chat/{id}", s.handleHideChat)
	mux.HandleFunc("POST /api/courses/{id}/chat", s.handleCourseChat)
	mux.HandleFunc("GET /api/chat/{id}/messages", s.handleListMessages)
	mux.HandleFunc("POST /api/chat/{id}/messages", s.handleSendMessage)
	mux.HandleFunc("PUT /api/chat/{id}/messages/{mid}", s.handleEditMessage)
	mux.HandleFunc("DELETE /api/chat/{id}/messages/{mid}", s.handleDeleteMessage)
	mux.HandleFunc("POST /api/chat/{id}/react", s.handleReact)
	mux.HandleFunc("POST /api/chat/{id}/read", s.handleRead)
	mux.HandleFunc("POST /api/chat/{id}/typing", s.handleTyping)
	mux.HandleFunc("POST /api/chat/{id}/upload", s.handleChatUpload)
	mux.HandleFunc("POST /api/chat/{id}/report", s.handleReport)
	mux.HandleFunc("POST /api/chat/{id}/block", s.handleBlock)
	mux.HandleFunc("DELETE /api/chat/{id}/block", s.handleBlock)
	// Хянагчийн самбар: зөвхөн ADMIN_PATH-аар (тохируулсан бол /admin нээгдэхгүй).
	mux.HandleFunc("GET "+s.adminPath(), s.pageAdmin)
	mux.HandleFunc("POST /api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("POST /api/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("GET /api/admin/me", s.handleAdminMe)
	mux.HandleFunc("GET /api/admin/overview", s.handleAdminOverview)
	mux.HandleFunc("GET /api/admin/teachers", s.handleAdminTeachers)
	mux.HandleFunc("GET /api/admin/teachers/{id}", s.handleAdminTeacher)
	mux.HandleFunc("GET /api/admin/students", s.handleAdminStudents)
	mux.HandleFunc("GET /api/admin/students/{id}", s.handleAdminStudent)
	mux.HandleFunc("POST /api/admin/students/{id}/make-teacher", s.handleAdminMakeTeacher)
	mux.HandleFunc("GET /api/admin/transactions", s.handleAdminTransactions)
	mux.HandleFunc("GET /api/admin/finance", s.handleAdminFinance)
	mux.HandleFunc("GET /api/admin/system", s.handleAdminSystem)
	mux.HandleFunc("POST /api/admin/expenses", s.handleAdminAddExpense)
	mux.HandleFunc("DELETE /api/admin/expenses/{id}", s.handleAdminDeleteExpense)
	mux.HandleFunc("POST /api/hit", s.handleHit)
	mux.HandleFunc("GET /api/admin/reports", s.handleAdminReports)
	mux.HandleFunc("POST /api/admin/reports/{id}/resolve", s.handleAdminResolve)
	mux.HandleFunc("GET /api/me/classmates", s.handleClassmates)
	mux.HandleFunc("POST /api/me/dm/{uid}", s.handleStartDM)
	mux.HandleFunc("POST /api/me/teams", s.handleCreateTeam)
	mux.HandleFunc("GET /api/chat/ws", s.handleWS)

	// Хуудсууд
	mux.HandleFunc("GET /{$}", s.pageHome)
	mux.HandleFunc("GET /robots.txt", s.handleRobots)
	mux.HandleFunc("GET /sitemap.xml", s.handleSitemap)
	mux.HandleFunc("GET /t/{username}", s.pageProfile)
	mux.HandleFunc("GET /c/{id}", s.pageCourse)
	mux.HandleFunc("GET /b/{id}", s.pageBook)
	mux.HandleFunc("GET /login", s.pageStatic("login"))
	mux.HandleFunc("GET /privacy", s.pageStatic("privacy"))
	mux.HandleFunc("GET /intro2", s.pageIntro)
	mux.HandleFunc("GET /terms", s.pageStatic("terms"))
	mux.HandleFunc("GET /data-deletion", s.pageDataDeletion)
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
		if !strings.HasPrefix(r.URL.Path, "/static/") && !boardHot(r) && !s.limiter.Allow(s.clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт, түр хүлээнэ үү")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// boardHot — цагаан самбарын байнгын хүсэлтүүд (өөрчлөлт, «байна» дохио, зураг): нэг ангийн хүүхдүүд нэг IP-аас
// орж болох тул IP-гаар биш, хэрэглэгч тус бүрээр (boardLim) хязгаарлана.
func boardHot(r *http.Request) bool {
	p := r.URL.Path
	if strings.HasPrefix(p, "/board-img/") || strings.HasPrefix(p, "/note-img/") {
		return r.Method == http.MethodGet
	}
	return r.Method == http.MethodPost && strings.HasPrefix(p, "/api/courses/") && strings.Contains(p, "/board/") &&
		(strings.HasSuffix(p, "/ops") || strings.HasSuffix(p, "/presence"))
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
