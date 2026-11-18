package store

import (
	"context"
	"sort"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- Бүлэг чатын харилцан үнэлгээ (peer_ratings) ----
// Сургалтын бүлэг чатад суралцагч ангийн найз бүрээ «хамтарч ажилласан» оноо 0–100-аар үнэлнэ — ерөнхийд нь (lesson_id
// "") эсвэл хичээл бүрээр; (сургалт, үнэлэгдэгч, үнэлэгч, хичээл) бүрт нэг (сүүлийн) мөр. Хэн хэдийг өгсөн нь зөвхөн
// багшид харагдана (httpapi/peer_ratings.go). Бүртгэл устгахад (rater_id, ratee_id), сургалт устгахад (course_id),
// журналын «бүрмөсөн устгах»-д (тэр суралцагчийн өгсөн ба авсан, бүх хичээлийнх) устна.

// PeerRating — нэг суралцагчийн нөгөөдөө өгсөн оноо; LessonID "" = ерөнхий (сургалтын) үнэлгээ.
//
// Өдрийн snapshot (суралцагчид харагдах оноо өдөр дотор өөрчлөгдөхгүй): мөр бүр «өнөөдрийн хилээс өмнөх төлөв»-ийг
// Snap*-д авч явна. Бичих (Set/Delete) бүрт: өмнөх мөр хилээс (cut) өмнө шинэчлэгдсэн бол түүний төлөв (устгасан бол
// «байхгүй»), эс бөгөөс (өнөөдөр аль хэдийн өөрчлөгдсөн) өмнөх мөрийн Snap* хэвээр. Устгал нь Snap*-тай tombstone
// (Deleted) болж үлдэнэ; SnapshotAt(cut) нь тэр хилийн үеийн төлөвийг буцаана.
type PeerRating struct {
	CourseID  string
	RaterID   string
	RateeID   string
	LessonID  string
	Score     int
	UpdatedAt time.Time
	Deleted   bool // устгасан (PeerRatings зөвхөн SnapOK үед буцаана; шууд харагдацад алгасна)
	SnapScore int
	SnapAt    time.Time
	SnapOK    bool
}

// SnapshotAt — cut мөчийн төлөв: оноо, тэр онооны мөч, байсан эсэх.
func (p PeerRating) SnapshotAt(cut time.Time) (int, time.Time, bool) {
	if p.UpdatedAt.Before(cut) {
		return p.Score, p.UpdatedAt, !p.Deleted
	}
	if p.SnapOK && p.SnapAt.Before(cut) {
		return p.SnapScore, p.SnapAt, true
	}
	return 0, time.Time{}, false
}

// nextPeerRating — old-ийн (байхгүй бол nil) дараах мөр: шинэ оноо/устгал, snapshot-ийг авч явна.
func nextPeerRating(old *PeerRating, courseID, raterID, rateeID, lessonID string, score int, deleted bool, now, cut time.Time) PeerRating {
	p := PeerRating{CourseID: courseID, RaterID: raterID, RateeID: rateeID, LessonID: lessonID, Score: max(0, min(100, score)),
		UpdatedAt: now, Deleted: deleted}
	if deleted {
		p.Score = 0
	}
	if old != nil {
		if old.UpdatedAt.Before(cut) {
			if !old.Deleted {
				p.SnapScore, p.SnapAt, p.SnapOK = old.Score, old.UpdatedAt, true
			}
		} else {
			p.SnapScore, p.SnapAt, p.SnapOK = old.SnapScore, old.SnapAt, old.SnapOK
		}
	}
	return p
}

type PeerRatingStore interface {
	// SetPeerRating — үнэлгээг бичнэ (байвал сольж, сүүлийнх нь үлдэнэ) ба бичсэн мөрийг (snapshot-тай) буцаана. cut —
	// суралцагчийн өдрийн snapshot-ийн хил (өнөөдөр 00:00, Улаанбаатар). score-ийг дуудагч 0..100-д, lessonID-г (хоосон
	// эсвэл сургалтын хичээл) шалгана. Нэг (сургалт, үнэлэгдэгч, үнэлэгч, хичээл)-ийн бичилтүүд дараалсан (уншаад бичнэ).
	SetPeerRating(ctx context.Context, courseID, raterID, rateeID, lessonID string, score int, cut time.Time) (PeerRating, error)
	// DeletePeerRating — тэр (хичээлийн) үнэлгээг устгана: өдрийн snapshot хэрэгтэй бол tombstone болж үлдэнэ.
	DeletePeerRating(ctx context.Context, courseID, raterID, rateeID, lessonID string, cut time.Time) error
	// PeerRatings — сургалтын бүх үнэлгээ бүх хичээлээр, (ratee, rater, lesson)-оор эрэмбэлсэн: устгаагүй мөрүүд ба
	// SnapOK-той tombstone-ууд (Deleted).
	PeerRatings(ctx context.Context, courseID string) ([]PeerRating, error)
	// PeerRaterFacts — users-ийн хувьд л (үнэлэгчид): сургалтад анх орсон мөч (элссэн эсвэл анх хичээл авсан) ба анх
	// идэвхтэй байсан мөч (сургалтад суралцах сесс эхлүүлсэн эсвэл бүлэг чатад бичсэн). teacherID — сургалтын эзэмшигч
	// (сесс, бүлэг чатыг түүгээр нь индексээр уншина). Хүснэгт бүрт нэг асуулга; мэдээлэлгүй хүн map-д орохгүй.
	PeerRaterFacts(ctx context.Context, courseID, teacherID string, users []string) (map[string]RaterFacts, error)
}

// RaterFacts — үнэлгээ тоологдох эсэхийн баримт (тэг = байхгүй).
type RaterFacts struct {
	JoinedAt time.Time
	ActiveAt time.Time
}

// CountsAt — t мөчид тоологдох үнэлэгч: seasoning-ээс өмнө сургалтад орсон, t-ээс өмнө идэвхтэй байсан.
func (f RaterFacts) CountsAt(t time.Time, seasoning time.Duration) bool {
	return !f.JoinedAt.IsZero() && !f.JoinedAt.After(t.Add(-seasoning)) && !f.ActiveAt.IsZero() && !f.ActiveAt.After(t)
}

// earliest — тэг биш хамгийн эрт мөч.
func earliest(cur, at time.Time) time.Time {
	if at.IsZero() || (!cur.IsZero() && !at.Before(cur)) {
		return cur
	}
	return at
}

type factsAcc map[string]RaterFacts

func (m factsAcc) join(uid string, at time.Time) {
	f := m[uid]
	f.JoinedAt = earliest(f.JoinedAt, at)
	m[uid] = f
}

func (m factsAcc) active(uid string, at time.Time) {
	f := m[uid]
	f.ActiveAt = earliest(f.ActiveAt, at)
	m[uid] = f
}

// PeerRaterFacts — enrollments/lesson_access нь ORDER BY (user_id, …), messages нь (conversation_id, …) тул анхдагч
// түлхүүрээр; study_sessions, conversations нь teacher_id bloom индексээр.
func (c *ClickHouse) PeerRaterFacts(ctx context.Context, courseID, teacherID string, users []string) (map[string]RaterFacts, error) {
	out := factsAcc{}
	if len(users) == 0 {
		return out, nil
	}
	scan := func(add func(string, time.Time)) func(r driver.Rows) error {
		return func(r driver.Rows) error {
			var uid string
			var at time.Time
			if err := r.Scan(&uid, &at); err != nil {
				return err
			}
			add(uid, at)
			return nil
		}
	}
	if err := c.query(ctx, `SELECT user_id, min(created_at) FROM (
		SELECT user_id, created_at FROM enrollments FINAL WHERE has(?, user_id) AND course_id = ?
		UNION ALL SELECT user_id, created_at FROM lesson_access FINAL WHERE has(?, user_id) AND course_id = ?) GROUP BY user_id`,
		[]any{users, courseID, users, courseID}, scan(out.join)); err != nil {
		return nil, err
	}
	if err := c.query(ctx, "SELECT user_id, min(started_at) FROM study_sessions WHERE teacher_id = ? AND course_id = ? AND has(?, user_id) GROUP BY user_id",
		[]any{teacherID, courseID, users}, scan(out.active)); err != nil {
		return nil, err
	}
	if err := c.query(ctx, `SELECT sender_id, min(created_at) FROM messages WHERE conversation_id IN (
		SELECT id FROM conversations WHERE teacher_id = ? AND kind = ? AND course_id = ?) AND has(?, sender_id) GROUP BY sender_id`,
		[]any{teacherID, ConvGroup, courseID, users}, scan(out.active)); err != nil {
		return nil, err
	}
	return out, nil
}

var peerRatingCols = []string{"course_id", "rater_id", "ratee_id", "lesson_id", "score", "updated_at", "ver", "deleted",
	"snap_score", "snap_at", "snap_ok"}

const peerRatingSelect = "rater_id, ratee_id, lesson_id, score, updated_at, deleted, snap_score, snap_at, snap_ok"

func scanPeerRating(r driver.Rows, courseID string) (PeerRating, error) {
	p := PeerRating{CourseID: courseID}
	var score, deleted, snapScore, snapOK uint8
	if err := r.Scan(&p.RaterID, &p.RateeID, &p.LessonID, &score, &p.UpdatedAt, &deleted, &snapScore, &p.SnapAt, &snapOK); err != nil {
		return p, err
	}
	p.Score, p.Deleted, p.SnapScore, p.SnapOK = int(score), deleted == 1, int(snapScore), snapOK == 1
	if !p.SnapOK {
		p.SnapAt = time.Time{}
	}
	return p, nil
}

// writePeerRating — "peer:" түгжээн дор одоогийн мөрийг (анхдагч түлхүүрээр) уншиж, snapshot-ийг авч явсан шинэ мөр бичнэ.
func (c *ClickHouse) writePeerRating(ctx context.Context, courseID, raterID, rateeID, lessonID string, score int, deleted bool, cut time.Time) (PeerRating, error) {
	unlock, err := c.lock(ctx, "peer:"+courseID+"|"+rateeID+"|"+raterID+"|"+lessonID)
	if err != nil {
		return PeerRating{}, err
	}
	defer unlock()
	var old *PeerRating
	if err := c.query(ctx, "SELECT "+peerRatingSelect+" FROM peer_ratings FINAL WHERE course_id = ? AND ratee_id = ? AND rater_id = ? AND lesson_id = ? LIMIT 1",
		[]any{courseID, rateeID, raterID, lessonID}, func(r driver.Rows) error {
			p, err := scanPeerRating(r, courseID)
			old = &p
			return err
		}); err != nil {
		return PeerRating{}, err
	}
	p := nextPeerRating(old, courseID, raterID, rateeID, lessonID, score, deleted, time.Now().UTC(), cut)
	snapAt, snapOK, del := time.Unix(0, 0).UTC(), uint8(0), uint8(0)
	if p.SnapOK {
		snapAt, snapOK = p.SnapAt.UTC(), 1
	}
	if p.Deleted {
		del = 1
	}
	return p, c.insert(ctx, "peer_ratings", peerRatingCols,
		courseID, raterID, rateeID, lessonID, uint8(p.Score), p.UpdatedAt, ver(), del, uint8(p.SnapScore), snapAt, snapOK)
}

func (c *ClickHouse) SetPeerRating(ctx context.Context, courseID, raterID, rateeID, lessonID string, score int, cut time.Time) (PeerRating, error) {
	return c.writePeerRating(ctx, courseID, raterID, rateeID, lessonID, score, false, cut)
}

// DeletePeerRating — deleted=1 мөр (ReplacingMergeTree: шинэ ver нь хуучныг дарна), өдрийн snapshot-тай.
func (c *ClickHouse) DeletePeerRating(ctx context.Context, courseID, raterID, rateeID, lessonID string, cut time.Time) error {
	_, err := c.writePeerRating(ctx, courseID, raterID, rateeID, lessonID, 0, true, cut)
	return err
}

func (c *ClickHouse) PeerRatings(ctx context.Context, courseID string) ([]PeerRating, error) {
	out := []PeerRating{}
	err := c.query(ctx, "SELECT "+peerRatingSelect+" FROM peer_ratings FINAL WHERE course_id = ? AND (deleted = 0 OR snap_ok = 1) ORDER BY ratee_id, rater_id, lesson_id",
		[]any{courseID}, func(r driver.Rows) error {
			p, err := scanPeerRating(r, courseID)
			if err != nil {
				return err
			}
			out = append(out, p)
			return nil
		})
	return out, err
}

// peerKey — санах ойн түлхүүр: (сургалт, үнэлэгдэгч, үнэлэгч, хичээл).
type peerKey [4]string

func (m *Memory) SetPeerRating(_ context.Context, courseID, raterID, rateeID, lessonID string, score int, cut time.Time) (PeerRating, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.peers == nil {
		m.peers = map[peerKey]PeerRating{}
	}
	k := peerKey{courseID, rateeID, raterID, lessonID}
	var old *PeerRating
	if cur, ok := m.peers[k]; ok {
		old = &cur
	}
	p := nextPeerRating(old, courseID, raterID, rateeID, lessonID, score, false, time.Now().UTC(), cut)
	m.peers[k] = p
	return p, nil
}

// DeletePeerRating — snapshot хэрэггүй бол мөрийг бүр мөсөн хасна (ClickHouse-ийн snap_ok=0 tombstone-той адил: уншигдахгүй).
func (m *Memory) DeletePeerRating(_ context.Context, courseID, raterID, rateeID, lessonID string, cut time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := peerKey{courseID, rateeID, raterID, lessonID}
	cur, ok := m.peers[k]
	if !ok {
		return nil
	}
	p := nextPeerRating(&cur, courseID, raterID, rateeID, lessonID, 0, true, time.Now().UTC(), cut)
	if p.SnapOK {
		m.peers[k] = p
	} else {
		delete(m.peers, k)
	}
	return nil
}

func (m *Memory) PeerRatings(_ context.Context, courseID string) ([]PeerRating, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []PeerRating{}
	for k, p := range m.peers {
		if k[0] == courseID && (!p.Deleted || p.SnapOK) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.RateeID != b.RateeID {
			return a.RateeID < b.RateeID
		}
		if a.RaterID != b.RaterID {
			return a.RaterID < b.RaterID
		}
		return a.LessonID < b.LessonID
	})
	return out, nil
}

