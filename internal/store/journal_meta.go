package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- Журналын нэмэлт мэдээлэл (journal_meta.go) ----
// Багш сургалтынхаа журналд суралцагч бүрт өгсөн нэр (alias), бүлэг, архивын төлөв, өгөгдөл устгасан тэмдэг.
// (сургалт, суралцагч) бүрт нэг мөр. Бүх талбар хоосон мөр = «мэдээлэлгүй» (тусдаа deleted багана байхгүй).
// Суралцагчийн бүртгэлийн нэр (DisplayName) хэзээ ч өөрчлөгдөхгүй; alias зөвхөн тухайн сургалтын журналд харагдана.

// CourseStudentMeta — нэг сургалт дахь нэг суралцагчийн журналын мэдээлэл.
type CourseStudentMeta struct {
	UserID     string     `json:"user_id"`
	Alias      string     `json:"alias"`
	Group      string     `json:"group"`
	Archived   bool       `json:"archived"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	ResetAt    *time.Time `json:"reset_at,omitempty"`
	ResetBy    string     `json:"-"`
}

// empty — уншихад алгасах «мэдээлэлгүй» мөр.
func (m CourseStudentMeta) empty() bool {
	return m.Alias == "" && m.Group == "" && !m.Archived && m.ResetAt == nil
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func (m CourseStudentMeta) equal(o CourseStudentMeta) bool {
	return m.UserID == o.UserID && m.Alias == o.Alias && m.Group == o.Group && m.Archived == o.Archived &&
		sameTime(m.ArchivedAt, o.ArchivedAt) && sameTime(m.ResetAt, o.ResetAt) && m.ResetBy == o.ResetBy
}

// clone — заагч талбаруудыг тусад нь хуулна (дуудагч өөрчилсөн ч санах ойн store хөндөгдөхгүй).
func (m CourseStudentMeta) clone() CourseStudentMeta {
	if m.ArchivedAt != nil {
		t := *m.ArchivedAt
		m.ArchivedAt = &t
	}
	if m.ResetAt != nil {
		t := *m.ResetAt
		m.ResetAt = &t
	}
	return m
}

type JournalMetaStore interface {
	// CourseStudentMetas — сургалтын хоосон биш мөрүүд: user_id → мэдээлэл.
	CourseStudentMetas(ctx context.Context, courseID string) (map[string]CourseStudentMeta, error)
	// UpdateCourseStudentMeta — "cmeta:"+courseID түгжээ авч, сургалтын бүх мөрийг уншаад fn(cur, all)-ийг дуудна;
	// cur өөрчлөгдсөн бол л бичнэ. fn-ийн алдааг өөрчлөлтгүй буцааж, юу ч бичихгүй (UpdateUser-ийн загвар).
	// all нь дуудагчийн хувийн хуулбар (fn хадгалж болно); cur-ийн хуучин утга all дотор байна.
	UpdateCourseStudentMeta(ctx context.Context, courseID, userID, by string,
		fn func(cur *CourseStudentMeta, all map[string]CourseStudentMeta) error) (*CourseStudentMeta, error)
	// RenameCourseGroup — мөн тэр түгжээ. Бүлэг нь from-той (том жижиг үсэг үл харгалзан) таарах бүх мөрийг (архивласныг
	// ч) to болгож НЭГ INSERT-ээр бичнэ ("" бол бүлгийг тараана). to нь өөр байгаа бүлэгтэй таарвал нэгтгэж, тэр бүлгийн
	// бичлэгийг хэвээр үлдээнэ; зөвхөн from-ийн том жижиг үсгийг сольж буй бол to-г бичсэнээр нь. Өөрчлөгдсөн мөрийн
	// тоог буцаана; таарах мөр огт байхгүй бол ErrNotFound.
	RenameCourseGroup(ctx context.Context, courseID, from, to, by string) (changed int, err error)
	// ResetCourseStudent — суралцагчийн энэ сургалт дахь суралцахуйн өгөгдлийг бүрмөсөн устгана (course_reset.go).
	// Түгжээн дор шалгахад архивлаагүй бол ErrConflict, ResetCooldown-оос дотор өмнө нь устгасан бол ErrResetTooSoon.
	// Дараа нь мөр alias="", grp="", archived=false, reset_at=одоо, reset_by=by болно. Элсэлт, худалдан авалт, захиалга,
	// хичээлийн эрх, нууц үгээр нээсэн, шууд хичээлийн эрх хэвээр; төлсөн хоцролт/оролцооны эрхийг (late_pass, fee_pass)
	// устгал эхэлмэгц ямар ч үр дүнтэй (алдаа, хугацаа) заавал сэргээнэ — хичээлийг «үзээгүй» хэвээр (GrantLessonPass).
	// Устгал эхэлсний дараах алдаа ErrResetPartial-ийг агуулна (суралцагч архивтаа үлдэж, дахин оролдоход аюулгүй).
	ResetCourseStudent(ctx context.Context, courseID, userID, by string) (ResetResult, error)
}

// ResetResult — амжилттай устгалын үр дүн.
type ResetResult struct {
	At    time.Time // reset_at
	Wiped bool      // устгах өгөгдөл байсан (false бол зөвхөн мөрийг шинэчилсэн: суралцагчид мэдэгдэх шаардлагагүй)
}

var (
	// ErrResetPartial — устгал эхэлсний дараа алдаа гарсан: өгөгдлийн нэг хэсэг устсан байж болно.
	ErrResetPartial = errors.New("суралцагчийн өгөгдлийн устгал бүрэн дуусаагүй")
	// ErrResetTooSoon — энэ (сургалт, суралцагч)-ийн өгөгдлийг ResetCooldown-оос дотор устгасан.
	ErrResetTooSoon = errors.New("суралцагчийн өгөгдлийг саяхан устгасан")
	// ErrResetNoTime — хүсэлтийн үлдсэн хугацаа устгалд хүрэлцэхгүй: юу ч устгаагүй, дахин оролдоно.
	ErrResetNoTime = errors.New("устгалд хангалттай хугацаа үлдсэнгүй")
)

// ResetCooldown — нэг (сургалт, суралцагч)-ийн өгөгдлийг дахин устгахын өмнөх хүлээлт (архивлаад устгахыг давтах
// ачааллаас хамгаална).
const ResetCooldown = 10 * time.Minute

// resetAllowed — түгжээн доорх урьдчилсан нөхцөл: архивласан, саяхан устгаагүй.
func resetAllowed(cur CourseStudentMeta, ok bool, now time.Time) error {
	if !ok || !cur.Archived {
		return ErrConflict
	}
	if cur.ResetAt != nil && now.Sub(*cur.ResetAt) < ResetCooldown {
		return ErrResetTooSoon
	}
	return nil
}

// renameTarget — RenameCourseGroup-ийн шинэ нэр: to нь from-оос өөр байгаа бүлэгтэй таарвал тэр бүлгийн бичлэг.
func renameTarget(all map[string]CourseStudentMeta, from, to string) string {
	if to == "" || strings.EqualFold(to, from) {
		return to
	}
	for _, m := range all {
		if m.Group != "" && !strings.EqualFold(m.Group, from) && strings.EqualFold(m.Group, to) {
			return m.Group
		}
	}
	return to
}

// renameRows — from бүлгийн мөрүүдийг target болгосон хуулбарууд (өөрчлөгдөхгүйг алгасна) ба таарсан мөрийн тоо.
func renameRows(all map[string]CourseStudentMeta, from, target string) (rows []CourseStudentMeta, matched int) {
	for _, m := range all {
		if m.Group == "" || !strings.EqualFold(m.Group, from) {
			continue
		}
		matched++
		if m.Group != target {
			m.Group = target
			rows = append(rows, m)
		}
	}
	return rows, matched
}

// resetMeta — устгалын дараах мөр: нэр, бүлэг, архив арилж, устгасан тэмдэг үлдэнэ.
func resetMeta(userID, by string, at time.Time) CourseStudentMeta {
	at = at.UTC()
	return CourseStudentMeta{UserID: userID, ResetAt: &at, ResetBy: by}
}

// ---- ClickHouse ----

const cmetaCols = "user_id, alias, grp, archived, archived_at, reset_at, reset_by"

func (c *ClickHouse) CourseStudentMetas(ctx context.Context, courseID string) (map[string]CourseStudentMeta, error) {
	out := map[string]CourseStudentMeta{}
	err := c.query(ctx, "SELECT "+cmetaCols+" FROM course_student_meta FINAL WHERE course_id = ?", []any{courseID}, func(r driver.Rows) error {
		var m CourseStudentMeta
		if err := r.Scan(&m.UserID, &m.Alias, &m.Group, &m.Archived, &m.ArchivedAt, &m.ResetAt, &m.ResetBy); err != nil {
			return err
		}
		if !m.empty() {
			out[m.UserID] = m
		}
		return nil
	})
	return out, err
}

// writeMetas — бүтэн мөрүүдийг нэг INSERT-ээр (шинэ ver).
func (c *ClickHouse) writeMetas(ctx context.Context, courseID, by string, ms []CourseStudentMeta) error {
	if len(ms) == 0 {
		return nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO course_student_meta (course_id, user_id, alias, grp, archived, archived_at, reset_at, reset_by, updated_by, updated_at, ver)")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, m := range ms {
		if err := b.Append(courseID, m.UserID, m.Alias, m.Group, m.Archived, nullTime(m.ArchivedAt), nullTime(m.ResetAt), m.ResetBy, by, now, ver()); err != nil {
			return err
		}
	}
	return b.Send()
}

func (c *ClickHouse) UpdateCourseStudentMeta(ctx context.Context, courseID, userID, by string,
	fn func(cur *CourseStudentMeta, all map[string]CourseStudentMeta) error) (*CourseStudentMeta, error) {
	unlock, err := c.lock(ctx, "cmeta:"+courseID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	all, err := c.CourseStudentMetas(ctx, courseID)
	if err != nil {
		return nil, err
	}
	before := all[userID].clone()
	before.UserID = userID
	cur := before.clone()
	if err := fn(&cur, all); err != nil {
		return nil, err
	}
	cur.UserID = userID
	if !cur.equal(before) {
		if err := c.writeMetas(ctx, courseID, by, []CourseStudentMeta{cur}); err != nil {
			return nil, err
		}
	}
	return &cur, nil
}

func (c *ClickHouse) RenameCourseGroup(ctx context.Context, courseID, from, to, by string) (int, error) {
	unlock, err := c.lock(ctx, "cmeta:"+courseID)
	if err != nil {
		return 0, err
	}
	defer unlock()
	all, err := c.CourseStudentMetas(ctx, courseID)
	if err != nil {
		return 0, err
	}
	rows, matched := renameRows(all, from, renameTarget(all, from, to))
	if matched == 0 {
		return 0, ErrNotFound
	}
	if err := c.writeMetas(ctx, courseID, by, rows); err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ---- Memory ----

func (m *Memory) CourseStudentMetas(_ context.Context, courseID string) (map[string]CourseStudentMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.courseMetasLocked(courseID), nil
}

// courseMetasLocked — m.mu түгжээтэй үед: сургалтын хоосон биш мөрүүдийн хуулбар.
func (m *Memory) courseMetasLocked(courseID string) map[string]CourseStudentMeta {
	out := map[string]CourseStudentMeta{}
	for k, v := range m.cmeta {
		if k[0] == courseID && !v.empty() {
			out[k[1]] = v.clone()
		}
	}
	return out
}

func (m *Memory) putMetaLocked(courseID string, v CourseStudentMeta) {
	if m.cmeta == nil {
		m.cmeta = map[[2]string]CourseStudentMeta{}
	}
	m.cmeta[[2]string{courseID, v.UserID}] = v.clone()
}

func (m *Memory) UpdateCourseStudentMeta(_ context.Context, courseID, userID, by string,
	fn func(cur *CourseStudentMeta, all map[string]CourseStudentMeta) error) (*CourseStudentMeta, error) {
	m.cmetaMu.Lock() // сургалтын "cmeta:" түгжээтэй адил: уншаад бичих хооронд өөр өөрчлөлт орохгүй
	defer m.cmetaMu.Unlock()
	m.mu.RLock()
	all := m.courseMetasLocked(courseID)
	m.mu.RUnlock()
	before := all[userID].clone()
	before.UserID = userID
	cur := before.clone()
	if err := fn(&cur, all); err != nil {
		return nil, err
	}
	cur.UserID = userID
	if !cur.equal(before) {
		m.mu.Lock()
		m.putMetaLocked(courseID, cur)
		m.mu.Unlock()
	}
	out := cur.clone()
	return &out, nil
}

func (m *Memory) RenameCourseGroup(_ context.Context, courseID, from, to, by string) (int, error) {
	m.cmetaMu.Lock()
	defer m.cmetaMu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.courseMetasLocked(courseID)
	rows, matched := renameRows(all, from, renameTarget(all, from, to))
	if matched == 0 {
		return 0, ErrNotFound
	}
	for _, r := range rows {
		m.putMetaLocked(courseID, r)
	}
	return len(rows), nil
}
