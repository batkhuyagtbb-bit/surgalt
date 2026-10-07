package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouse — Store-ийн ClickHouse хэрэгжилт (native протокол, clickhouse-go/v2).
//
// ClickHouse бол OLAP сан: транзакц, unique constraint, шуурхай UPDATE байхгүй.
// Тиймээс:
//   - Өөрчлөгддөг бүх entity нь ReplacingMergeTree(ver): "засвар" = шинэ ver-тэй бүтэн мөр
//     нэмэх, уншихдаа FINAL (сүүлийн хувилбар). Устгал = deleted=true мөр.
//   - Зөвхөн нэмэгддэг өгөгдөл (мессеж, мэдэгдэл, лог, үйл явдал) нь энгийн MergeTree.
//   - Тоолуур (үзэлт, борлуулалт) нь SummingMergeTree "counters" хүснэгт: нэмэх = мөр нэмэх,
//     унших = sum().
//   - Имэйл/нэрний давхардалгүй байдал, "яг нэг удаа" төлбөр, pending захиалгын дедуп зэргийг
//     Locker-ээр хангана: нэг хуулбарт процесс доторх мутекс, олон хуулбарт ClickHouse Keeper
//     дээрх түгжээ (locker.go). Имэйлийн давхардалд нэмээд бичсэний дараах шалгалт (CreateUser) бий.
type ClickHouse struct {
	conn   driver.Conn
	locker Locker // процесс доторх эсвэл Keeper дээрх түгжээ (олон хуулбарт)
}

type ClickHouseOptions struct {
	// DSN: clickhouse://user:pass@host:9000/surgalt?dial_timeout=5s
	// (ClickHouse-ийн тохиргоог query параметрээр дамжуулж болно, ж: &async_insert=1&wait_for_async_insert=1)
	DSN string
	// Locker: nil бол процесс доторх (нэг хуулбар). Олон хуулбарт NewKeeperLocker өгнө.
	Locker Locker
}

