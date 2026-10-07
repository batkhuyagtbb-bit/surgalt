package store

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// where нь ActivityFilter-ийг WHERE нөхцөл болгоно (хоосон талбар шүүхгүй).
func (f ActivityFilter) where(timeField string, withKind bool) (string, []any) {
	cond := "1 = 1"
	var args []any
	add := func(col, v string) {
		if v != "" {
			cond += " AND " + col + " = ?"
			args = append(args, v)
		}
	}
	add("teacher_id", f.TeacherID)
	add("course_id", f.CourseID)
	add("user_id", f.UserID)
	add("lesson_id", f.LessonID)
	if withKind {
		add("kind", f.Kind)
	}
	if !f.Since.IsZero() {
		cond += " AND " + timeField + " >= ?"
		args = append(args, f.Since.UTC())
	}
	return cond, args
}

func limitClause(limit int) string {
	if limit > 0 {
		return " LIMIT " + strconv.Itoa(limit)
	}
	return ""
}

// ---- шалгалт ----

const attemptCols = `id, user_id, user_name, course_id, lesson_id, teacher_id, started_at, deadline_at, finished_at, status,
	reason, score, max, pct, passed, violations, results, order_ids`

func scanAttempt(r driver.Rows) (ExamAttempt, error) {
	var a ExamAttempt
	var pct, viol int32
	var deadline, finished *time.Time
	err := r.Scan(&a.ID, &a.UserID, &a.UserName, &a.CourseID, &a.LessonID, &a.TeacherID, &a.StartedAt, &deadline, &finished,
		&a.Status, &a.Reason, &a.Score, &a.Max, &pct, &a.Passed, &viol, &a.Results, &a.Order)
	a.Pct, a.Violations, a.DeadlineAt, a.FinishedAt = int(pct), int(viol), deadline, finished
	if len(a.Results) == 0 {
		a.Results = nil
	}
	return a, err
}

func (c *ClickHouse) writeAttempt(ctx context.Context, a *ExamAttempt) error {
	results := a.Results
	if results == nil {
		results = map[string]bool{}
	}
	order := a.Order
	if order == nil {
		order = []string{}
	}
	return c.insert(ctx, "exam_attempts", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "teacher_id", "started_at",
		"deadline_at", "finished_at", "status", "reason", "score", "max", "pct", "passed", "violations", "results", "order_ids", "ver"},
		a.ID, a.UserID, a.UserName, a.CourseID, a.LessonID, a.TeacherID, a.StartedAt.UTC(), nullTime(a.DeadlineAt), nullTime(a.FinishedAt),
		a.Status, a.Reason, a.Score, a.Max, int32(a.Pct), a.Passed, int32(a.Violations), results, order, ver())
}

func (c *ClickHouse) attemptsWhere(ctx context.Context, where, order string, args ...any) ([]ExamAttempt, error) {
	out := []ExamAttempt{}
	err := c.query(ctx, "SELECT "+attemptCols+" FROM exam_attempts FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		a, err := scanAttempt(r)
		if err != nil {
			return err
		}
		out = append(out, a)
		return nil
	})
	return out, err
}

func (c *ClickHouse) CreateExamAttempt(ctx context.Context, a *ExamAttempt) error {
	a.ID = NewID()
	return c.writeAttempt(ctx, a)
}

func (c *ClickHouse) ExamAttemptByID(ctx context.Context, id string) (*ExamAttempt, error) {
	as, err := c.attemptsWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, ErrNotFound
	}
	return &as[0], nil
}

func (c *ClickHouse) ExamAttempts(ctx context.Context, f ActivityFilter) ([]ExamAttempt, error) {
	w, args := f.where("started_at", false)
	return c.attemptsWhere(ctx, w, "ORDER BY started_at DESC LIMIT 5000", args...)
}

