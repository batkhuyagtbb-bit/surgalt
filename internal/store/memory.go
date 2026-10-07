package store

import (
	"cmp"
	"context"
	"slices"

	"sort"
	"strings"
	"sync"
	"time"
)

// Memory нь хөгжүүлэлт, тестэд зориулсан санах ойн store. Production-д Postgres ашиглана.
type Memory struct {
	mu          sync.RWMutex
	users       map[string]*User
	byEmail     map[string]string
	byUsername  map[string]string
	courses     map[string]*Course
	lessons     map[string][]*Lesson // course_id -> lessons
	orders      map[string]*Order
	pending     map[[2]string]string // (user,course) -> order id
	enrollments map[[2]string]time.Time
	convs       map[string]*Conversation
	convByKey   map[string]string // teacherID|visitorKey
	messages    map[string][]*Message
	identities  map[string]string // provider:subject -> user id
	applied     map[string]bool   // хэрэгжсэн багтаамжийн захиалга
	meetings    []*Meeting
	meetAcc     map[[2]string]string          // (user, meeting) → багш: худалдаж авсан шууд хичээл
	lessonAcc   map[[2]string]string          // (user, lesson) -> course
	pendingL    map[[2]string]string          // (user, lesson) -> order
	notifs      map[string][]*Notification    // user -> шинээс хуучин биш, нэмэгдэх дарааллаар
	progress    map[[2]string]*LessonProgress // (user, lesson)
	discuss     *discussMem
	discussOnce sync.Once
	chatX       *chatExtraMem
	chatXOnce   sync.Once
	teamMembers map[string]map[string]bool // team conv → гишүүд
	learn       learnMem                   // шалгалт, сесс, лог (memory_learning.go)
	engage      engageMem                  // идэвхийн нэмэлт (memory_engagement.go)
	books       bookMem                    // ном (memory_books.go)
}

func NewMemory() *Memory {
	return &Memory{
		users: map[string]*User{}, byEmail: map[string]string{}, byUsername: map[string]string{},
		courses: map[string]*Course{}, lessons: map[string][]*Lesson{}, orders: map[string]*Order{},
		pending: map[[2]string]string{}, enrollments: map[[2]string]time.Time{},
		convs: map[string]*Conversation{}, convByKey: map[string]string{}, messages: map[string][]*Message{}, identities: map[string]string{}, applied: map[string]bool{}, notifs: map[string][]*Notification{}, progress: map[[2]string]*LessonProgress{}, lessonAcc: map[[2]string]string{}, pendingL: map[[2]string]string{},
	}
}

func (m *Memory) Close() {}

// next нь процесс дотор өсөх дараалалтай ObjectID үүсгэнэ.
func (m *Memory) next() string { return NewID() }

func (m *Memory) CreateUser(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := strings.ToLower(u.Email)
	if _, ok := m.byEmail[email]; ok {
		return ErrConflict
	}
	if _, ok := m.byUsername[u.Username]; ok {
		return ErrConflict
	}
	u.ID, u.CreatedAt, u.Email = m.next(), time.Now(), email
	u.Subjects, u.Links = cloneStrings(u.Subjects), cloneLinks(u.Links)
	c := *u
	c.Subjects, c.Links = cloneStrings(u.Subjects), cloneLinks(u.Links)
	m.users[u.ID], m.byEmail[email], m.byUsername[u.Username] = &c, u.ID, u.ID
	return nil
}

func (m *Memory) userLocked(id string) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *u
	c.Subjects, c.Links = cloneStrings(u.Subjects), cloneLinks(u.Links)
	return &c, nil
}

// cloneStrings / cloneLinks нь nil-ийг хоосон утга болгож хуулна: JSON-д null гарахгүй,
// дуудагч буцаасан утгыг өөрчилсөн ч store-ын төлөв хөндөгдөхгүй.
func cloneStrings(in []string) []string {
	return append(make([]string, 0, len(in)), in...)
}

func cloneLinks(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m *Memory) UserByID(_ context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userLocked(id)
}

func (m *Memory) UserByEmail(_ context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, ErrNotFound
	}
	return m.userLocked(id)
}

func (m *Memory) UserByUsername(_ context.Context, username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byUsername[username]
	if !ok {
		return nil, ErrNotFound
	}
	return m.userLocked(id)
}

