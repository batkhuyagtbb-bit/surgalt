package store

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- сургалт ----

const courseCols = `id, teacher_id, title, description, price, published, drip, unlock_all_paid, camera, certificate,
	created_at, updated_at, deleted`

func scanCourse(r driver.Rows) (Course, bool, error) {
	var c Course
	var deleted bool
	err := r.Scan(&c.ID, &c.TeacherID, &c.Title, &c.Description, &c.Price, &c.Published, &c.Drip, &c.UnlockAllPaid,
		&c.Camera, &c.Certificate, &c.CreatedAt, &c.UpdatedAt, &deleted)
	return c, deleted, err
}

func (c *ClickHouse) writeCourse(ctx context.Context, co *Course) error {
	return c.insert(ctx, "courses", []string{"id", "teacher_id", "title", "description", "price", "published", "drip",
		"unlock_all_paid", "camera", "certificate", "created_at", "updated_at", "ver", "deleted"},
		co.ID, co.TeacherID, co.Title, co.Description, co.Price, co.Published, co.Drip, co.UnlockAllPaid, co.Camera,
		co.Certificate, co.CreatedAt.UTC(), co.UpdatedAt.UTC(), ver(), false)
}

// enrichCourses нь хичээлийн тоо (lessons-оос) ба үзэлт (counters-оос) бөглөнө.
func (c *ClickHouse) enrichCourses(ctx context.Context, cs []Course) error {
	if len(cs) == 0 {
		return nil
	}
	ids := make([]string, len(cs))
	for i := range cs {
		ids[i] = cs[i].ID
	}
	type cnt struct{ total, free int }
	counts := map[string]cnt{}
	err := c.query(ctx, `SELECT course_id, count(), countIf(is_free) FROM lessons FINAL
		WHERE has(?, course_id) AND deleted = false GROUP BY course_id`, []any{ids}, func(r driver.Rows) error {
		var id string
		var total, free uint64
		if err := r.Scan(&id, &total, &free); err != nil {
			return err
		}
		counts[id] = cnt{int(total), int(free)}
		return nil
	})
	if err != nil {
		return err
	}
	views, err := c.counters(ctx, "course", ids)
	if err != nil {
		return err
	}
	for i := range cs {
		cs[i].LessonCount, cs[i].FreeLessonCount = counts[cs[i].ID].total, counts[cs[i].ID].free
		cs[i].Views = views[cs[i].ID].views
	}
	return nil
}