func (m *Memory) PeerRaterFacts(_ context.Context, courseID, teacherID string, users []string) (map[string]RaterFacts, error) {
	out := factsAcc{}
	want := make(map[string]bool, len(users))
	for _, u := range users {
		want[u] = true
	}
	m.mu.RLock()
	for k, at := range m.enrollments {
		if k[1] == courseID && want[k[0]] {
			out.join(k[0], at)
		}
	}
	for _, o := range m.orders { // хичээлийн эрх (lessonAcc) зөвхөн төлсөн захиалгаас үүсдэг
		if o.Kind == OrderKindLesson && o.Status == OrderPaid && o.CourseID == courseID && want[o.UserID] && o.PaidAt != nil &&
			m.lessonAcc[[2]string{o.UserID, o.LessonID}] == courseID {
			out.join(o.UserID, *o.PaidAt)
		}
	}
	for _, cv := range m.convs {
		if cv.Kind == ConvGroup && cv.CourseID == courseID {
			for _, msg := range m.messages[cv.ID] {
				if want[msg.SenderID] {
					out.active(msg.SenderID, msg.CreatedAt)
				}
			}
		}
	}
	m.mu.RUnlock()
	l := &m.learn
	l.mu.RLock()
	for _, s := range l.sessions {
		if s.CourseID == courseID && want[s.UserID] {
			out.active(s.UserID, s.StartedAt)
		}
	}
	l.mu.RUnlock()
	return out, nil
}