func (m *Memory) UpdateProfile(_ context.Context, id string, p ProfileUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	u.DisplayName, u.Headline, u.Bio, u.AvatarURL = p.DisplayName, p.Headline, p.Bio, p.AvatarURL
	u.CoverURL = p.CoverURL
	u.Subjects, u.Location, u.Links = cloneStrings(p.Subjects), p.Location, cloneLinks(p.Links)
	return nil
}

func (m *Memory) TeacherStudentCount(_ context.Context, teacherID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := map[string]bool{}
	for k := range m.enrollments {
		if c, ok := m.courses[k[1]]; ok && c.TeacherID == teacherID {
			seen[k[0]] = true
		}
	}
	return int64(len(seen)), nil
}

func (m *Memory) courseCopyLocked(c *Course) Course {
	cc := *c
	cc.LessonCount, cc.FreeLessonCount = 0, 0
	for _, l := range m.lessons[c.ID] {
		cc.LessonCount++
		if l.IsFree {
			cc.FreeLessonCount++
		}
	}
	return cc
}

func (m *Memory) CreateCourse(_ context.Context, c *Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[c.TeacherID]; !ok {
		return ErrNotFound
	}
	now := time.Now()
	c.ID, c.CreatedAt, c.UpdatedAt = m.next(), now, now
	cc := *c
	m.courses[c.ID] = &cc
	return nil
}

func (m *Memory) UpdateCourse(_ context.Context, c *Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.courses[c.ID]
	if !ok || cur.TeacherID != c.TeacherID {
		return ErrNotFound
	}
	cur.Title, cur.Description, cur.Price, cur.Published, cur.UpdatedAt = c.Title, c.Description, c.Price, c.Published, time.Now()
	cur.Drip, cur.UnlockAllPaid, cur.Camera, cur.Certificate = c.Drip, c.UnlockAllPaid, c.Camera, c.Certificate
	cur.MaxWarnings, cur.BlockHours, cur.BlockMinutes = c.MaxWarnings, c.BlockHours, c.BlockMinutes
	*c = m.courseCopyLocked(cur)
	return nil
}

func (m *Memory) CourseByID(_ context.Context, id string) (*Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.courses[id]
	if !ok {
		return nil, ErrNotFound
	}
	cc := m.courseCopyLocked(c)
	return &cc, nil
}