func NewClickHouse(ctx context.Context, o ClickHouseOptions) (*ClickHouse, error) {
	opts, err := clickhouse.ParseDSN(o.DSN)
	if err != nil {
		return nil, fmt.Errorf("clickhouse dsn: %w", err)
	}
	db := opts.Auth.Database
	if db == "" {
		db = "surgalt"
	}
	// Эхлээд "default" сан руу холбогдож өөрийн санг үүсгэнэ.
	boot := *opts
	boot.Auth.Database = "default"
	bc, err := clickhouse.Open(&boot)
	if err != nil {
		return nil, fmt.Errorf("clickhouse connect: %w", err)
	}
	if err := bc.Ping(ctx); err != nil {
		_ = bc.Close()
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	if err := bc.Exec(ctx, "CREATE DATABASE IF NOT EXISTS "+quoteIdent(db)); err != nil {
		_ = bc.Close()
		return nil, fmt.Errorf("clickhouse create database: %w", err)
	}
	_ = bc.Close()

	opts.Auth.Database = db
	// Жижиг, олон INSERT-ийг ClickHouse өөрөө багцалж нэг part болгоно (async insert): мянга мянган суралцагчийн
	// beat/явц бичилтэд "too many parts"-гүй, дамжуулалт олон дахин өснө. wait_for_async_insert=1 тул
	// хариу ирэхэд өгөгдөл уншигдах боломжтой (бичсэний дараах шалгалтууд хэвээр). DSN-д өгвөл тэр нь давамгайлна.
	if opts.Settings == nil {
		opts.Settings = clickhouse.Settings{}
	}
	for k, v := range map[string]any{"async_insert": 1, "wait_for_async_insert": 1, "async_insert_busy_timeout_ms": 50, "async_insert_max_data_size": 1_000_000} {
		if _, ok := opts.Settings[k]; !ok {
			opts.Settings[k] = v
		}
	}
	if opts.MaxOpenConns == 0 {
		opts.MaxOpenConns = 64
	}
	if opts.MaxIdleConns == 0 {
		opts.MaxIdleConns = opts.MaxOpenConns // холболтыг нээж/хаахгүй дахин ашиглана (TIME_WAIT порт дуусахаас сэргийлнэ)
	}
	if opts.ConnMaxLifetime == 0 {
		opts.ConnMaxLifetime = time.Hour
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse connect: %w", err)
	}
	c := &ClickHouse{conn: conn, locker: o.Locker}
	if c.locker == nil {
		c.locker = &LocalLocker{}
	}
	if err := c.ensureSchema(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *ClickHouse) Close() { _ = c.conn.Close() }

// DropAll нь тестэд: бүх хүснэгтийг устгана.
func (c *ClickHouse) DropAll(ctx context.Context) error {
	for _, t := range chTables {
		if err := c.conn.Exec(ctx, "DROP TABLE IF EXISTS "+t.name); err != nil {
			return err
		}
	}
	return nil
}

// DropDatabase нь тестэд: өөрийн санг бүхэлд нь устгана.
func (c *ClickHouse) DropDatabase(ctx context.Context, name string) error {
	return c.conn.Exec(ctx, "DROP DATABASE IF EXISTS "+quoteIdent(name))
}

func quoteIdent(s string) string { return "`" + strings.ReplaceAll(s, "`", "") + "`" }

// ver — ReplacingMergeTree-ийн хувилбар: наносекунд. Нэг процесст өсөх дараалалтай.
var verMu sync.Mutex
var lastVer uint64

func ver() uint64 {
	verMu.Lock()
	defer verMu.Unlock()
	v := uint64(time.Now().UnixNano())
	if v <= lastVer {
		v = lastVer + 1
	}
	lastVer = v
	return v
}

// lock нь түлхүүрээр түгжээ авна (Locker-ээс хамаарч процесс доторх эсвэл Keeper); буцаасан функцээр тавина.
func (c *ClickHouse) lock(ctx context.Context, key string) (func(), error) {
	return c.locker.Lock(ctx, key)
}

// ---- схем ----

type chTable struct{ name, ddl string }

const tsType = "DateTime64(3, 'UTC')"

// chMigrations — өмнөх хувилбарын хүснэгтэд багана нэмэх ALTER-ууд (IF NOT EXISTS тул давтахад аюулгүй).
var chMigrations = []string{
	"ALTER TABLE lesson_progress ADD COLUMN IF NOT EXISTS quiz_done_at Nullable(" + tsType + ")",
	"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS assignment String DEFAULT ''",
	"ALTER TABLE submissions ADD COLUMN IF NOT EXISTS links Array(String)",
	"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS discussion Bool DEFAULT true",
	"ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to String DEFAULT ''",
	"ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_body String DEFAULT ''",
	"ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_name String DEFAULT ''",
	"ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment String DEFAULT ''",
	"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS unlock_rule String DEFAULT ''",
	"ALTER TABLE courses ADD COLUMN IF NOT EXISTS max_warnings Int32 DEFAULT 0",
	"ALTER TABLE courses ADD COLUMN IF NOT EXISTS block_hours Int32 DEFAULT 0",
	"ALTER TABLE courses ADD COLUMN IF NOT EXISTS block_minutes Int32 DEFAULT 0",
	"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS hidden Bool DEFAULT false",
	"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS price Int64 DEFAULT 0",
	"ALTER TABLE meetings ADD COLUMN IF NOT EXISTS members_free Bool DEFAULT false",
	"ALTER TABLE orders ADD COLUMN IF NOT EXISTS meeting_id String DEFAULT ''",
}

var chTables = []chTable{
	{"users", `(
		id String, username String, email String, password_hash String, role String,
		display_name String, headline String, bio String, avatar_url String, cover_url String,
		subjects Array(String), location String, links Map(String, String),
		created_at ` + tsType + `, storage_extra_bytes Int64, storage_expires_at Nullable(` + tsType + `),
		google_token String, ver UInt64, deleted Bool DEFAULT false,
		INDEX ix_email email TYPE bloom_filter GRANULARITY 1,
		INDEX ix_username username TYPE bloom_filter GRANULARITY 1
	) ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"identities", `(provider String, subject String, user_id String, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (provider, subject)`},
	// Тоолуур: мөр нэмэх = нэмэгдүүлэх; унших = sum(). entity: user | course | book
	{"counters", `(entity String, id String, views Int64, sales Int64, revenue Int64,
		previews Int64, reads Int64, interest Int64)
	ENGINE = SummingMergeTree ORDER BY (entity, id)`},
	{"notifications", `(id String, user_id String, type String, title String, body String, link String,
		count Int64, created_at ` + tsType + `)
	ENGINE = MergeTree ORDER BY (user_id, id) TTL toDateTime(created_at) + INTERVAL 90 DAY`},
	// Уншсан тэмдэглэгээ: user_id -> уншсан хамгийн сүүлийн мэдэгдлийн id (ID цаг хугацаагаар өсдөг).
	{"notification_reads", `(user_id String, upto String, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY user_id`},
	{"meetings", `(id String, teacher_id String, course_id String, title String, starts_at ` + tsType + `,
		duration_min Int32, meet_url String, event_id String, created_at ` + tsType + `, price Int64 DEFAULT 0,
		members_free Bool DEFAULT false, ver UInt64,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"courses", `(id String, teacher_id String, title String, description String, price Int64,
		published Bool, drip Bool, unlock_all_paid Bool, camera String, certificate Bool,
		created_at ` + tsType + `, updated_at ` + tsType + `, ver UInt64, deleted Bool DEFAULT false,
		max_warnings Int32 DEFAULT 0, block_hours Int32 DEFAULT 0, block_minutes Int32 DEFAULT 0,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"lessons", `(id String, course_id String, title String, content String, video_url String,
		is_free Bool, price Int64, unlock_after_h Int32, always_open Bool, format String, mode String,
		section String, blocks String, active_min Int32, exam String, position Int32,
		created_at ` + tsType + `, ver UInt64, deleted Bool DEFAULT false, assignment String DEFAULT '', discussion Bool DEFAULT true, unlock_rule String DEFAULT '',
		hidden Bool DEFAULT false)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (course_id, id)`},
	{"comments", `(id String, course_id String, lesson_id String, teacher_id String, user_id String, user_name String,
		parent_id String, body String, created_at ` + tsType + `, deleted Bool DEFAULT false, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (lesson_id, id)`},
	{"likes", `(target String, target_id String, user_id String, liked Bool, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (target, target_id, user_id)`},
	{"submissions", `(id String, user_id String, user_name String, course_id String, lesson_id String, teacher_id String,
		text String, files Array(String), submitted_at ` + tsType + `, late Bool, score Nullable(Int32), feedback String,
		graded_at Nullable(` + tsType + `), ver UInt64, links Array(String))
	ENGINE = ReplacingMergeTree(ver) ORDER BY (lesson_id, user_id)`},
	{"lesson_progress", `(user_id String, course_id String, lesson_id String, viewed_at ` + tsType + `,
		completed_at Nullable(` + tsType + `), quiz Map(String, Bool), quiz_done_at Nullable(` + tsType + `), ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, lesson_id)`},
	{"enrollments", `(user_id String, course_id String, teacher_id String, created_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, course_id)`},
	{"lesson_access", `(user_id String, lesson_id String, course_id String, teacher_id String,
		created_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, lesson_id)`},
	{"orders", `(id String, kind String, storage_mb Int64, months Int32, user_id String, course_id String,
		lesson_id String, book_id String, meeting_id String DEFAULT '', teacher_id String, title String, amount Int64, status String,
		created_at ` + tsType + `, paid_at Nullable(` + tsType + `), applied Bool DEFAULT false, ver UInt64,
		INDEX ix_user user_id TYPE bloom_filter GRANULARITY 1,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"conversations", `(id String, teacher_id String, visitor_key String, visitor_name String, user_id String,
		kind String, course_id String, last_message String, created_at ` + tsType + `,
		last_message_at ` + tsType + `, ver UInt64,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1,
		INDEX ix_user user_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"messages", `(id String, conversation_id String, teacher_id String, visitor_key String, sender String,
		sender_id String, sender_name String, body String, created_at ` + tsType + `,
		reply_to String DEFAULT '', reply_body String DEFAULT '', reply_name String DEFAULT '', attachment String DEFAULT '')
	ENGINE = MergeTree ORDER BY (conversation_id, id)`},
	{"message_edits", `(message_id String, conversation_id String, body String, deleted Bool, edited_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (conversation_id, message_id)`},
	{"conversation_members", `(conversation_id String, user_id String, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (conversation_id, user_id)`},
	{"message_reactions", `(message_id String, user_key String, name String, emoji String, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (message_id, user_key)`},
	{"conversation_reads", `(conversation_id String, user_key String, last_id String, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (conversation_id, user_key)`},
	{"rank_points", `(user_id String, course_id String, points Int32, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, course_id)`},
	{"rank_levels", `(user_id String, level Int32, awarded_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY user_id`},
	{"exam_attempts", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		teacher_id String, started_at ` + tsType + `, deadline_at Nullable(` + tsType + `),
		finished_at Nullable(` + tsType + `), status String, reason String, score Float64, max Float64,
		pct Int32, passed Bool, violations Int32, results Map(String, Bool), order_ids Array(String), ver UInt64,
		INDEX ix_user user_id TYPE bloom_filter GRANULARITY 1,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"study_sessions", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		teacher_id String, kind String, title String, started_at ` + tsType + `, last_at ` + tsType + `,
		active_sec Int32, idle_sec Int32, away_sec Int32, focus_sum Float64, focus_n Int32, camera Bool,
		ip String, ended Bool, end_reason String, counts Map(String, Int32), ver UInt64,
		INDEX ix_user user_id TYPE bloom_filter GRANULARITY 1,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"activity_events", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		teacher_id String, session_id String, type String, detail String, at ` + tsType + `)
	ENGINE = MergeTree ORDER BY (teacher_id, at, id)`},
	{"quiz_logs", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		teacher_id String, block_id String, question String, correct Bool, ms Int32, at ` + tsType + `)
	ENGINE = MergeTree ORDER BY (teacher_id, at, id)`},
	{"reflections", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		lesson String, teacher_id String, text String, words Int32, at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"video_watches", `(id String, user_id String, user_name String, course_id String, lesson_id String,
		teacher_id String, block_id String, duration Int32, buckets Map(Int32, Int32), at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"books", `(id String, teacher_id String, kind String, title String, author String, description String,
		cover_url String, price Int64, published Bool, pages Int32, preview_n Int32,
		created_at ` + tsType + `, updated_at ` + tsType + `, ver UInt64, deleted Bool DEFAULT false)
	ENGINE = ReplacingMergeTree(ver) ORDER BY id`},
	{"book_access", `(user_id String, book_id String, created_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, book_id)`},
	{"meeting_access", `(user_id String, meeting_id String, course_id String, teacher_id String, created_at ` + tsType + `, ver UInt64,
		INDEX ix_teacher teacher_id TYPE bloom_filter GRANULARITY 1)
	ENGINE = ReplacingMergeTree(ver) ORDER BY (user_id, meeting_id)`},
	{"payment_invoices", `(order_id String, provider String, invoice_id String, amount Int64, qr_text String, urls String,
		created_at ` + tsType + `, ver UInt64)
	ENGINE = ReplacingMergeTree(ver) ORDER BY order_id`},
	{"book_events", `(id String, book_id String, teacher_id String, user_id String, user_name String,
		type String, detail String, ip String, at ` + tsType + `)
	ENGINE = MergeTree ORDER BY (teacher_id, at, id)`},
}

func (c *ClickHouse) ensureSchema(ctx context.Context) error {
	for _, t := range chTables {
		if err := c.conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS "+t.name+" "+t.ddl); err != nil {
			return fmt.Errorf("clickhouse schema %s: %w", t.name, err)
		}
	}
	// Хуучин сангуудад нэмэгдсэн баганууд (шинэ санд DDL-д байгаа; IF NOT EXISTS тул давтахад аюулгүй).
	for _, alter := range chMigrations {
		if err := c.conn.Exec(ctx, alter); err != nil {
			return fmt.Errorf("clickhouse migrate: %w", err)
		}
	}
	return nil
}

// ---- туслах ----

// insert нь нэг мөрийг багцын API-аар бичнэ (массив, map, Nullable төрлийг найдвартай холбоно).
func (c *ClickHouse) insert(ctx context.Context, table string, cols []string, vals ...any) error {
	if len(cols) != len(vals) {
		return fmt.Errorf("insert %s: %d багана, %d утга", table, len(cols), len(vals))
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO "+table+" ("+strings.Join(cols, ", ")+")")
	if err != nil {
		return err
	}
	if err := b.Append(vals...); err != nil {
		return err
	}
	return b.Send()
}

// query нь мөр бүрт fn дуудна.
func (c *ClickHouse) query(ctx context.Context, q string, args []any, fn func(driver.Rows) error) error {
	rows, err := c.conn.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// count нь нэг тоон утга буцаана.
func (c *ClickHouse) count(ctx context.Context, q string, args ...any) (int64, error) {
	var n uint64
	err := c.query(ctx, q, args, func(r driver.Rows) error { return r.Scan(&n) })
	return int64(n), err
}

// nzStrings / nzLinks — nil-ийг хоосон болгоно (драйвер nil массив/map-ыг холбохгүй).
func nzStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nzLinks(v map[string]string) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	return v
}

func nullTime(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func nilIfZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// ---- тоолуур ----

func (c *ClickHouse) counterAdd(ctx context.Context, entity, id string, col string, by int64) error {
	cols := []string{"entity", "id", "views", "sales", "revenue", "previews", "reads", "interest"}
	vals := []any{entity, id, int64(0), int64(0), int64(0), int64(0), int64(0), int64(0)}
	for i, cname := range cols {
		if cname == col {
			vals[i] = by
		}
	}
	return c.insert(ctx, "counters", cols, vals...)
}

type counterRow struct{ views, sales, revenue, previews, reads, interest int64 }

func (c *ClickHouse) counters(ctx context.Context, entity string, ids []string) (map[string]counterRow, error) {
	out := map[string]counterRow{}
	if len(ids) == 0 {
		return out, nil
	}
	err := c.query(ctx, `SELECT id, sum(views), sum(sales), sum(revenue), sum(previews), sum(reads), sum(interest)
		FROM counters WHERE entity = ? AND has(?, id) GROUP BY id`, []any{entity, ids}, func(r driver.Rows) error {
		var id string
		var cr counterRow
		if err := r.Scan(&id, &cr.views, &cr.sales, &cr.revenue, &cr.previews, &cr.reads, &cr.interest); err != nil {
			return err
		}
		out[id] = cr
		return nil
	})
	return out, err
}

// ---- хэрэглэгч ----

const userCols = `id, username, email, password_hash, role, display_name, headline, bio, avatar_url, cover_url,
	subjects, location, links, created_at, storage_extra_bytes, storage_expires_at, google_token, deleted`

func scanUser(r driver.Rows) (*User, bool, error) {
	var u User
	var role string
	var deleted bool
	var exp *time.Time
	if err := r.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &role, &u.DisplayName, &u.Headline, &u.Bio,
		&u.AvatarURL, &u.CoverURL, &u.Subjects, &u.Location, &u.Links, &u.CreatedAt, &u.StorageExtraBytes, &exp,
		&u.GoogleToken, &deleted); err != nil {
		return nil, false, err
	}
	u.Role = Role(role)
	u.StorageExpiresAt = exp
	u.MeetConnected = u.GoogleToken != ""
	if u.Subjects == nil {
		u.Subjects = []string{}
	}
	if u.Links == nil {
		u.Links = map[string]string{}
	}
	return &u, deleted, nil
}

func (c *ClickHouse) writeUser(ctx context.Context, u *User, deleted bool) error {
	return c.insert(ctx, "users", []string{"id", "username", "email", "password_hash", "role", "display_name", "headline", "bio",
		"avatar_url", "cover_url", "subjects", "location", "links", "created_at", "storage_extra_bytes", "storage_expires_at",
		"google_token", "ver", "deleted"},
		u.ID, u.Username, u.Email, u.PasswordHash, string(u.Role), u.DisplayName, u.Headline, u.Bio,
		u.AvatarURL, u.CoverURL, nzStrings(u.Subjects), u.Location, nzLinks(u.Links), u.CreatedAt.UTC(), u.StorageExtraBytes,
		nullTime(u.StorageExpiresAt), u.GoogleToken, ver(), deleted)
}

func (c *ClickHouse) userWhere(ctx context.Context, where string, args ...any) (*User, error) {
	var found *User
	err := c.query(ctx, "SELECT "+userCols+" FROM users FINAL WHERE "+where+" LIMIT 1", args, func(r driver.Rows) error {
		u, deleted, err := scanUser(r)
		if err != nil {
			return err
		}
		if !deleted {
			found = u
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, ErrNotFound
	}
	cs, err := c.counters(ctx, "user", []string{found.ID})
	if err != nil {
		return nil, err
	}
	found.ProfileViews = cs[found.ID].views
	return found, nil
}

func (c *ClickHouse) CreateUser(ctx context.Context, u *User) error {
	unlock, err := c.lock(ctx, "users")
	if err != nil {
		return err
	}
	defer unlock()
	email := strings.ToLower(u.Email)
	n, err := c.count(ctx, "SELECT count() FROM users FINAL WHERE deleted = false AND (email = ? OR username = ?)", email, u.Username)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	u.ID, u.CreatedAt, u.Email = NewID(), time.Now(), email
	if err := c.writeUser(ctx, u, false); err != nil {
		return err
	}
	// Хоёр дахь хамгаалалт (түгжээ алдагдсан ч): ижил имэйл/нэртэй хэд хэдэн мөр үүссэн бол зөвхөн
	// хамгийн эрт ID-тай нь үлдэнэ — ID цаг хугацаагаар өсдөг тул бүх хуулбар ижил шийдвэрт хүрнэ.
	var first string
	if err := c.query(ctx, "SELECT min(id) FROM users FINAL WHERE deleted = false AND (email = ? OR username = ?)", []any{email, u.Username},
		func(r driver.Rows) error { return r.Scan(&first) }); err != nil {
		return err
	}
	if first != "" && first != u.ID {
		_ = c.writeUser(ctx, u, true) // өөрийн мөрөө устгасан гэж тэмдэглэнэ
		return ErrConflict
	}
	return nil
}

func (c *ClickHouse) UserByID(ctx context.Context, id string) (*User, error) {
	return c.userWhere(ctx, "id = ?", id)
}

func (c *ClickHouse) UserByEmail(ctx context.Context, email string) (*User, error) {
	return c.userWhere(ctx, "email = ?", strings.ToLower(email))
}

func (c *ClickHouse) UserByUsername(ctx context.Context, username string) (*User, error) {
	return c.userWhere(ctx, "username = ?", username)
}

func (c *ClickHouse) UpdateProfile(ctx context.Context, id string, p ProfileUpdate) error {
	unlock, err := c.lock(ctx, "user:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	u, err := c.UserByID(ctx, id)
	if err != nil {
		return err
	}
	u.DisplayName, u.Headline, u.Bio, u.AvatarURL, u.CoverURL = p.DisplayName, p.Headline, p.Bio, p.AvatarURL, p.CoverURL
	u.Subjects, u.Location, u.Links = cloneStrings(p.Subjects), p.Location, cloneLinks(p.Links)
	return c.writeUser(ctx, u, false)
}

func (c *ClickHouse) UpdateUsername(ctx context.Context, id, username string) error {
	unlock, err := c.lock(ctx, "users")
	if err != nil {
		return err
	}
	defer unlock()
	n, err := c.count(ctx, "SELECT count() FROM users FINAL WHERE deleted = false AND username = ? AND id != ?", username, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	u, err := c.UserByID(ctx, id)
	if err != nil {
		return err
	}
	u.Username = username
	return c.writeUser(ctx, u, false)
}

func (c *ClickHouse) UsersByIDs(ctx context.Context, ids []string) ([]User, error) {
	out := []User{}
	if len(ids) == 0 {
		return out, nil
	}
	err := c.query(ctx, "SELECT "+userCols+" FROM users FINAL WHERE has(?, id)", []any{ids}, func(r driver.Rows) error {
		u, deleted, err := scanUser(r)
		if err != nil {
			return err
		}
		if !deleted {
			out = append(out, *u)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	cs, err := c.counters(ctx, "user", ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].ProfileViews = cs[out[i].ID].views
	}
	return out, nil
}

func (c *ClickHouse) TeacherStudentCount(ctx context.Context, teacherID string) (int64, error) {
	return c.count(ctx, `SELECT uniqExact(user_id) FROM (
		SELECT user_id FROM enrollments FINAL WHERE teacher_id = ?
		UNION ALL SELECT user_id FROM lesson_access FINAL WHERE teacher_id = ?)`, teacherID, teacherID)
}

func (c *ClickHouse) SetGoogleToken(ctx context.Context, userID, encrypted string) error {
	unlock, err := c.lock(ctx, "user:"+userID)
	if err != nil {
		return err
	}
	defer unlock()
	u, err := c.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	u.GoogleToken = encrypted
	return c.writeUser(ctx, u, false)
}

func (c *ClickHouse) IncViews(ctx context.Context, profile map[string]int64, course map[string]int64) error {
	for id, n := range profile {
		if err := c.counterAdd(ctx, "user", id, "views", n); err != nil {
			return err
		}
	}
	for id, n := range course {
		if err := c.counterAdd(ctx, "course", id, "views", n); err != nil {
			return err
		}
	}
	return nil
}

// ---- гадаад нэвтрэлт ----

func (c *ClickHouse) UserByIdentity(ctx context.Context, provider, subject string) (*User, error) {
	var uid string
	err := c.query(ctx, "SELECT user_id FROM identities FINAL WHERE provider = ? AND subject = ? LIMIT 1", []any{provider, subject},
		func(r driver.Rows) error { return r.Scan(&uid) })
	if err != nil {
		return nil, err
	}
	if uid == "" {
		return nil, ErrNotFound
	}
	return c.UserByID(ctx, uid)
}

func (c *ClickHouse) LinkIdentity(ctx context.Context, userID, provider, subject string) error {
	return c.insert(ctx, "identities", []string{"provider", "subject", "user_id", "ver"}, provider, subject, userID, ver())
}

// ---- мэдэгдэл ----

func (c *ClickHouse) AddNotifications(ctx context.Context, ns []*Notification) error {
	if len(ns) == 0 {
		return nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO notifications (id, user_id, type, title, body, link, count, created_at)")
	if err != nil {
		return err
	}
	for _, n := range ns {
		n.ID, n.CreatedAt = NewID(), time.Now()
		if err := b.Append(n.ID, n.UserID, n.Type, n.Title, n.Body, n.Link, n.Count, n.CreatedAt.UTC()); err != nil {
			return err
		}
	}
	return b.Send()
}

func (c *ClickHouse) readUpto(ctx context.Context, userID string) (string, error) {
	var upto string
	err := c.query(ctx, "SELECT upto FROM notification_reads FINAL WHERE user_id = ? LIMIT 1", []any{userID},
		func(r driver.Rows) error { return r.Scan(&upto) })
	return upto, err
}

func (c *ClickHouse) Notifications(ctx context.Context, userID string, limit int) ([]Notification, int64, error) {
	upto, err := c.readUpto(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	out := []Notification{}
	err = c.query(ctx, `SELECT id, user_id, type, title, body, link, count, created_at FROM notifications
		WHERE user_id = ? ORDER BY id DESC LIMIT ?`, []any{userID, limit}, func(r driver.Rows) error {
		var n Notification
		if err := r.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.Count, &n.CreatedAt); err != nil {
			return err
		}
		n.Read = n.ID <= upto
		out = append(out, n)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	unread, err := c.count(ctx, "SELECT count() FROM notifications WHERE user_id = ? AND id > ?", userID, upto)
	if err != nil {
		return nil, 0, err
	}
	return out, unread, nil
}

func (c *ClickHouse) MarkNotificationsRead(ctx context.Context, userID string) error {
	var maxID string
	if err := c.query(ctx, "SELECT max(id) FROM notifications WHERE user_id = ?", []any{userID},
		func(r driver.Rows) error { return r.Scan(&maxID) }); err != nil {
		return err
	}
	if maxID == "" {
		return nil
	}
	return c.insert(ctx, "notification_reads", []string{"user_id", "upto", "ver"}, userID, maxID, ver())
}

// ---- уулзалт ----

const meetingCols = "id, teacher_id, course_id, title, starts_at, duration_min, meet_url, event_id, created_at, price, members_free"

func scanMeeting(r driver.Rows) (Meeting, error) {
	var m Meeting
	var dur int32
	err := r.Scan(&m.ID, &m.TeacherID, &m.CourseID, &m.Title, &m.StartsAt, &dur, &m.MeetURL, &m.EventID, &m.CreatedAt, &m.Price, &m.MembersFree)
	m.DurationMin = int(dur)
	return m, err
}

func (c *ClickHouse) writeMeeting(ctx context.Context, m *Meeting) error {
	return c.insert(ctx, "meetings", []string{"id", "teacher_id", "course_id", "title", "starts_at", "duration_min", "meet_url", "event_id", "created_at", "price", "members_free", "ver"},
		m.ID, m.TeacherID, m.CourseID, m.Title, m.StartsAt.UTC(), int32(m.DurationMin), m.MeetURL, m.EventID, m.CreatedAt.UTC(), m.Price, m.MembersFree, ver())
}

func (c *ClickHouse) CreateMeeting(ctx context.Context, m *Meeting) error {
	m.ID, m.CreatedAt = NewID(), time.Now()
	return c.writeMeeting(ctx, m)
}

func (c *ClickHouse) MeetingByID(ctx context.Context, id string) (*Meeting, error) {
	var out *Meeting
	err := c.query(ctx, "SELECT "+meetingCols+" FROM meetings FINAL WHERE id = ? LIMIT 1", []any{id}, func(r driver.Rows) error {
		m, err := scanMeeting(r)
		out = &m
		return err
	})
	if err == nil && out == nil {
		err = ErrNotFound
	}
	return out, err
}

func (c *ClickHouse) SetMeetingPrice(ctx context.Context, id string, price int64, membersFree bool) error {
	unlock, err := c.lock(ctx, "meeting:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := c.MeetingByID(ctx, id)
	if err != nil {
		return err
	}
	m.Price, m.MembersFree = price, membersFree
	return c.writeMeeting(ctx, m)
}

func (c *ClickHouse) CreateOrGetPendingMeetingOrder(ctx context.Context, userID string, m *Meeting, title string) (*Order, error) {
	return c.pendingOrder(ctx, userID+":m:"+m.ID, "user_id = ? AND meeting_id = ? AND kind = ?", []any{userID, m.ID, OrderKindMeeting},
		&Order{Kind: OrderKindMeeting, Title: title, UserID: userID, CourseID: m.CourseID, MeetingID: m.ID, TeacherID: m.TeacherID, Amount: m.Price})
}

// grantMeeting — захиалга төлөгдөхөд шууд хичээлд нэгдэх эрх (MarkOrderPaid-ийн түгжээн дотор, нэг л удаа).
func (c *ClickHouse) grantMeeting(ctx context.Context, o *Order) error {
	return c.insert(ctx, "meeting_access", []string{"user_id", "meeting_id", "course_id", "teacher_id", "created_at", "ver"},
		o.UserID, o.MeetingID, o.CourseID, o.TeacherID, time.Now().UTC(), ver())
}

func (c *ClickHouse) MeetingAccess(ctx context.Context, userID string) (map[string]bool, error) {
	out := map[string]bool{}
	err := c.query(ctx, "SELECT meeting_id FROM meeting_access FINAL WHERE user_id = ?", []any{userID}, func(r driver.Rows) error {
		var id string
		if err := r.Scan(&id); err != nil {
			return err
		}
		out[id] = true
		return nil
	})
	return out, err
}

func (c *ClickHouse) MeetingBuyers(ctx context.Context, teacherID string) (map[string]int, error) {
	out := map[string]int{}
	err := c.query(ctx, "SELECT meeting_id, toInt64(count()) FROM meeting_access FINAL WHERE teacher_id = ? GROUP BY meeting_id", []any{teacherID}, func(r driver.Rows) error {
		var id string
		var n int64
		if err := r.Scan(&id, &n); err != nil {
			return err
		}
		out[id] = int(n)
		return nil
	})
	return out, err
}

func (c *ClickHouse) Meetings(ctx context.Context, teacherID, courseID string, from time.Time, limit int) ([]Meeting, error) {
	out := []Meeting{}
	err := c.query(ctx, "SELECT "+meetingCols+` FROM meetings FINAL
		WHERE teacher_id = ? AND (? = '' OR course_id = ?) AND addMinutes(starts_at, duration_min) > ?
		ORDER BY starts_at LIMIT ?`, []any{teacherID, courseID, courseID, from.UTC(), limit}, func(r driver.Rows) error {
		m, err := scanMeeting(r)
		if err != nil {
			return err
		}
		out = append(out, m)
		return nil
	})
	return out, err
}

func (c *ClickHouse) MeetingsByCourses(ctx context.Context, courseIDs []string, from time.Time, limit int) ([]Meeting, error) {
	out := []Meeting{}
	if len(courseIDs) == 0 {
		return out, nil
	}
	err := c.query(ctx, "SELECT "+meetingCols+` FROM meetings FINAL
		WHERE course_id != '' AND has(?, course_id) AND addMinutes(starts_at, duration_min) > ?
		ORDER BY starts_at LIMIT ?`, []any{courseIDs, from.UTC(), limit}, func(r driver.Rows) error {
		m, err := scanMeeting(r)
		if err != nil {
			return err
		}
		out = append(out, m)
		return nil
	})
	return out, err
}