// SetEnrolledAt — тестэд: элссэн мөчийг өөрчилнө (харилцан үнэлгээ тоологдох 3 хоногийг дуурайна).
func (m *Memory) SetEnrolledAt(userID, courseID string, at time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{userID, courseID}
	if _, ok := m.enrollments[k]; !ok {
		return false
	}
	m.enrollments[k] = at
	return true
}

// SetPeerRatingAt — тестэд: үнэлгээний updated_at-ийг өөрчилнө (өдрийн snapshot-ийг дуурайна).
func (m *Memory) SetPeerRatingAt(courseID, raterID, rateeID, lessonID string, at time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := peerKey{courseID, rateeID, raterID, lessonID}
	p, ok := m.peers[k]
	if ok {
		p.UpdatedAt = at
		m.peers[k] = p
	}
	return ok
}

// ShiftPeerRatings — тестэд: сургалтын бүх үнэлгээний (tombstone-ийн ч) updated_at, snap_at-ийг by-оор хойш шилжүүлнэ
// (хоног өнгөрсөн мэт: өнөөдрийн өөрчлөлт «өчигдрийнх» болно).
func (m *Memory) ShiftPeerRatings(courseID string, by time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, p := range m.peers {
		if k[0] == courseID {
			p.UpdatedAt = p.UpdatedAt.Add(-by)
			if p.SnapOK {
				p.SnapAt = p.SnapAt.Add(-by)
			}
			m.peers[k] = p
		}
	}
}

// deletePeersLocked — m.mu бичих түгжээтэй үед: drop(k) үнэн мөрүүдийг устгаж, тоог буцаана.
func (m *Memory) deletePeersLocked(drop func(k peerKey) bool) int {
	n := 0
	for k := range m.peers {
		if drop(k) {
			delete(m.peers, k)
			n++
		}
	}
	return n
}