func (m *Memory) CoursesByTeacher(_ context.Context, teacherID string, onlyPublished bool) ([]Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Course{}
	for _, c := range m.courses {
		if c.TeacherID == teacherID && (!onlyPublished || c.Published) {
			out = append(out, m.courseCopyLocked(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) CreateLesson(_ context.Context, l *Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.courses[l.CourseID]; !ok {
		return ErrNotFound
	}
	l.ID, l.CreatedAt, l.Position = m.next(), time.Now(), len(m.lessons[l.CourseID])+1
	lc := *l
	m.lessons[l.CourseID] = append(m.lessons[l.CourseID], &lc)
	return nil
}

func (m *Memory) LessonsByCourse(_ context.Context, courseID string) ([]Lesson, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Lesson, 0, len(m.lessons[courseID]))
	for _, l := range m.lessons[courseID] {
		out = append(out, *l)
	}
	return out, nil
}

func (m *Memory) LessonByID(_ context.Context, courseID, lessonID string) (*Lesson, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, l := range m.lessons[courseID] {
		if l.ID == lessonID {
			lc := *l
			return &lc, nil
		}
	}
	return nil, ErrNotFound
}

func (m *Memory) IsEnrolled(_ context.Context, userID, courseID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.enrollments[[2]string{userID, courseID}]
	return ok, nil
}

func (m *Memory) Enroll(_ context.Context, userID string, c *Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, c.ID}
	if _, ok := m.enrollments[k]; !ok {
		m.enrollments[k] = time.Now()
	}
	return nil
}

func (m *Memory) EnrolledCourses(_ context.Context, userID string) ([]Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Course{}
	for k := range m.enrollments {
		if k[0] == userID {
			if c, ok := m.courses[k[1]]; ok {
				out = append(out, m.courseCopyLocked(c))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) CreateOrGetPendingOrder(_ context.Context, userID string, c *Course) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	courseID, amount := c.ID, c.Price
	k := [2]string{userID, courseID}
	if id, ok := m.pending[k]; ok {
		o := m.orders[id]
		o.Amount = amount
		oc := *o
		return &oc, nil
	}
	o := &Order{ID: m.next(), Kind: OrderKindCourse, Title: c.Title, UserID: userID, CourseID: courseID, TeacherID: c.TeacherID, Amount: amount, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID], m.pending[k] = o, o.ID
	oc := *o
	return &oc, nil
}

func (m *Memory) CreateStorageOrder(_ context.Context, userID string, mb int64, months int, amount int64) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[userID]; !ok {
		return nil, ErrNotFound
	}
	o := &Order{ID: m.next(), Kind: OrderKindStorage, UserID: userID, StorageMB: mb, Months: months, Amount: amount,
		Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID] = o
	oc := *o
	return &oc, nil
}

func (m *Memory) OrderByID(_ context.Context, id string) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	oc := *o
	return &oc, nil
}

func (m *Memory) MarkOrderPaid(_ context.Context, orderID string, amount int64) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok {
		return nil, ErrNotFound
	}
	if o.Status != OrderPaid {
		if o.Amount != amount {
			return nil, ErrConflict
		}
		now := time.Now()
		o.Status, o.PaidAt = OrderPaid, &now
	}
	switch o.Kind {
	case OrderKindLesson:
		delete(m.pendingL, [2]string{o.UserID, o.LessonID})
		m.lessonAcc[[2]string{o.UserID, o.LessonID}] = o.CourseID
	case OrderKindLate, OrderKindFee:
		delete(m.pendingL, [2]string{o.UserID, o.Kind + ":" + o.LessonID})
		k := [2]string{o.UserID, o.LessonID}
		p, ok := m.progress[k]
		if !ok {
			p = &LessonProgress{LessonID: o.LessonID, ViewedAt: time.Now()}
			m.progress[k] = p
		}
		if p.Quiz == nil {
			p.Quiz = map[string]bool{}
		}
		p.Quiz[FeePassKey] = true
		if o.Kind == OrderKindLate {
			p.Quiz[LatePassKey] = true
		}
	case OrderKindBook:
		m.books.grant(o.UserID, o.BookID, o.ID, o.Amount)
	case OrderKindMeeting:
		if m.meetAcc == nil {
			m.meetAcc = map[[2]string]string{}
		}
		m.meetAcc[[2]string{o.UserID, o.MeetingID}] = o.TeacherID
	case OrderKindStorage:
		if !m.applied[o.ID] {
			m.applied[o.ID] = true
			if u := m.users[o.UserID]; u != nil {
				base := time.Now()
				if u.StorageExpiresAt != nil && u.StorageExpiresAt.After(base) {
					base = *u.StorageExpiresAt
				}
				exp := base.Add(time.Duration(o.Months) * StorageMonth)
				u.StorageExtraBytes, u.StorageExpiresAt = o.StorageMB<<20, &exp
			}
		}
	default:
		delete(m.pending, [2]string{o.UserID, o.CourseID})
		k := [2]string{o.UserID, o.CourseID}
		if _, ok := m.enrollments[k]; !ok {
			m.enrollments[k] = *o.PaidAt
		}
	}
	oc := *o
	return &oc, nil
}

func (m *Memory) TeacherSales(_ context.Context, teacherID string, limit int) (*SalesSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sum := &SalesSummary{Recent: []Sale{}}
	for _, o := range m.orders {
		c := m.courses[o.CourseID]
		if o.Kind == OrderKindStorage || o.Status != OrderPaid || c == nil || c.TeacherID != teacherID {
			continue
		}
		sum.TotalAmount += o.Amount
		sum.Count++
		buyer := ""
		if u := m.users[o.UserID]; u != nil {
			buyer = u.Username
		}
		sum.Recent = append(sum.Recent, Sale{OrderID: o.ID, CourseID: c.ID, CourseTitle: o.Title, BuyerUsername: buyer, Amount: o.Amount, PaidAt: *o.PaidAt})
	}
	sort.Slice(sum.Recent, func(i, j int) bool { return sum.Recent[i].PaidAt.After(sum.Recent[j].PaidAt) })
	if len(sum.Recent) > limit {
		sum.Recent = sum.Recent[:limit]
	}
	return sum, nil
}

func (m *Memory) GetOrCreateConversation(_ context.Context, teacherID, visitorKey, visitorName, userID string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[teacherID]; !ok {
		return nil, ErrNotFound
	}
	key := teacherID + "|" + visitorKey
	if id, ok := m.convByKey[key]; ok {
		c := m.convs[id]
		c.VisitorName = visitorName
		cc := *c
		return &cc, nil
	}
	now := time.Now()
	c := &Conversation{ID: m.next(), TeacherID: teacherID, VisitorKey: visitorKey, VisitorName: visitorName, UserID: userID, CreatedAt: now, LastMessageAt: now}
	m.convs[c.ID], m.convByKey[key] = c, c.ID
	cc := *c
	return &cc, nil
}

func (m *Memory) ConversationByID(_ context.Context, id string) (*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.convs[id]
	if !ok {
		return nil, ErrNotFound
	}
	cc := *c
	return &cc, nil
}

func (m *Memory) AddMessage(_ context.Context, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.convs[msg.ConversationID]
	if !ok {
		return ErrNotFound
	}
	msg.ID, msg.CreatedAt, msg.TeacherID, msg.VisitorKey = m.next(), time.Now(), c.TeacherID, c.VisitorKey
	if msg.SenderName == "" {
		if u := m.users[msg.SenderID]; u != nil {
			msg.SenderName = u.DisplayName
		}
	}
	mc := *msg
	m.messages[c.ID] = append(m.messages[c.ID], &mc)
	c.LastMessageAt, c.LastMessage = msg.CreatedAt, truncate(msg.Body, 200)
	return nil
}

func (m *Memory) Messages(_ context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := m.messages[conversationID]
	end := len(all)
	if beforeID != "" {
		end = sort.Search(len(all), func(i int) bool { return all[i].ID >= beforeID })
	}
	start := max(0, end-limit)
	out := make([]Message, 0, end-start)
	for _, msg := range all[start:end] {
		out = append(out, *msg)
	}
	return out, nil
}

func (m *Memory) TeacherConversations(_ context.Context, teacherID string, limit int) ([]Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Conversation{}
	for _, c := range m.convs {
		if c.TeacherID == teacherID && c.LastMessage != "" {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastMessageAt.After(out[j].LastMessageAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func (m *Memory) UserByIdentity(_ context.Context, provider, subject string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.identities[provider+":"+subject]
	if !ok {
		return nil, ErrNotFound
	}
	return m.userLocked(id)
}

func (m *Memory) LinkIdentity(_ context.Context, userID, provider, subject string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := provider + ":" + subject
	if cur, ok := m.identities[k]; ok && cur != userID {
		return ErrConflict
	}
	m.identities[k] = userID
	return nil
}

func (m *Memory) SetGoogleToken(_ context.Context, userID, encrypted string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	u.GoogleToken, u.MeetConnected = encrypted, encrypted != ""
	return nil
}

func (m *Memory) CreateMeeting(_ context.Context, mt *Meeting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mt.ID, mt.CreatedAt = m.next(), time.Now()
	c := *mt
	m.meetings = append(m.meetings, &c)
	return nil
}

func (m *Memory) MeetingByID(_ context.Context, id string) (*Meeting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mt := range m.meetings {
		if mt.ID == id {
			c := *mt
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *Memory) SetMeetingPrice(_ context.Context, id string, price int64, membersFree bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mt := range m.meetings {
		if mt.ID == id {
			mt.Price, mt.MembersFree = price, membersFree
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) CreateOrGetPendingMeetingOrder(_ context.Context, userID string, mt *Meeting, title string) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.orders {
		if o.Kind == OrderKindMeeting && o.UserID == userID && o.MeetingID == mt.ID && o.Status == OrderPending {
			o.Amount = mt.Price
			oc := *o
			return &oc, nil
		}
	}
	o := &Order{ID: m.next(), Kind: OrderKindMeeting, Title: title, UserID: userID, CourseID: mt.CourseID, MeetingID: mt.ID,
		TeacherID: mt.TeacherID, Amount: mt.Price, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID] = o
	oc := *o
	return &oc, nil
}

func (m *Memory) MeetingAccess(_ context.Context, userID string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]bool{}
	for k := range m.meetAcc {
		if k[0] == userID {
			out[k[1]] = true
		}
	}
	return out, nil
}

func (m *Memory) MeetingBuyers(_ context.Context, teacherID string) (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]int{}
	for k, t := range m.meetAcc {
		if t == teacherID {
			out[k[1]]++
		}
	}
	return out, nil
}

func (m *Memory) Meetings(_ context.Context, teacherID, courseID string, from time.Time, limit int) ([]Meeting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Meeting{}
	for _, mt := range m.meetings {
		if mt.TeacherID == teacherID && (courseID == "" || mt.CourseID == courseID) &&
			mt.StartsAt.Add(time.Duration(mt.DurationMin)*time.Minute).After(from) {
			out = append(out, *mt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) IncViews(_ context.Context, profile, course map[string]int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, n := range profile {
		if u, ok := m.users[id]; ok {
			u.ProfileViews += n
		}
	}
	for id, n := range course {
		if c, ok := m.courses[id]; ok {
			c.Views += n
		}
	}
	return nil
}

func (m *Memory) AddNotifications(_ context.Context, ns []*Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range ns {
		n.ID, n.CreatedAt = m.next(), time.Now()
		c := *n
		m.notifs[n.UserID] = append(m.notifs[n.UserID], &c)
		if l := m.notifs[n.UserID]; len(l) > 500 {
			m.notifs[n.UserID] = l[len(l)-500:]
		}
	}
	return nil
}

func (m *Memory) Notifications(_ context.Context, userID string, limit int) ([]Notification, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := m.notifs[userID]
	var unread int64
	for _, n := range all {
		if !n.Read {
			unread++
		}
	}
	out := []Notification{}
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, *all[i])
	}
	return out, unread, nil
}

func (m *Memory) MarkNotificationsRead(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range m.notifs[userID] {
		n.Read = true
	}
	return nil
}

func (m *Memory) UpdateLesson(_ context.Context, l *Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cur := range m.lessons[l.CourseID] {
		if cur.ID == l.ID {
			cur.Title, cur.Content, cur.VideoURL, cur.IsFree, cur.Price = l.Title, l.Content, l.VideoURL, l.IsFree, l.Price
			cur.UnlockAfterH, cur.AlwaysOpen, cur.Format, cur.Mode, cur.Section, cur.Blocks = l.UnlockAfterH, l.AlwaysOpen, l.Format, l.Mode, l.Section, l.Blocks
			cur.ActiveMin, cur.Exam, cur.Assignment, cur.Discussion, cur.UnlockRule = l.ActiveMin, l.Exam, l.Assignment, l.Discussion, l.UnlockRule
			*l = *cur
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) DeleteLesson(_ context.Context, courseID, lessonID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ls := m.lessons[courseID]
	for i, cur := range ls {
		if cur.ID == lessonID {
			m.lessons[courseID] = append(ls[:i:i], ls[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) CreateOrGetPendingLessonOrder(_ context.Context, userID string, c *Course, l *Lesson) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, l.ID}
	if id, ok := m.pendingL[k]; ok {
		o := m.orders[id]
		o.Amount = l.Price
		oc := *o
		return &oc, nil
	}
	o := &Order{ID: m.next(), Kind: OrderKindLesson, Title: c.Title + " — " + l.Title, UserID: userID, CourseID: c.ID,
		LessonID: l.ID, TeacherID: c.TeacherID, Amount: l.Price, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID], m.pendingL[k] = o, o.ID
	oc := *o
	return &oc, nil
}

func (m *Memory) CreateOrGetPendingPassOrder(_ context.Context, userID string, c *Course, l *Lesson, kind string, amount int64) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, kind + ":" + l.ID}
	if id, ok := m.pendingL[k]; ok {
		o := m.orders[id]
		o.Amount = amount
		oc := *o
		return &oc, nil
	}
	suffix := map[string]string{OrderKindLate: " (хоцорсон)", OrderKindFee: " (оролцооны төлбөр)"}[kind]
	o := &Order{ID: m.next(), Kind: kind, Title: c.Title + " — " + l.Title + suffix, UserID: userID, CourseID: c.ID,
		LessonID: l.ID, TeacherID: c.TeacherID, Amount: amount, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID], m.pendingL[k] = o, o.ID
	oc := *o
	return &oc, nil
}

func (m *Memory) PurchasedLessons(_ context.Context, userID, courseID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []string{}
	for k, cid := range m.lessonAcc {
		if k[0] == userID && cid == courseID {
			out = append(out, k[1])
		}
	}
	return out, nil
}

func (m *Memory) CoursesByIDs(_ context.Context, ids []string) ([]Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Course, 0, len(ids))
	for _, id := range ids {
		if c, ok := m.courses[id]; ok {
			out = append(out, m.courseCopyLocked(c))
		}
	}
	return out, nil
}

func (m *Memory) UserOrders(_ context.Context, userID string, limit int) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Order{}
	for _, o := range m.orders {
		if o.UserID == userID && o.Kind != OrderKindStorage {
			out = append(out, *o)
		}
	}
	// ObjectID нь цаг хугацаагаар өсдөг тул ижил агшинд үүссэн захиалгыг ID-аар ялгана.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) UserConversations(_ context.Context, userID string, limit int) ([]Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Conversation{}
	for _, c := range m.convs {
		if c.UserID == userID && c.VisitorKey == "u:"+userID && c.LastMessage != "" {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastMessageAt.After(out[j].LastMessageAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) MeetingsByCourses(_ context.Context, courseIDs []string, from time.Time, limit int) ([]Meeting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	want := make(map[string]bool, len(courseIDs))
	for _, id := range courseIDs {
		want[id] = true
	}
	out := []Meeting{}
	for _, mt := range m.meetings {
		if mt.CourseID != "" && want[mt.CourseID] && mt.StartsAt.Add(time.Duration(mt.DurationMin)*time.Minute).After(from) {
			out = append(out, *mt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) UsersByIDs(_ context.Context, ids []string) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]User, 0, len(ids))
	for _, id := range ids {
		if u, err := m.userLocked(id); err == nil {
			out = append(out, *u)
		}
	}
	return out, nil
}

func (m *Memory) UpdateUsername(_ context.Context, id, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	if owner, taken := m.byUsername[username]; taken {
		if owner == id {
			return nil
		}
		return ErrConflict
	}
	delete(m.byUsername, u.Username)
	u.Username, m.byUsername[username] = username, id
	return nil
}

func (m *Memory) GetOrCreateGroupConversation(_ context.Context, teacherID, courseID, title string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.courses[courseID]
	if !ok || c.TeacherID != teacherID {
		return nil, ErrNotFound
	}
	key := teacherID + "|" + GroupVisitorKey(courseID)
	if id, ok := m.convByKey[key]; ok {
		cc := *m.convs[id]
		return &cc, nil
	}
	now := time.Now()
	conv := &Conversation{ID: m.next(), TeacherID: teacherID, VisitorKey: GroupVisitorKey(courseID), VisitorName: title,
		Kind: ConvGroup, CourseID: courseID, CreatedAt: now, LastMessageAt: now}
	m.convs[conv.ID], m.convByKey[key] = conv, conv.ID
	cc := *conv
	return &cc, nil
}

func (m *Memory) UserGroupConversations(_ context.Context, userID string, limit int) ([]Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	courses := map[string]bool{}
	for k := range m.enrollments {
		if k[0] == userID {
			courses[k[1]] = true
		}
	}
	for k, cid := range m.lessonAcc {
		if k[0] == userID {
			courses[cid] = true
		}
	}
	out := []Conversation{}
	for _, c := range m.convs {
		if c.Kind == ConvGroup && courses[c.CourseID] {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastMessageAt.After(out[j].LastMessageAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) CourseStudents(_ context.Context, courseID string) ([]CourseStudent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	by := map[string]*CourseStudent{}
	get := func(uid string) *CourseStudent {
		if cs, ok := by[uid]; ok {
			return cs
		}
		cs := &CourseStudent{UserID: uid, LessonIDs: []string{}}
		by[uid] = cs
		return cs
	}
	for k, at := range m.enrollments {
		if k[1] == courseID {
			t := at
			get(k[0]).EnrolledAt = &t
		}
	}
	for k, cid := range m.lessonAcc {
		if cid == courseID {
			cs := get(k[0])
			cs.LessonIDs = append(cs.LessonIDs, k[1])
		}
	}
	out := make([]CourseStudent, 0, len(by))
	for _, cs := range by {
		sort.Strings(cs.LessonIDs)
		out = append(out, *cs)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := time.Time{}, time.Time{}
		if out[i].EnrolledAt != nil {
			ti = *out[i].EnrolledAt
		}
		if out[j].EnrolledAt != nil {
			tj = *out[j].EnrolledAt
		}
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].UserID < out[j].UserID
	})
	return out, nil
}

func (m *Memory) MarkLessonViewed(_ context.Context, userID, courseID, lessonID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, lessonID}
	if _, ok := m.progress[k]; !ok {
		m.progress[k] = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
	}
	return nil
}

func (m *Memory) MarkLessonCompleted(_ context.Context, userID, courseID, lessonID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, lessonID}
	p, ok := m.progress[k]
	if !ok {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
		m.progress[k] = p
	}
	if p.CompletedAt == nil {
		now := time.Now()
		p.CompletedAt = &now
	}
	return nil
}

func (m *Memory) MarkQuizDone(_ context.Context, userID, courseID, lessonID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, lessonID}
	p, ok := m.progress[k]
	if !ok {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
		m.progress[k] = p
	}
	if p.QuizDoneAt == nil {
		now := time.Now()
		p.QuizDoneAt = &now
	}
	return nil
}

func copyProgress(p *LessonProgress) LessonProgress {
	cp := *p
	if p.Quiz != nil {
		cp.Quiz = make(map[string]bool, len(p.Quiz))
		for k, v := range p.Quiz {
			cp.Quiz[k] = v
		}
	}
	return cp
}

func (m *Memory) LessonProgress(_ context.Context, userID, courseID string) (map[string]LessonProgress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]LessonProgress{}
	for _, l := range m.lessons[courseID] {
		if p, ok := m.progress[[2]string{userID, l.ID}]; ok {
			out[l.ID] = copyProgress(p)
		}
	}
	return out, nil
}

func (m *Memory) CourseProgress(_ context.Context, courseID string) (map[string]map[string]LessonProgress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]map[string]LessonProgress{}
	for _, l := range m.lessons[courseID] {
		for k, p := range m.progress {
			if k[1] != l.ID {
				continue
			}
			if out[k[0]] == nil {
				out[k[0]] = map[string]LessonProgress{}
			}
			out[k[0]][l.ID] = copyProgress(p)
		}
	}
	return out, nil
}

// BackdateProgress нь тестэд: хичээлийг үзсэн цагийг хойш нь шилжүүлнэ (таймер өнгөрсөн мэт).
func (m *Memory) BackdateProgress(userID, lessonID string, by time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.progress[[2]string{userID, lessonID}]; ok {
		p.ViewedAt = p.ViewedAt.Add(-by)
		if p.QuizDoneAt != nil {
			t := p.QuizDoneAt.Add(-by)
			p.QuizDoneAt = &t
		}
	}
}

func (m *Memory) ReorderLessons(_ context.Context, courseID string, items []LessonOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.lessons[courseID]
	if len(cur) != len(items) {
		return ErrNotFound
	}
	byID := make(map[string]*Lesson, len(cur))
	for _, l := range cur {
		byID[l.ID] = l
	}
	out := make([]*Lesson, 0, len(items))
	for _, it := range items {
		l, ok := byID[it.ID]
		if !ok {
			return ErrNotFound
		}
		delete(byID, it.ID) // давхардлыг хориглоно
		out = append(out, l)
	}
	// Бүгд хүчинтэй бол л өөрчилнө (хагас хадгалалт үүсэхгүй).
	for i, l := range out {
		l.Position, l.Section = i+1, items[i].Section
	}
	m.lessons[courseID] = out
	return nil
}

func (m *Memory) SaveQuizResult(_ context.Context, userID, courseID, lessonID, blockID string, correct bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, lessonID}
	p, ok := m.progress[k]
	if !ok {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
		m.progress[k] = p
	}
	if p.Quiz == nil {
		p.Quiz = map[string]bool{}
	}
	p.Quiz[blockID] = correct
	return nil
}

func (m *Memory) PublishedCourses(_ context.Context, limit int) ([]Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Course{}
	for _, c := range m.courses {
		if c.Published {
			out = append(out, *c)
		}
	}
	slices.SortFunc(out, func(a, b Course) int { return cmp.Compare(b.Views, a.Views) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) LessonOutlines(_ context.Context, courseIDs []string) ([]Lesson, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Lesson{}
	for _, id := range courseIDs {
		for _, l := range m.lessons[id] {
			c := *l
			c.Content, c.Blocks, c.VideoURL = "", nil, ""
			out = append(out, c)
		}
	}
	return out, nil
}