func (c *ClickHouse) FinishExamAttempt(ctx context.Context, a *ExamAttempt) error {
	unlock, err := c.lock(ctx, "attempt:"+a.ID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.ExamAttemptByID(ctx, a.ID)
	if err != nil {
		return err
	}
	if cur.Status != AttemptActive {
		return ErrConflict
	}
	cur.Status, cur.Reason, cur.FinishedAt = a.Status, a.Reason, a.FinishedAt
	cur.Score, cur.Max, cur.Pct, cur.Passed, cur.Results = a.Score, a.Max, a.Pct, a.Passed, a.Results
	return c.writeAttempt(ctx, cur)
}

func (c *ClickHouse) AddAttemptViolation(ctx context.Context, id string) error {
	unlock, err := c.lock(ctx, "attempt:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.ExamAttemptByID(ctx, id)
	if err != nil {
		return err
	}
	cur.Violations++
	return c.writeAttempt(ctx, cur)
}

// ---- идэвхийн сесс ----

const sessionCols = `id, user_id, user_name, course_id, lesson_id, teacher_id, kind, title, started_at, last_at, active_sec,
	idle_sec, away_sec, focus_sum, focus_n, camera, ip, ended, end_reason, counts`

func scanSession(r driver.Rows) (StudySession, error) {
	var s StudySession
	var active, idle, away, focusN int32
	var counts map[string]int32
	err := r.Scan(&s.ID, &s.UserID, &s.UserName, &s.CourseID, &s.LessonID, &s.TeacherID, &s.Kind, &s.Title, &s.StartedAt, &s.LastAt,
		&active, &idle, &away, &s.FocusSum, &focusN, &s.Camera, &s.IP, &s.Ended, &s.EndReason, &counts)
	s.ActiveSec, s.IdleSec, s.AwaySec, s.FocusN = int(active), int(idle), int(away), int(focusN)
	s.Counts = map[string]int{}
	for k, v := range counts {
		s.Counts[k] = int(v)
	}
	return s, err
}

func (c *ClickHouse) writeSession(ctx context.Context, s *StudySession) error {
	counts := make(map[string]int32, len(s.Counts))
	for k, v := range s.Counts {
		counts[k] = int32(v)
	}
	return c.insert(ctx, "study_sessions", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "teacher_id", "kind", "title",
		"started_at", "last_at", "active_sec", "idle_sec", "away_sec", "focus_sum", "focus_n", "camera", "ip", "ended", "end_reason", "counts", "ver"},
		s.ID, s.UserID, s.UserName, s.CourseID, s.LessonID, s.TeacherID, s.Kind, s.Title, s.StartedAt.UTC(), s.LastAt.UTC(),
		int32(s.ActiveSec), int32(s.IdleSec), int32(s.AwaySec), s.FocusSum, int32(s.FocusN), s.Camera, s.IP, s.Ended, s.EndReason, counts, ver())
}

func (c *ClickHouse) CreateSession(ctx context.Context, s *StudySession) error {
	s.ID = NewID()
	if s.Counts == nil {
		s.Counts = map[string]int{}
	}
	return c.writeSession(ctx, s)
}

func (c *ClickHouse) SessionByID(ctx context.Context, id string) (*StudySession, error) {
	var found *StudySession
	err := c.query(ctx, "SELECT "+sessionCols+" FROM study_sessions FINAL WHERE id = ? LIMIT 1", []any{id}, func(r driver.Rows) error {
		s, err := scanSession(r)
		if err != nil {
			return err
		}
		found = &s
		return nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (c *ClickHouse) AddSessionBeat(ctx context.Context, id string, b SessionBeat) error {
	unlock, err := c.lock(ctx, "session:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	s, err := c.SessionByID(ctx, id)
	if err != nil {
		return err
	}
	s.ActiveSec += b.ActiveSec
	s.IdleSec += b.IdleSec
	s.AwaySec += b.AwaySec
	s.FocusSum += b.FocusSum
	s.FocusN += b.FocusN
	s.Camera = s.Camera || b.Camera
	for k, v := range b.Counts {
		s.Counts[k] += v
	}
	s.LastAt = time.Now()
	if b.End != "" {
		s.Ended, s.EndReason = true, b.End
	}
	return c.writeSession(ctx, s)
}

func (c *ClickHouse) Sessions(ctx context.Context, f ActivityFilter, limit int) ([]StudySession, error) {
	w, args := f.where("last_at", true)
	out := []StudySession{}
	err := c.query(ctx, "SELECT "+sessionCols+" FROM study_sessions FINAL WHERE "+w+" ORDER BY last_at DESC"+limitClause(limit), args,
		func(r driver.Rows) error {
			s, err := scanSession(r)
			if err != nil {
				return err
			}
			out = append(out, s)
			return nil
		})
	return out, err
}

// ---- үйл явдлын лог ----

func (c *ClickHouse) AddActivityEvents(ctx context.Context, evs []ActivityEvent) error {
	if len(evs) == 0 {
		return nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO activity_events (id, user_id, user_name, course_id, lesson_id, teacher_id, session_id, type, detail, at)")
	if err != nil {
		return err
	}
	for i := range evs {
		e := &evs[i]
		e.ID = NewID()
		if e.At.IsZero() {
			e.At = time.Now()
		}
		if err := b.Append(e.ID, e.UserID, e.UserName, e.CourseID, e.LessonID, e.TeacherID, e.SessionID, e.Type, e.Detail, e.At.UTC()); err != nil {
			return err
		}
	}
	return b.Send()
}

func (c *ClickHouse) ActivityEvents(ctx context.Context, f ActivityFilter, limit int) ([]ActivityEvent, error) {
	w, args := f.where("at", false)
	if !f.BeforeAt.IsZero() {
		// Параметр болгон дамжуулахад секунд хүртэл тайрагддаг тул миллисекундээр харьцуулна.
		ms := f.BeforeAt.UnixMilli()
		w += " AND (toUnixTimestamp64Milli(at) < ? OR (toUnixTimestamp64Milli(at) = ? AND id < ?))"
		args = append(args, ms, ms, f.BeforeID)
	}
	out := []ActivityEvent{}
	err := c.query(ctx, `SELECT id, user_id, user_name, course_id, lesson_id, teacher_id, session_id, type, detail, at
		FROM activity_events WHERE `+w+" ORDER BY at DESC, id DESC"+limitClause(limit), args, func(r driver.Rows) error {
		var e ActivityEvent
		if err := r.Scan(&e.ID, &e.UserID, &e.UserName, &e.CourseID, &e.LessonID, &e.TeacherID, &e.SessionID, &e.Type, &e.Detail, &e.At); err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	return out, err
}

// ---- асуултын хурд ----

func (c *ClickHouse) AddQuizLog(ctx context.Context, l QuizLog) error {
	l.ID = NewID()
	if l.At.IsZero() {
		l.At = time.Now()
	}
	return c.insert(ctx, "quiz_logs", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "teacher_id", "block_id", "question", "correct", "ms", "at"},
		l.ID, l.UserID, l.UserName, l.CourseID, l.LessonID, l.TeacherID, l.BlockID, l.Question, l.Correct, int32(l.Ms), l.At.UTC())
}

func (c *ClickHouse) QuizLogs(ctx context.Context, f ActivityFilter, limit int) ([]QuizLog, error) {
	w, args := f.where("at", false)
	out := []QuizLog{}
	err := c.query(ctx, `SELECT id, user_id, user_name, course_id, lesson_id, teacher_id, block_id, question, correct, ms, at
		FROM quiz_logs WHERE `+w+" ORDER BY at DESC, id DESC"+limitClause(limit), args, func(r driver.Rows) error {
		var q QuizLog
		var ms int32
		if err := r.Scan(&q.ID, &q.UserID, &q.UserName, &q.CourseID, &q.LessonID, &q.TeacherID, &q.BlockID, &q.Question, &q.Correct, &ms, &q.At); err != nil {
			return err
		}
		q.Ms = int(ms)
		out = append(out, q)
		return nil
	})
	return out, err
}

// ---- дүгнэлт ----

func (c *ClickHouse) SaveReflection(ctx context.Context, r *Reflection) error {
	r.ID = r.UserID + ":" + r.LessonID
	if r.At.IsZero() {
		r.At = time.Now()
	}
	return c.insert(ctx, "reflections", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "lesson", "teacher_id", "text", "words", "at", "ver"},
		r.ID, r.UserID, r.UserName, r.CourseID, r.LessonID, r.Lesson, r.TeacherID, r.Text, int32(r.Words), r.At.UTC(), ver())
}

func (c *ClickHouse) Reflections(ctx context.Context, f ActivityFilter, limit int) ([]Reflection, error) {
	w, args := f.where("at", false)
	out := []Reflection{}
	err := c.query(ctx, `SELECT id, user_id, user_name, course_id, lesson_id, lesson, teacher_id, text, words, at
		FROM reflections FINAL WHERE `+w+" ORDER BY at DESC"+limitClause(limit), args, func(r driver.Rows) error {
		var rf Reflection
		var words int32
		if err := r.Scan(&rf.ID, &rf.UserID, &rf.UserName, &rf.CourseID, &rf.LessonID, &rf.Lesson, &rf.TeacherID, &rf.Text, &words, &rf.At); err != nil {
			return err
		}
		rf.Words = int(words)
		out = append(out, rf)
		return nil
	})
	return out, err
}

// ---- видео үзэлтийн зураглал ----

const watchCols = `id, user_id, user_name, course_id, lesson_id, teacher_id, block_id, duration, buckets, at`

func scanWatch(r driver.Rows) (VideoWatch, error) {
	var v VideoWatch
	var dur int32
	var buckets map[int32]int32
	err := r.Scan(&v.ID, &v.UserID, &v.UserName, &v.CourseID, &v.LessonID, &v.TeacherID, &v.BlockID, &dur, &buckets, &v.At)
	v.Duration = int(dur)
	v.Buckets = make(map[int]int, len(buckets))
	for k, n := range buckets {
		v.Buckets[int(k)] = int(n)
	}
	return v, err
}

func (c *ClickHouse) writeWatch(ctx context.Context, v *VideoWatch) error {
	buckets := make(map[int32]int32, len(v.Buckets))
	for k, n := range v.Buckets {
		buckets[int32(k)] = int32(n)
	}
	return c.insert(ctx, "video_watches", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "teacher_id", "block_id", "duration", "buckets", "at", "ver"},
		v.ID, v.UserID, v.UserName, v.CourseID, v.LessonID, v.TeacherID, v.BlockID, int32(v.Duration), buckets, v.At.UTC(), ver())
}

func (c *ClickHouse) AddVideoWatch(ctx context.Context, v VideoWatch) error {
	id := v.UserID + ":" + v.LessonID + ":" + v.BlockID
	unlock, err := c.lock(ctx, "watch:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	var cur *VideoWatch
	err = c.query(ctx, "SELECT "+watchCols+" FROM video_watches FINAL WHERE id = ? LIMIT 1", []any{id}, func(r driver.Rows) error {
		w, err := scanWatch(r)
		if err != nil {
			return err
		}
		cur = &w
		return nil
	})
	if err != nil {
		return err
	}
	if cur == nil {
		v.ID, v.At = id, time.Now()
		if v.Buckets == nil {
			v.Buckets = map[int]int{}
		}
		return c.writeWatch(ctx, &v)
	}
	if v.Duration > cur.Duration {
		cur.Duration = v.Duration
	}
	for k, n := range v.Buckets {
		cur.Buckets[k] += n
	}
	if v.UserName != "" {
		cur.UserName = v.UserName
	}
	cur.At = time.Now()
	return c.writeWatch(ctx, cur)
}

func (c *ClickHouse) VideoWatches(ctx context.Context, f ActivityFilter) ([]VideoWatch, error) {
	w, args := f.where("at", false)
	out := []VideoWatch{}
	err := c.query(ctx, "SELECT "+watchCols+" FROM video_watches FINAL WHERE "+w, args, func(r driver.Rows) error {
		v, err := scanWatch(r)
		if err != nil {
			return err
		}
		out = append(out, v)
		return nil
	})
	return out, err
}

// ---- цэргийн цол ----

func (c *ClickHouse) SaveRankPoints(ctx context.Context, userID, courseID string, points int) (int, error) {
	if err := c.insert(ctx, "rank_points", []string{"user_id", "course_id", "points", "ver"}, userID, courseID, int32(points), ver()); err != nil {
		return 0, err
	}
	var total int64
	err := c.query(ctx, "SELECT toInt64(sum(points)) FROM rank_points FINAL WHERE user_id = ?", []any{userID}, func(r driver.Rows) error { return r.Scan(&total) })
	return int(total), err
}

func (c *ClickHouse) RankCourseIDs(ctx context.Context, userID string) ([]string, error) {
	var out []string
	err := c.query(ctx, "SELECT course_id FROM rank_points FINAL WHERE user_id = ? AND points > 0 ORDER BY course_id", []any{userID}, func(r driver.Rows) error {
		var id string
		if err := r.Scan(&id); err != nil {
			return err
		}
		out = append(out, id)
		return nil
	})
	return out, err
}

func (c *ClickHouse) UserRankLevel(ctx context.Context, userID string) (int, error) {
	var lvl int32
	err := c.query(ctx, "SELECT level FROM rank_levels FINAL WHERE user_id = ? LIMIT 1", []any{userID}, func(r driver.Rows) error { return r.Scan(&lvl) })
	return int(lvl), err
}

func (c *ClickHouse) SetUserRankLevel(ctx context.Context, userID string, level int) error {
	return c.insert(ctx, "rank_levels", []string{"user_id", "level", "awarded_at", "ver"}, userID, int32(level), time.Now().UTC(), ver())
}

// ---- даалгаврын хариу ----

const subCols = `id, user_id, user_name, course_id, lesson_id, teacher_id, text, files, submitted_at, late, score, feedback, graded_at, links, rubric`

func scanSub(r driver.Rows) (Submission, error) {
	var s Submission
	var score *int32
	var rubric string
	err := r.Scan(&s.ID, &s.UserID, &s.UserName, &s.CourseID, &s.LessonID, &s.TeacherID, &s.Text, &s.Files, &s.SubmittedAt, &s.Late, &score, &s.Feedback, &s.GradedAt, &s.Links, &rubric)
	if rubric != "" {
		_ = json.Unmarshal([]byte(rubric), &s.Rubric)
	}
	if score != nil {
		v := int(*score)
		s.Score = &v
	}
	if s.Files == nil {
		s.Files = []string{}
	}
	if s.Links == nil {
		s.Links = []string{}
	}
	return s, err
}

func (c *ClickHouse) writeSub(ctx context.Context, s *Submission) error {
	var score *int32
	if s.Score != nil {
		v := int32(*s.Score)
		score = &v
	}
	files, links := s.Files, s.Links
	if files == nil {
		files = []string{}
	}
	if links == nil {
		links = []string{}
	}
	rubric := ""
	if len(s.Rubric) > 0 {
		b, _ := json.Marshal(s.Rubric)
		rubric = string(b)
	}
	return c.insert(ctx, "submissions", []string{"id", "user_id", "user_name", "course_id", "lesson_id", "teacher_id", "text", "files",
		"submitted_at", "late", "score", "feedback", "graded_at", "ver", "links", "rubric"},
		s.ID, s.UserID, s.UserName, s.CourseID, s.LessonID, s.TeacherID, s.Text, files, s.SubmittedAt.UTC(), s.Late, score, s.Feedback, nullTime(s.GradedAt), ver(), links, rubric)
}

func (c *ClickHouse) subsWhere(ctx context.Context, where string, args ...any) ([]Submission, error) {
	out := []Submission{}
	err := c.query(ctx, "SELECT "+subCols+" FROM submissions FINAL WHERE "+where+" ORDER BY submitted_at DESC", args, func(r driver.Rows) error {
		s, err := scanSub(r)
		if err != nil {
			return err
		}
		out = append(out, s)
		return nil
	})
	return out, err
}

func (c *ClickHouse) SaveSubmission(ctx context.Context, sub *Submission) error {
	unlock, err := c.lock(ctx, "sub:"+sub.UserID+":"+sub.LessonID)
	if err != nil {
		return err
	}
	defer unlock()
	sub.ID = sub.UserID + ":" + sub.LessonID
	if cur, err := c.SubmissionFor(ctx, sub.UserID, sub.LessonID); err == nil { // дахин илгээхэд өмнөх дүн хүчингүй болно
		_ = cur
	}
	return c.writeSub(ctx, sub)
}

func (c *ClickHouse) SubmissionFor(ctx context.Context, userID, lessonID string) (*Submission, error) {
	ss, err := c.subsWhere(ctx, "user_id = ? AND lesson_id = ?", userID, lessonID)
	if err != nil {
		return nil, err
	}
	if len(ss) == 0 {
		return nil, ErrNotFound
	}
	return &ss[0], nil
}

func (c *ClickHouse) Submissions(ctx context.Context, lessonID string) ([]Submission, error) {
	return c.subsWhere(ctx, "lesson_id = ?", lessonID)
}

func (c *ClickHouse) GradeSubmission(ctx context.Context, lessonID, userID string, score int, feedback string, rubric map[string]int) error {
	unlock, err := c.lock(ctx, "sub:"+userID+":"+lessonID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.SubmissionFor(ctx, userID, lessonID)
	if err != nil {
		return err
	}
	now := time.Now()
	cur.Score, cur.Feedback, cur.GradedAt, cur.Rubric = &score, feedback, &now, rubric
	return c.writeSub(ctx, cur)
}