func (c *ClickHouse) coursesWhere(ctx context.Context, where, order string, args ...any) ([]Course, error) {
	out := []Course{}
	err := c.query(ctx, "SELECT "+courseCols+" FROM courses FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		co, deleted, err := scanCourse(r)
		if err != nil {
			return err
		}
		if !deleted {
			out = append(out, co)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, c.enrichCourses(ctx, out)
}

func (c *ClickHouse) CreateCourse(ctx context.Context, co *Course) error {
	now := time.Now()
	co.ID, co.CreatedAt, co.UpdatedAt = NewID(), now, now
	return c.writeCourse(ctx, co)
}

func (c *ClickHouse) UpdateCourse(ctx context.Context, co *Course) error {
	unlock, err := c.lock(ctx, "course:"+co.ID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.CourseByID(ctx, co.ID)
	if err != nil {
		return err
	}
	if cur.TeacherID != co.TeacherID {
		return ErrNotFound
	}
	cur.Title, cur.Description, cur.Price, cur.Published = co.Title, co.Description, co.Price, co.Published
	cur.Drip, cur.UnlockAllPaid, cur.Camera, cur.Certificate = co.Drip, co.UnlockAllPaid, co.Camera, co.Certificate
	cur.UpdatedAt = time.Now()
	if err := c.writeCourse(ctx, cur); err != nil {
		return err
	}
	*co = *cur
	return nil
}

func (c *ClickHouse) CourseByID(ctx context.Context, id string) (*Course, error) {
	cs, err := c.coursesWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, ErrNotFound
	}
	return &cs[0], nil
}

func (c *ClickHouse) CoursesByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Course, error) {
	where := "teacher_id = ?"
	if onlyPublished {
		where += " AND published = true"
	}
	return c.coursesWhere(ctx, where, "ORDER BY created_at DESC LIMIT 500", teacherID)
}

func (c *ClickHouse) PublishedCourses(ctx context.Context, limit int) ([]Course, error) {
	cs, err := c.coursesWhere(ctx, "published = true", "LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Views > cs[j].Views })
	return cs, nil
}

func (c *ClickHouse) CoursesByIDs(ctx context.Context, ids []string) ([]Course, error) {
	if len(ids) == 0 {
		return []Course{}, nil
	}
	return c.coursesWhere(ctx, "has(?, id)", "", ids)
}

// ---- хичээл ----

const lessonCols = `id, course_id, title, content, video_url, is_free, price, unlock_after_h, always_open, format, mode,
	section, blocks, active_min, exam, position, created_at, deleted, assignment`

func scanLesson(r driver.Rows) (Lesson, bool, error) {
	var l Lesson
	var deleted bool
	var unlock, active, pos int32
	var blocks, exam, asg string
	err := r.Scan(&l.ID, &l.CourseID, &l.Title, &l.Content, &l.VideoURL, &l.IsFree, &l.Price, &unlock, &l.AlwaysOpen,
		&l.Format, &l.Mode, &l.Section, &blocks, &active, &exam, &pos, &l.CreatedAt, &deleted, &asg)
	if err != nil {
		return l, false, err
	}
	l.UnlockAfterH, l.ActiveMin, l.Position = int(unlock), int(active), int(pos)
	if blocks != "" {
		_ = json.Unmarshal([]byte(blocks), &l.Blocks)
	}
	if exam != "" {
		var e Exam
		if json.Unmarshal([]byte(exam), &e) == nil {
			l.Exam = &e
		}
	}
	if asg != "" {
		var a Assignment
		if json.Unmarshal([]byte(asg), &a) == nil {
			l.Assignment = &a
		}
	}
	return l, deleted, nil
}

func (c *ClickHouse) writeLesson(ctx context.Context, l *Lesson, deleted bool) error {
	blocks := ""
	if len(l.Blocks) > 0 {
		b, _ := json.Marshal(l.Blocks)
		blocks = string(b)
	}
	exam, asg := "", ""
	if l.Exam != nil {
		b, _ := json.Marshal(l.Exam)
		exam = string(b)
	}
	if l.Assignment != nil {
		b, _ := json.Marshal(l.Assignment)
		asg = string(b)
	}
	return c.insert(ctx, "lessons", []string{"id", "course_id", "title", "content", "video_url", "is_free", "price",
		"unlock_after_h", "always_open", "format", "mode", "section", "blocks", "active_min", "exam", "position", "created_at", "ver", "deleted", "assignment"},
		l.ID, l.CourseID, l.Title, l.Content, l.VideoURL, l.IsFree, l.Price, int32(l.UnlockAfterH), l.AlwaysOpen,
		l.Format, l.Mode, l.Section, blocks, int32(l.ActiveMin), exam, int32(l.Position), l.CreatedAt.UTC(), ver(), deleted, asg)
}

func (c *ClickHouse) lessonsWhere(ctx context.Context, where string, args ...any) ([]Lesson, error) {
	out := []Lesson{}
	err := c.query(ctx, "SELECT "+lessonCols+" FROM lessons FINAL WHERE "+where+" ORDER BY course_id, position, id", args,
		func(r driver.Rows) error {
			l, deleted, err := scanLesson(r)
			if err != nil {
				return err
			}
			if !deleted {
				out = append(out, l)
			}
			return nil
		})
	return out, err
}

func (c *ClickHouse) CreateLesson(ctx context.Context, l *Lesson) error {
	unlock, err := c.lock(ctx, "lessons:"+l.CourseID)
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := c.CourseByID(ctx, l.CourseID); err != nil {
		return err
	}
	n, err := c.count(ctx, "SELECT count() FROM lessons FINAL WHERE course_id = ? AND deleted = false", l.CourseID)
	if err != nil {
		return err
	}
	l.ID, l.CreatedAt, l.Position = NewID(), time.Now(), int(n)+1
	return c.writeLesson(ctx, l, false)
}

func (c *ClickHouse) LessonsByCourse(ctx context.Context, courseID string) ([]Lesson, error) {
	return c.lessonsWhere(ctx, "course_id = ?", courseID)
}

func (c *ClickHouse) LessonByID(ctx context.Context, courseID, lessonID string) (*Lesson, error) {
	ls, err := c.lessonsWhere(ctx, "course_id = ? AND id = ?", courseID, lessonID)
	if err != nil {
		return nil, err
	}
	if len(ls) == 0 {
		return nil, ErrNotFound
	}
	return &ls[0], nil
}

func (c *ClickHouse) UpdateLesson(ctx context.Context, l *Lesson) error {
	unlock, err := c.lock(ctx, "lessons:"+l.CourseID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.LessonByID(ctx, l.CourseID, l.ID)
	if err != nil {
		return err
	}
	cur.Title, cur.Content, cur.VideoURL, cur.IsFree, cur.Price = l.Title, l.Content, l.VideoURL, l.IsFree, l.Price
	cur.UnlockAfterH, cur.AlwaysOpen, cur.Format, cur.Mode, cur.Section, cur.Blocks = l.UnlockAfterH, l.AlwaysOpen, l.Format, l.Mode, l.Section, l.Blocks
	cur.ActiveMin, cur.Exam, cur.Assignment = l.ActiveMin, l.Exam, l.Assignment
	if err := c.writeLesson(ctx, cur, false); err != nil {
		return err
	}
	*l = *cur
	return nil
}

func (c *ClickHouse) DeleteLesson(ctx context.Context, courseID, lessonID string) error {
	unlock, err := c.lock(ctx, "lessons:"+courseID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.LessonByID(ctx, courseID, lessonID)
	if err != nil {
		return err
	}
	return c.writeLesson(ctx, cur, true)
}

func (c *ClickHouse) ReorderLessons(ctx context.Context, courseID string, items []LessonOrder) error {
	unlock, err := c.lock(ctx, "lessons:"+courseID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.LessonsByCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if len(cur) != len(items) {
		return ErrNotFound
	}
	byID := make(map[string]*Lesson, len(cur))
	for i := range cur {
		byID[cur[i].ID] = &cur[i]
	}
	ordered := make([]*Lesson, 0, len(items))
	for _, it := range items {
		l, ok := byID[it.ID]
		if !ok {
			return ErrNotFound
		}
		delete(byID, it.ID) // давхардлыг хориглоно
		ordered = append(ordered, l)
	}
	// Бүгд хүчинтэй бол л бичнэ.
	for i, l := range ordered {
		l.Position, l.Section = i+1, items[i].Section
		if err := c.writeLesson(ctx, l, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClickHouse) LessonOutlines(ctx context.Context, courseIDs []string) ([]Lesson, error) {
	if len(courseIDs) == 0 {
		return []Lesson{}, nil
	}
	ls, err := c.lessonsWhere(ctx, "has(?, course_id)", courseIDs)
	if err != nil {
		return nil, err
	}
	for i := range ls {
		ls[i].Content, ls[i].Blocks, ls[i].VideoURL = "", nil, ""
	}
	return ls, nil
}

// ---- явц ----

func (c *ClickHouse) progressRow(ctx context.Context, userID, lessonID string) (*LessonProgress, string, error) {
	var p *LessonProgress
	var courseID string
	err := c.query(ctx, `SELECT course_id, viewed_at, completed_at, quiz, quiz_done_at FROM lesson_progress FINAL
		WHERE user_id = ? AND lesson_id = ? LIMIT 1`, []any{userID, lessonID}, func(r driver.Rows) error {
		var lp LessonProgress
		var done, qdone *time.Time
		if err := r.Scan(&courseID, &lp.ViewedAt, &done, &lp.Quiz, &qdone); err != nil {
			return err
		}
		lp.LessonID, lp.CompletedAt, lp.QuizDoneAt = lessonID, done, qdone
		p = &lp
		return nil
	})
	return p, courseID, err
}

func (c *ClickHouse) writeProgress(ctx context.Context, userID, courseID string, p *LessonProgress) error {
	quiz := p.Quiz
	if quiz == nil {
		quiz = map[string]bool{}
	}
	return c.insert(ctx, "lesson_progress", []string{"user_id", "course_id", "lesson_id", "viewed_at", "completed_at", "quiz", "quiz_done_at", "ver"},
		userID, courseID, p.LessonID, p.ViewedAt.UTC(), nullTime(p.CompletedAt), quiz, nullTime(p.QuizDoneAt), ver())
}

func (c *ClickHouse) MarkLessonViewed(ctx context.Context, userID, courseID, lessonID string) error {
	unlock, err := c.lock(ctx, "prog:"+userID+":"+lessonID)
	if err != nil {
		return err
	}
	defer unlock()
	p, _, err := c.progressRow(ctx, userID, lessonID)
	if err != nil || p != nil {
		return err
	}
	return c.writeProgress(ctx, userID, courseID, &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()})
}

func (c *ClickHouse) MarkLessonCompleted(ctx context.Context, userID, courseID, lessonID string) error {
	unlock, err := c.lock(ctx, "prog:"+userID+":"+lessonID)
	if err != nil {
		return err
	}
	defer unlock()
	p, _, err := c.progressRow(ctx, userID, lessonID)
	if err != nil {
		return err
	}
	if p == nil {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
	}
	if p.CompletedAt != nil {
		return nil
	}
	now := time.Now()
	p.CompletedAt = &now
	return c.writeProgress(ctx, userID, courseID, p)
}

func (c *ClickHouse) SaveQuizResult(ctx context.Context, userID, courseID, lessonID, blockID string, correct bool) error {
	unlock, err := c.lock(ctx, "prog:"+userID+":"+lessonID)
	if err != nil {
		return err
	}
	defer unlock()
	p, _, err := c.progressRow(ctx, userID, lessonID)
	if err != nil {
		return err
	}
	if p == nil {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
	}
	if p.Quiz == nil {
		p.Quiz = map[string]bool{}
	}
	p.Quiz[blockID] = correct
	return c.writeProgress(ctx, userID, courseID, p)
}

func (c *ClickHouse) MarkQuizDone(ctx context.Context, userID, courseID, lessonID string) error {
	unlock, err := c.lock(ctx, "prog:"+userID+":"+lessonID)
	if err != nil {
		return err
	}
	defer unlock()
	p, _, err := c.progressRow(ctx, userID, lessonID)
	if err != nil {
		return err
	}
	if p == nil {
		p = &LessonProgress{LessonID: lessonID, ViewedAt: time.Now()}
	}
	if p.QuizDoneAt != nil {
		return nil
	}
	now := time.Now()
	p.QuizDoneAt = &now
	return c.writeProgress(ctx, userID, courseID, p)
}

func (c *ClickHouse) LessonProgress(ctx context.Context, userID, courseID string) (map[string]LessonProgress, error) {
	all, err := c.progressWhere(ctx, "user_id = ? AND course_id = ?", userID, courseID)
	if err != nil {
		return nil, err
	}
	if out := all[userID]; out != nil {
		return out, nil
	}
	return map[string]LessonProgress{}, nil
}

func (c *ClickHouse) CourseProgress(ctx context.Context, courseID string) (map[string]map[string]LessonProgress, error) {
	return c.progressWhere(ctx, "course_id = ?", courseID)
}

func (c *ClickHouse) progressWhere(ctx context.Context, where string, args ...any) (map[string]map[string]LessonProgress, error) {
	out := map[string]map[string]LessonProgress{}
	err := c.query(ctx, `SELECT user_id, lesson_id, viewed_at, completed_at, quiz, quiz_done_at FROM lesson_progress FINAL
		WHERE `+where, args, func(r driver.Rows) error {
		var lp LessonProgress
		var uid string
		var done, qdone *time.Time
		if err := r.Scan(&uid, &lp.LessonID, &lp.ViewedAt, &done, &lp.Quiz, &qdone); err != nil {
			return err
		}
		lp.CompletedAt, lp.QuizDoneAt = done, qdone
		if len(lp.Quiz) == 0 {
			lp.Quiz = nil
		}
		if out[uid] == nil {
			out[uid] = map[string]LessonProgress{}
		}
		out[uid][lp.LessonID] = lp
		return nil
	})
	return out, err
}

// ---- элсэлт ----

func (c *ClickHouse) IsEnrolled(ctx context.Context, userID, courseID string) (bool, error) {
	n, err := c.count(ctx, "SELECT count() FROM enrollments FINAL WHERE user_id = ? AND course_id = ?", userID, courseID)
	return n > 0, err
}

func (c *ClickHouse) enroll(ctx context.Context, userID, courseID, teacherID string, at time.Time) error {
	ok, err := c.IsEnrolled(ctx, userID, courseID)
	if err != nil || ok {
		return err
	}
	return c.insert(ctx, "enrollments", []string{"user_id", "course_id", "teacher_id", "created_at", "ver"},
		userID, courseID, teacherID, at.UTC(), ver())
}

func (c *ClickHouse) Enroll(ctx context.Context, userID string, co *Course) error {
	unlock, err := c.lock(ctx, "enroll:"+userID+":"+co.ID)
	if err != nil {
		return err
	}
	defer unlock()
	return c.enroll(ctx, userID, co.ID, co.TeacherID, time.Now())
}

func (c *ClickHouse) EnrolledCourses(ctx context.Context, userID string) ([]Course, error) {
	var ids []string
	err := c.query(ctx, "SELECT course_id FROM enrollments FINAL WHERE user_id = ? ORDER BY created_at DESC", []any{userID},
		func(r driver.Rows) error {
			var id string
			if err := r.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
			return nil
		})
	if err != nil {
		return nil, err
	}
	cs, err := c.CoursesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	pos := map[string]int{}
	for i, id := range ids {
		pos[id] = i
	}
	sort.SliceStable(cs, func(i, j int) bool { return pos[cs[i].ID] < pos[cs[j].ID] })
	return cs, nil
}

func (c *ClickHouse) PurchasedLessons(ctx context.Context, userID, courseID string) ([]string, error) {
	out := []string{}
	err := c.query(ctx, "SELECT lesson_id FROM lesson_access FINAL WHERE user_id = ? AND course_id = ?", []any{userID, courseID},
		func(r driver.Rows) error {
			var id string
			if err := r.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
			return nil
		})
	return out, err
}

func (c *ClickHouse) CourseStudents(ctx context.Context, courseID string) ([]CourseStudent, error) {
	by := map[string]*CourseStudent{}
	last := map[string]time.Time{}
	get := func(uid string, at time.Time) *CourseStudent {
		cs, ok := by[uid]
		if !ok {
			cs = &CourseStudent{UserID: uid, LessonIDs: []string{}}
			by[uid] = cs
		}
		if at.After(last[uid]) {
			last[uid] = at
		}
		return cs
	}
	err := c.query(ctx, "SELECT user_id, created_at FROM enrollments FINAL WHERE course_id = ?", []any{courseID}, func(r driver.Rows) error {
		var uid string
		var at time.Time
		if err := r.Scan(&uid, &at); err != nil {
			return err
		}
		t := at
		get(uid, at).EnrolledAt = &t
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = c.query(ctx, "SELECT user_id, lesson_id, created_at FROM lesson_access FINAL WHERE course_id = ?", []any{courseID}, func(r driver.Rows) error {
		var uid, lid string
		var at time.Time
		if err := r.Scan(&uid, &lid, &at); err != nil {
			return err
		}
		cs := get(uid, at)
		cs.LessonIDs = append(cs.LessonIDs, lid)
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]CourseStudent, 0, len(by))
	for _, cs := range by {
		out = append(out, *cs)
	}
	sort.Slice(out, func(i, j int) bool { return last[out[i].UserID].After(last[out[j].UserID]) })
	return out, nil
}

// ---- захиалга ----

const orderCols = `id, kind, storage_mb, months, user_id, course_id, lesson_id, book_id, teacher_id, title, amount, status,
	created_at, paid_at, applied`

type orderRow struct {
	Order
	applied bool
}

func scanOrder(r driver.Rows) (orderRow, error) {
	var o orderRow
	var months int32
	var status string
	var paid *time.Time
	err := r.Scan(&o.ID, &o.Kind, &o.StorageMB, &months, &o.UserID, &o.CourseID, &o.LessonID, &o.BookID, &o.TeacherID,
		&o.Title, &o.Amount, &status, &o.CreatedAt, &paid, &o.applied)
	o.Months, o.Status, o.PaidAt = int(months), OrderStatus(status), paid
	return o, err
}

func (c *ClickHouse) writeOrder(ctx context.Context, o *Order, applied bool) error {
	return c.insert(ctx, "orders", []string{"id", "kind", "storage_mb", "months", "user_id", "course_id", "lesson_id", "book_id",
		"teacher_id", "title", "amount", "status", "created_at", "paid_at", "applied", "ver"},
		o.ID, o.Kind, o.StorageMB, int32(o.Months), o.UserID, o.CourseID, o.LessonID, o.BookID, o.TeacherID, o.Title, o.Amount,
		string(o.Status), o.CreatedAt.UTC(), nullTime(o.PaidAt), applied, ver())
}

func (c *ClickHouse) ordersWhere(ctx context.Context, where, order string, args ...any) ([]orderRow, error) {
	out := []orderRow{}
	err := c.query(ctx, "SELECT "+orderCols+" FROM orders FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		o, err := scanOrder(r)
		if err != nil {
			return err
		}
		out = append(out, o)
		return nil
	})
	return out, err
}

func (c *ClickHouse) orderRowByID(ctx context.Context, id string) (*orderRow, error) {
	os, err := c.ordersWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(os) == 0 {
		return nil, ErrNotFound
	}
	return &os[0], nil
}

func (c *ClickHouse) OrderByID(ctx context.Context, id string) (*Order, error) {
	o, err := c.orderRowByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &o.Order, nil
}

// pendingOrder нь хүлээгдэж буй захиалгыг нэг л байлгана (давхар дарсан ч нэг захиалга).
func (c *ClickHouse) pendingOrder(ctx context.Context, key, where string, args []any, fresh *Order) (*Order, error) {
	unlock, err := c.lock(ctx, "pend:"+key)
	if err != nil {
		return nil, err
	}
	defer unlock()
	os, err := c.ordersWhere(ctx, where+" AND status = ?", "ORDER BY created_at DESC LIMIT 1", append(args, string(OrderPending))...)
	if err != nil {
		return nil, err
	}
	if len(os) > 0 {
		o := os[0].Order
		if o.Amount != fresh.Amount { // үнэ өөрчлөгдсөн бол шинэчилнэ
			o.Amount = fresh.Amount
			if err := c.writeOrder(ctx, &o, false); err != nil {
				return nil, err
			}
		}
		return &o, nil
	}
	fresh.ID, fresh.Status, fresh.CreatedAt = NewID(), OrderPending, time.Now()
	if err := c.writeOrder(ctx, fresh, false); err != nil {
		return nil, err
	}
	return fresh, nil
}

func (c *ClickHouse) CreateOrGetPendingOrder(ctx context.Context, userID string, co *Course) (*Order, error) {
	return c.pendingOrder(ctx, userID+":c:"+co.ID, "user_id = ? AND course_id = ? AND kind = ?", []any{userID, co.ID, OrderKindCourse},
		&Order{Kind: OrderKindCourse, Title: co.Title, UserID: userID, CourseID: co.ID, TeacherID: co.TeacherID, Amount: co.Price})
}

func (c *ClickHouse) CreateOrGetPendingLessonOrder(ctx context.Context, userID string, co *Course, l *Lesson) (*Order, error) {
	return c.pendingOrder(ctx, userID+":l:"+l.ID, "user_id = ? AND lesson_id = ? AND kind = ?", []any{userID, l.ID, OrderKindLesson},
		&Order{Kind: OrderKindLesson, Title: co.Title + " — " + l.Title, UserID: userID, CourseID: co.ID, LessonID: l.ID, TeacherID: co.TeacherID, Amount: l.Price})
}

func (c *ClickHouse) CreateOrGetPendingLateOrder(ctx context.Context, userID string, co *Course, l *Lesson, fee int64) (*Order, error) {
	return c.pendingOrder(ctx, userID+":late:"+l.ID, "user_id = ? AND lesson_id = ? AND kind = ?", []any{userID, l.ID, OrderKindLate},
		&Order{Kind: OrderKindLate, Title: co.Title + " — " + l.Title + " (хоцорсон)", UserID: userID, CourseID: co.ID, LessonID: l.ID, TeacherID: co.TeacherID, Amount: fee})
}

func (c *ClickHouse) CreateStorageOrder(ctx context.Context, userID string, mb int64, months int, amount int64) (*Order, error) {
	o := &Order{ID: NewID(), Kind: OrderKindStorage, StorageMB: mb, Months: months, UserID: userID, Amount: amount, Status: OrderPending, CreatedAt: time.Now()}
	return o, c.writeOrder(ctx, o, false)
}

// MarkOrderPaid: процесс доторх түгжээ + applied тэмдэг → давтан дуудахад (webhook давхардсан ч)
// элсэлт, багтаамж, номын эрх нэг л удаа хэрэгжинэ.
func (c *ClickHouse) MarkOrderPaid(ctx context.Context, orderID string, amount int64) (*Order, error) {
	unlock, err := c.lock(ctx, "order:"+orderID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	row, err := c.orderRowByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	o := row.Order
	if o.Status != OrderPaid {
		if o.Amount != amount {
			return nil, ErrConflict
		}
		now := time.Now()
		o.Status, o.PaidAt = OrderPaid, &now
		if err := c.writeOrder(ctx, &o, row.applied); err != nil {
			return nil, err
		}
	}
	if row.applied {
		return &o, nil
	}
	switch o.Kind {
	case OrderKindLesson:
		if err := c.insert(ctx, "lesson_access", []string{"user_id", "lesson_id", "course_id", "teacher_id", "created_at", "ver"},
			o.UserID, o.LessonID, o.CourseID, o.TeacherID, time.Now().UTC(), ver()); err != nil {
			return nil, err
		}
	case OrderKindLate: // хоцорсон шалгалт/даалгаврын эрх: явцад late_pass тэмдэг
		if err := c.SaveQuizResult(ctx, o.UserID, o.CourseID, o.LessonID, LatePassKey, true); err != nil {
			return nil, err
		}
	case OrderKindStorage:
		if err := c.applyStorage(ctx, &o); err != nil {
			return nil, err
		}
	case OrderKindBook:
		if err := c.grantBook(ctx, o.UserID, o.BookID, o.Amount); err != nil {
			return nil, err
		}
	default:
		if err := c.enroll(ctx, o.UserID, o.CourseID, o.TeacherID, *o.PaidAt); err != nil {
			return nil, err
		}
	}
	if err := c.writeOrder(ctx, &o, true); err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *ClickHouse) applyStorage(ctx context.Context, o *Order) error {
	unlock, err := c.lock(ctx, "user:"+o.UserID)
	if err != nil {
		return err
	}
	defer unlock()
	u, err := c.UserByID(ctx, o.UserID)
	if err != nil {
		return err
	}
	base := time.Now()
	if u.StorageExpiresAt != nil && u.StorageExpiresAt.After(base) {
		base = *u.StorageExpiresAt
	}
	exp := base.Add(time.Duration(o.Months) * StorageMonth)
	u.StorageExtraBytes, u.StorageExpiresAt = o.StorageMB<<20, &exp
	return c.writeUser(ctx, u, false)
}

func (c *ClickHouse) UserOrders(ctx context.Context, userID string, limit int) ([]Order, error) {
	rows, err := c.ordersWhere(ctx, "user_id = ? AND kind != ?", "ORDER BY created_at DESC LIMIT ?", userID, OrderKindStorage, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Order, len(rows))
	for i := range rows {
		out[i] = rows[i].Order
	}
	return out, nil
}

func (c *ClickHouse) TeacherSales(ctx context.Context, teacherID string, limit int) (*SalesSummary, error) {
	rows, err := c.ordersWhere(ctx, "teacher_id = ? AND status = ? AND kind != ?", "ORDER BY paid_at DESC",
		teacherID, string(OrderPaid), OrderKindStorage)
	if err != nil {
		return nil, err
	}
	sum := &SalesSummary{Recent: []Sale{}}
	var buyers []string
	for _, o := range rows {
		sum.TotalAmount += o.Amount
		sum.Count++
		buyers = append(buyers, o.UserID)
	}
	users, err := c.UsersByIDs(ctx, buyers)
	if err != nil {
		return nil, err
	}
	name := map[string]string{}
	for _, u := range users {
		name[u.ID] = u.Username
	}
	for i, o := range rows {
		if i >= limit {
			break
		}
		paid := o.CreatedAt
		if o.PaidAt != nil {
			paid = *o.PaidAt
		}
		sum.Recent = append(sum.Recent, Sale{OrderID: o.ID, CourseID: o.CourseID, CourseTitle: o.Title, BuyerUsername: name[o.UserID], Amount: o.Amount, PaidAt: paid})
	}
	return sum, nil
}
