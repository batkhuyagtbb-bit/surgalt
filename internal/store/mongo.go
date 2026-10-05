package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Mongo нь MongoDB дээрх Store. Бүх халуун асуулга индекстэй, бичилт бүр
// идемпотент (давтан илгээхэд давхардахгүй) байхаар хийгдсэн.
type Mongo struct {
	client *mongo.Client
	db     *mongo.Database

	users, identities, meetings, notifications, lessonAccess, courses, lessons, orders, enrollments, conversations, messages, progress *mongo.Collection
}

type MongoOptions struct {
	URI         string
	Database    string
	MaxPoolSize uint64
}

func NewMongo(ctx context.Context, o MongoOptions) (*Mongo, error) {
	if o.MaxPoolSize == 0 {
		o.MaxPoolSize = 200
	}
	copts := options.Client().ApplyURI(o.URI).
		SetMaxPoolSize(o.MaxPoolSize).
		SetMinPoolSize(o.MaxPoolSize / 10).
		SetMaxConnIdleTime(5 * time.Minute).
		SetServerSelectionTimeout(5 * time.Second).
		SetTimeout(10 * time.Second)
	client, err := mongo.Connect(copts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	db := client.Database(o.Database)
	m := &Mongo{
		client: client, db: db,
		users: db.Collection("users"), identities: db.Collection("identities"), meetings: db.Collection("meetings"), notifications: db.Collection("notifications"), lessonAccess: db.Collection("lesson_access"), courses: db.Collection("courses"), lessons: db.Collection("lessons"),
		orders: db.Collection("orders"), enrollments: db.Collection("enrollments"),
		conversations: db.Collection("conversations"), messages: db.Collection("messages"), progress: db.Collection("lesson_progress"),
	}
	if err := m.ensureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return m, nil
}

func (m *Mongo) Close() { _ = m.client.Disconnect(context.Background()) }

func (m *Mongo) ensureIndexes(ctx context.Context) error {
	idx := func(keys bson.D) mongo.IndexModel { return mongo.IndexModel{Keys: keys} }
	uniq := func(keys bson.D) mongo.IndexModel {
		return mongo.IndexModel{Keys: keys, Options: options.Index().SetUnique(true)}
	}
	all := map[*mongo.Collection][]mongo.IndexModel{
		m.users:      {uniq(bson.D{{Key: "username", Value: 1}}), uniq(bson.D{{Key: "email", Value: 1}})},
		m.identities: {idx(bson.D{{Key: "user_id", Value: 1}})},
		m.notifications: {
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}),
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "read", Value: 1}}),
			// 90 хоногийн дараа автоматаар устна.
			{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(90 * 24 * 3600)},
		},
		m.meetings: {
			idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "course_id", Value: 1}, {Key: "ends_at", Value: 1}}),
			idx(bson.D{{Key: "course_id", Value: 1}, {Key: "ends_at", Value: 1}}),
		},
		m.courses: {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "created_at", Value: -1}}), idx(bson.D{{Key: "published", Value: 1}, {Key: "views", Value: -1}})},
		m.lessons: {idx(bson.D{{Key: "course_id", Value: 1}, {Key: "position", Value: 1}})},
		m.orders: {
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "course_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"status": string(OrderPending), "kind": OrderKindCourse}),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "lesson_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"status": string(OrderPending), "kind": OrderKindLesson}),
			},
			idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "status", Value: 1}, {Key: "paid_at", Value: -1}}),
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}),
		},
		m.lessonAccess: {
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "course_id", Value: 1}}),
			idx(bson.D{{Key: "course_id", Value: 1}}),
		},
		m.enrollments: {
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}),
			idx(bson.D{{Key: "course_id", Value: 1}, {Key: "created_at", Value: -1}}),
			idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "user_id", Value: 1}}),
		},
		m.conversations: {
			uniq(bson.D{{Key: "teacher_id", Value: 1}, {Key: "visitor_key", Value: 1}}),
			idx(bson.D{{Key: "kind", Value: 1}, {Key: "course_id", Value: 1}}),
			idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "last_message_at", Value: -1}}),
			idx(bson.D{{Key: "user_id", Value: 1}, {Key: "last_message_at", Value: -1}}),
		},
		m.messages: {idx(bson.D{{Key: "conversation_id", Value: 1}, {Key: "_id", Value: -1}})},
		m.progress: {idx(bson.D{{Key: "user_id", Value: 1}, {Key: "course_id", Value: 1}})},
	}
	for c, ms := range learningIndexes(m) {
		all[c] = ms
	}
	for c, ms := range bookIndexes(m) {
		all[c] = ms
	}
	for c, ms := range engagementIndexes(m) {
		all[c] = ms
	}
	for coll, models := range all {
		if _, err := coll.Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("index %s: %w", coll.Name(), err)
		}
	}
	return nil
}

// ---- туслах функцууд ----

func oid(hex string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		return id, ErrNotFound
	}
	return id, nil
}

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, mongo.ErrNoDocuments):
		return ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return ErrConflict
	}
	return err
}

func hexOrEmpty(id bson.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

// ---- баримтын бүтэц ----

type userDoc struct {
	ID           bson.ObjectID     `bson:"_id"`
	Username     string            `bson:"username"`
	Email        string            `bson:"email"`
	PasswordHash string            `bson:"password_hash"`
	Role         string            `bson:"role"`
	DisplayName  string            `bson:"display_name"`
	Headline     string            `bson:"headline"`
	Bio          string            `bson:"bio"`
	AvatarURL    string            `bson:"avatar_url"`
	CoverURL     string            `bson:"cover_url,omitempty"`
	Subjects     []string          `bson:"subjects,omitempty"`
	Location     string            `bson:"location,omitempty"`
	Links        map[string]string `bson:"links,omitempty"`
	CreatedAt    time.Time         `bson:"created_at"`
	StorageExtra int64             `bson:"storage_extra_bytes"`
	StorageExp   *time.Time        `bson:"storage_expires_at,omitempty"`
	GoogleToken  string            `bson:"google_token,omitempty"`
	ProfileViews int64             `bson:"profile_views"`
}

func (d *userDoc) model() *User {
	return &User{ID: d.ID.Hex(), Username: d.Username, Email: d.Email, PasswordHash: d.PasswordHash, Role: Role(d.Role),
		DisplayName: d.DisplayName, Headline: d.Headline, Bio: d.Bio, AvatarURL: d.AvatarURL, CreatedAt: d.CreatedAt,
		CoverURL: d.CoverURL, Subjects: cloneStrings(d.Subjects), Location: d.Location, Links: cloneLinks(d.Links),
		StorageExtraBytes: d.StorageExtra, StorageExpiresAt: d.StorageExp,
		GoogleToken: d.GoogleToken, MeetConnected: d.GoogleToken != "", ProfileViews: d.ProfileViews}
}

type courseDoc struct {
	ID              bson.ObjectID `bson:"_id"`
	TeacherID       bson.ObjectID `bson:"teacher_id"`
	Title           string        `bson:"title"`
	Description     string        `bson:"description"`
	Price           int64         `bson:"price"`
	Published       bool          `bson:"published"`
	Drip            bool          `bson:"drip,omitempty"`
	UnlockAllPaid   bool          `bson:"unlock_all_paid,omitempty"`
	Camera          string        `bson:"camera,omitempty"`
	Certificate     bool          `bson:"certificate,omitempty"`
	LessonCount     int           `bson:"lesson_count"`
	FreeLessonCount int           `bson:"free_lesson_count"`
	Views           int64         `bson:"views"`
	CreatedAt       time.Time     `bson:"created_at"`
	UpdatedAt       time.Time     `bson:"updated_at"`
}

func (d *courseDoc) model() Course {
	return Course{ID: d.ID.Hex(), TeacherID: d.TeacherID.Hex(), Title: d.Title, Description: d.Description, Price: d.Price,
		Published: d.Published, Drip: d.Drip, UnlockAllPaid: d.UnlockAllPaid, Camera: d.Camera, Certificate: d.Certificate, LessonCount: d.LessonCount, FreeLessonCount: d.FreeLessonCount, Views: d.Views, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

type lessonDoc struct {
	ID           bson.ObjectID `bson:"_id"`
	CourseID     bson.ObjectID `bson:"course_id"`
	Title        string        `bson:"title"`
	Content      string        `bson:"content"`
	VideoURL     string        `bson:"video_url"`
	IsFree       bool          `bson:"is_free"`
	Price        int64         `bson:"price"`
	UnlockAfterH int           `bson:"unlock_after_h,omitempty"`
	AlwaysOpen   bool          `bson:"always_open,omitempty"`
	Format       string        `bson:"format,omitempty"`
	Mode         string        `bson:"mode,omitempty"`
	Section      string        `bson:"section,omitempty"`
	Blocks       []Block       `bson:"blocks,omitempty"`
	ActiveMin    int           `bson:"active_min,omitempty"`
	Exam         *Exam         `bson:"exam,omitempty"`
	Position     int           `bson:"position"`
	CreatedAt    time.Time     `bson:"created_at"`
}

func (d *lessonDoc) model() Lesson {
	return Lesson{ID: d.ID.Hex(), CourseID: d.CourseID.Hex(), Title: d.Title, Content: d.Content, VideoURL: d.VideoURL,
		IsFree: d.IsFree, Price: d.Price, UnlockAfterH: d.UnlockAfterH, AlwaysOpen: d.AlwaysOpen, Format: d.Format, Mode: d.Mode, Section: d.Section, Blocks: d.Blocks, ActiveMin: d.ActiveMin, Exam: d.Exam, Position: d.Position, CreatedAt: d.CreatedAt}
}

type orderDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Kind      string        `bson:"kind"`
	StorageMB int64         `bson:"storage_mb,omitempty"`
	Months    int           `bson:"months,omitempty"`
	UserID    bson.ObjectID `bson:"user_id"`
	CourseID  bson.ObjectID `bson:"course_id,omitempty"`
	LessonID  bson.ObjectID `bson:"lesson_id,omitempty"`
	BookID    string        `bson:"book_id,omitempty"`
	TeacherID bson.ObjectID `bson:"teacher_id,omitempty"`
	Title     string        `bson:"title,omitempty"`
	Amount    int64         `bson:"amount"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"created_at"`
	PaidAt    *time.Time    `bson:"paid_at,omitempty"`
}

func (d *orderDoc) model() *Order {
	return &Order{ID: d.ID.Hex(), Kind: d.Kind, StorageMB: d.StorageMB, Months: d.Months, UserID: d.UserID.Hex(),
		CourseID: hexOrEmpty(d.CourseID), LessonID: hexOrEmpty(d.LessonID), BookID: d.BookID, TeacherID: hexOrEmpty(d.TeacherID), Title: d.Title, Amount: d.Amount, Status: OrderStatus(d.Status), CreatedAt: d.CreatedAt, PaidAt: d.PaidAt}
}

type enrollmentDoc struct {
	ID        string        `bson:"_id"` // "<user>:<course>" — давхардалгүй байдлыг _id баталгаажуулна
	UserID    bson.ObjectID `bson:"user_id"`
	CourseID  bson.ObjectID `bson:"course_id"`
	TeacherID bson.ObjectID `bson:"teacher_id"`
	CreatedAt time.Time     `bson:"created_at"`
}

type convDoc struct {
	ID            bson.ObjectID `bson:"_id"`
	TeacherID     bson.ObjectID `bson:"teacher_id"`
	VisitorKey    string        `bson:"visitor_key"`
	VisitorName   string        `bson:"visitor_name"`
	UserID        bson.ObjectID `bson:"user_id,omitempty"`
	Kind          string        `bson:"kind,omitempty"`
	CourseID      bson.ObjectID `bson:"course_id,omitempty"`
	LastMessage   string        `bson:"last_message"`
	CreatedAt     time.Time     `bson:"created_at"`
	LastMessageAt time.Time     `bson:"last_message_at"`
}

func (d *convDoc) model() Conversation {
	return Conversation{ID: d.ID.Hex(), TeacherID: d.TeacherID.Hex(), VisitorKey: d.VisitorKey, VisitorName: d.VisitorName,
		UserID: hexOrEmpty(d.UserID), Kind: d.Kind, CourseID: hexOrEmpty(d.CourseID), LastMessage: d.LastMessage, CreatedAt: d.CreatedAt, LastMessageAt: d.LastMessageAt}
}

type msgDoc struct {
	ID             bson.ObjectID `bson:"_id"`
	ConversationID bson.ObjectID `bson:"conversation_id"`
	TeacherID      bson.ObjectID `bson:"teacher_id"`
	VisitorKey     string        `bson:"visitor_key"`
	Sender         string        `bson:"sender"`
	SenderID       bson.ObjectID `bson:"sender_id,omitempty"`
	SenderName     string        `bson:"sender_name,omitempty"`
	Body           string        `bson:"body"`
	CreatedAt      time.Time     `bson:"created_at"`
}

func (d *msgDoc) model() Message {
	return Message{ID: d.ID.Hex(), ConversationID: d.ConversationID.Hex(), TeacherID: d.TeacherID.Hex(),
		VisitorKey: d.VisitorKey, Sender: d.Sender, SenderID: hexOrEmpty(d.SenderID), SenderName: d.SenderName, Body: d.Body, CreatedAt: d.CreatedAt}
}

// ---- хэрэглэгч ----

func (m *Mongo) CreateUser(ctx context.Context, u *User) error {
	d := userDoc{ID: bson.NewObjectID(), Username: u.Username, Email: strings.ToLower(u.Email), PasswordHash: u.PasswordHash,
		Role: string(u.Role), DisplayName: u.DisplayName, Headline: u.Headline, Bio: u.Bio, AvatarURL: u.AvatarURL,
		CoverURL: u.CoverURL, Subjects: u.Subjects, Location: u.Location, Links: u.Links,
		CreatedAt: time.Now().UTC()}
	if _, err := m.users.InsertOne(ctx, d); err != nil {
		return mapErr(err)
	}
	*u = *d.model()
	return nil
}

func (m *Mongo) findUser(ctx context.Context, filter any) (*User, error) {
	var d userDoc
	if err := m.users.FindOne(ctx, filter).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	return d.model(), nil
}

func (m *Mongo) UserByID(ctx context.Context, id string) (*User, error) {
	o, err := oid(id)
	if err != nil {
		return nil, err
	}
	return m.findUser(ctx, bson.M{"_id": o})
}

func (m *Mongo) UserByEmail(ctx context.Context, email string) (*User, error) {
	return m.findUser(ctx, bson.M{"email": strings.ToLower(email)})
}

func (m *Mongo) UserByUsername(ctx context.Context, username string) (*User, error) {
	return m.findUser(ctx, bson.M{"username": username})
}

func (m *Mongo) UpdateProfile(ctx context.Context, id string, p ProfileUpdate) error {
	o, err := oid(id)
	if err != nil {
		return err
	}
	res, err := m.users.UpdateByID(ctx, o, bson.M{"$set": bson.M{
		"display_name": p.DisplayName, "headline": p.Headline, "bio": p.Bio, "avatar_url": p.AvatarURL, "cover_url": p.CoverURL,
		"subjects": cloneStrings(p.Subjects), "location": p.Location, "links": cloneLinks(p.Links)}})
	if err != nil {
		return mapErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) TeacherStudentCount(ctx context.Context, teacherID string) (int64, error) {
	o, err := oid(teacherID)
	if err != nil {
		return 0, err
	}
	cur, err := m.enrollments.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"teacher_id": o}}},
		{{Key: "$group", Value: bson.M{"_id": "$user_id"}}},
		{{Key: "$count", Value: "n"}},
	})
	if err != nil {
		return 0, err
	}
	var out []struct {
		N int64 `bson:"n"`
	}
	if err := cur.All(ctx, &out); err != nil || len(out) == 0 {
		return 0, err
	}
	return out[0].N, nil
}

// identities: _id = "<provider>:<subject>" тул нэг гадаад данс нэг л хэрэглэгчтэй холбогдоно.
func (m *Mongo) UserByIdentity(ctx context.Context, provider, subject string) (*User, error) {
	var d struct {
		UserID bson.ObjectID `bson:"user_id"`
	}
	if err := m.identities.FindOne(ctx, bson.M{"_id": provider + ":" + subject}).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	return m.findUser(ctx, bson.M{"_id": d.UserID})
}

func (m *Mongo) LinkIdentity(ctx context.Context, userID, provider, subject string) error {
	uid, err := oid(userID)
	if err != nil {
		return err
	}
	_, err = m.identities.InsertOne(ctx, bson.M{"_id": provider + ":" + subject, "user_id": uid, "provider": provider, "created_at": time.Now().UTC()})
	if mongo.IsDuplicateKeyError(err) {
		existing, e2 := m.UserByIdentity(ctx, provider, subject)
		if e2 == nil && existing.ID == userID {
			return nil
		}
		return ErrConflict
	}
	return err
}

// ---- сургалт ----

func (m *Mongo) CreateCourse(ctx context.Context, c *Course) error {
	tid, err := oid(c.TeacherID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	d := courseDoc{ID: bson.NewObjectID(), TeacherID: tid, Title: c.Title, Description: c.Description, Price: c.Price,
		Published: c.Published, Drip: c.Drip, UnlockAllPaid: c.UnlockAllPaid, Camera: c.Camera, Certificate: c.Certificate, CreatedAt: now, UpdatedAt: now}
	if _, err := m.courses.InsertOne(ctx, d); err != nil {
		return mapErr(err)
	}
	*c = d.model()
	return nil
}

func (m *Mongo) UpdateCourse(ctx context.Context, c *Course) error {
	id, err := oid(c.ID)
	if err != nil {
		return err
	}
	tid, err := oid(c.TeacherID)
	if err != nil {
		return err
	}
	var d courseDoc
	err = m.courses.FindOneAndUpdate(ctx, bson.M{"_id": id, "teacher_id": tid},
		bson.M{"$set": bson.M{"title": c.Title, "description": c.Description, "price": c.Price,
			"published": c.Published, "drip": c.Drip, "unlock_all_paid": c.UnlockAllPaid, "camera": c.Camera, "certificate": c.Certificate, "updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&d)
	if err != nil {
		return mapErr(err)
	}
	*c = d.model()
	return nil
}

func (m *Mongo) CourseByID(ctx context.Context, id string) (*Course, error) {
	o, err := oid(id)
	if err != nil {
		return nil, err
	}
	var d courseDoc
	if err := m.courses.FindOne(ctx, bson.M{"_id": o}).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	c := d.model()
	return &c, nil
}

func (m *Mongo) findCourses(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) ([]Course, error) {
	cur, err := m.courses.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	var docs []courseDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Course, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

func (m *Mongo) CoursesByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Course, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"teacher_id": tid}
	if onlyPublished {
		filter["published"] = true
	}
	return m.findCourses(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(500))
}

func (m *Mongo) PublishedCourses(ctx context.Context, limit int) ([]Course, error) {
	return m.findCourses(ctx, bson.M{"published": true}, options.Find().SetSort(bson.D{{Key: "views", Value: -1}}).SetLimit(int64(limit)))
}

func (m *Mongo) LessonOutlines(ctx context.Context, courseIDs []string) ([]Lesson, error) {
	ids := make([]bson.ObjectID, 0, len(courseIDs))
	for _, id := range courseIDs {
		if o, err := oid(id); err == nil {
			ids = append(ids, o)
		}
	}
	if len(ids) == 0 {
		return []Lesson{}, nil
	}
	cur, err := m.lessons.Find(ctx, bson.M{"course_id": bson.M{"$in": ids}}, options.Find().
		SetProjection(bson.M{"content": 0, "blocks": 0, "video_url": 0}).SetSort(bson.D{{Key: "position", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []lessonDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Lesson, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

// ---- хичээл ----

func (m *Mongo) CreateLesson(ctx context.Context, l *Lesson) error {
	cid, err := oid(l.CourseID)
	if err != nil {
		return err
	}
	inc := bson.M{"lesson_count": 1}
	if l.IsFree {
		inc["free_lesson_count"] = 1
	}
	// Тоолуурыг атомаар нэмэгдүүлж, дарааллын дугаарыг тэндээс авна.
	var c courseDoc
	err = m.courses.FindOneAndUpdate(ctx, bson.M{"_id": cid},
		bson.M{"$inc": inc, "$set": bson.M{"updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&c)
	if err != nil {
		return mapErr(err)
	}
	d := lessonDoc{ID: bson.NewObjectID(), CourseID: cid, Title: l.Title, Content: l.Content, VideoURL: l.VideoURL,
		IsFree: l.IsFree, Price: l.Price, UnlockAfterH: l.UnlockAfterH, AlwaysOpen: l.AlwaysOpen, Format: l.Format, Mode: l.Mode, Section: l.Section, Blocks: l.Blocks, ActiveMin: l.ActiveMin, Exam: l.Exam, Position: c.LessonCount, CreatedAt: time.Now().UTC()}
	if _, err := m.lessons.InsertOne(ctx, d); err != nil {
		for k := range inc {
			inc[k] = -1
		}
		_, _ = m.courses.UpdateByID(context.WithoutCancel(ctx), cid, bson.M{"$inc": inc})
		return mapErr(err)
	}
	*l = d.model()
	return nil
}

func (m *Mongo) LessonsByCourse(ctx context.Context, courseID string) ([]Lesson, error) {
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	cur, err := m.lessons.Find(ctx, bson.M{"course_id": cid}, options.Find().SetSort(bson.D{{Key: "position", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []lessonDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Lesson, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

func (m *Mongo) LessonByID(ctx context.Context, courseID, lessonID string) (*Lesson, error) {
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	lid, err := oid(lessonID)
	if err != nil {
		return nil, err
	}
	var d lessonDoc
	if err := m.lessons.FindOne(ctx, bson.M{"_id": lid, "course_id": cid}).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	l := d.model()
	return &l, nil
}

// ---- элсэлт ----

func (m *Mongo) IsEnrolled(ctx context.Context, userID, courseID string) (bool, error) {
	err := m.enrollments.FindOne(ctx, bson.M{"_id": userID + ":" + courseID},
		options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func (m *Mongo) enroll(ctx context.Context, userID, courseID, teacherID bson.ObjectID) error {
	_, err := m.enrollments.UpdateOne(ctx,
		bson.M{"_id": userID.Hex() + ":" + courseID.Hex()},
		bson.M{"$setOnInsert": bson.M{"user_id": userID, "course_id": courseID, "teacher_id": teacherID, "created_at": time.Now().UTC()}},
		options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) { // зэрэг upsert — аль хэдийн элссэн
		return nil
	}
	return err
}

func (m *Mongo) Enroll(ctx context.Context, userID string, c *Course) error {
	uid, err := oid(userID)
	if err != nil {
		return err
	}
	cid, err := oid(c.ID)
	if err != nil {
		return err
	}
	tid, err := oid(c.TeacherID)
	if err != nil {
		return err
	}
	return m.enroll(ctx, uid, cid, tid)
}

func (m *Mongo) EnrolledCourses(ctx context.Context, userID string) ([]Course, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cur, err := m.enrollments.Find(ctx, bson.M{"user_id": uid},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(500))
	if err != nil {
		return nil, err
	}
	var docs []enrollmentDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []Course{}, nil
	}
	ids := make([]bson.ObjectID, len(docs))
	for i, d := range docs {
		ids[i] = d.CourseID
	}
	courses, err := m.findCourses(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Course, len(courses))
	for _, c := range courses {
		byID[c.ID] = c
	}
	out := make([]Course, 0, len(docs))
	for _, d := range docs {
		if c, ok := byID[d.CourseID.Hex()]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// ---- захиалга ----

func (m *Mongo) CreateOrGetPendingOrder(ctx context.Context, userID string, c *Course) (*Order, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cid, err := oid(c.ID)
	if err != nil {
		return nil, err
	}
	tid, err := oid(c.TeacherID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"user_id": uid, "course_id": cid, "status": string(OrderPending), "kind": OrderKindCourse}
	update := bson.M{
		"$set":         bson.M{"amount": c.Price},
		"$setOnInsert": bson.M{"teacher_id": tid, "title": c.Title, "created_at": time.Now().UTC()},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var d orderDoc
	for attempt := 0; ; attempt++ {
		err = m.orders.FindOneAndUpdate(ctx, filter, update, opts).Decode(&d)
		// Зэрэг хоёр хүсэлт upsert хийхэд нэг нь unique индекст мөргөнө — дахин оролдоход олдоно.
		if mongo.IsDuplicateKeyError(err) && attempt < 2 {
			continue
		}
		break
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return d.model(), nil
}

func (m *Mongo) OrderByID(ctx context.Context, id string) (*Order, error) {
	o, err := oid(id)
	if err != nil {
		return nil, err
	}
	var d orderDoc
	if err := m.orders.FindOne(ctx, bson.M{"_id": o}).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	return d.model(), nil
}

func (m *Mongo) MarkOrderPaid(ctx context.Context, orderID string, amount int64) (*Order, error) {
	o, err := oid(orderID)
	if err != nil {
		return nil, err
	}
	var d orderDoc
	err = m.orders.FindOneAndUpdate(ctx,
		bson.M{"_id": o, "status": string(OrderPending), "amount": amount},
		bson.M{"$set": bson.M{"status": string(OrderPaid), "paid_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Аль хэдийн төлөгдсөн (webhook давтагдсан) эсвэл дүн таарахгүй.
		if err := m.orders.FindOne(ctx, bson.M{"_id": o}).Decode(&d); err != nil {
			return nil, mapErr(err)
		}
		if d.Status != string(OrderPaid) {
			return nil, ErrConflict
		}
	} else if err != nil {
		return nil, mapErr(err)
	}
	// Дараах алхмууд идемпотент; өмнөх оролдлого энд тасарсан ч webhook давтахад нөхөгдөнө.
	if d.Kind == OrderKindLesson {
		_, err := m.lessonAccess.UpdateOne(ctx, bson.M{"_id": d.UserID.Hex() + ":" + d.LessonID.Hex()},
			bson.M{"$setOnInsert": bson.M{"user_id": d.UserID, "lesson_id": d.LessonID, "course_id": d.CourseID,
				"teacher_id": d.TeacherID, "created_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return nil, err
		}
		return d.model(), nil
	}
	if d.Kind == OrderKindBook {
		if err := m.grantBook(ctx, d.UserID.Hex(), d.BookID, d.Amount); err != nil {
			return nil, err
		}
		return d.model(), nil
	}
	if d.Kind == OrderKindStorage {
		if err := m.applyStorage(ctx, &d); err != nil {
			return nil, err
		}
		return d.model(), nil
	}
	if err := m.enroll(ctx, d.UserID, d.CourseID, d.TeacherID); err != nil {
		return nil, err
	}
	return d.model(), nil
}

// applyStorage нь багтаамжийн захиалгыг хэрэглэгч дээр яг нэг удаа хэрэгжүүлнэ:
// storage_orders массивт захиалгын ID байхгүй үед л шинэчлэгдэх нөхцөлтэй атом update.
func (m *Mongo) applyStorage(ctx context.Context, o *orderDoc) error {
	now := time.Now().UTC()
	months := time.Duration(o.Months) * StorageMonth
	_, err := m.users.UpdateOne(ctx,
		bson.M{"_id": o.UserID, "storage_orders": bson.M{"$ne": o.ID}},
		mongo.Pipeline{{{Key: "$set", Value: bson.M{
			"storage_extra_bytes": o.StorageMB << 20,
			"storage_expires_at": bson.M{"$add": bson.A{
				bson.M{"$max": bson.A{"$storage_expires_at", now}}, months.Milliseconds(),
			}},
			"storage_orders": bson.M{"$concatArrays": bson.A{bson.M{"$ifNull": bson.A{"$storage_orders", bson.A{}}}, bson.A{o.ID}}},
		}}}})
	return err
}

func (m *Mongo) CreateStorageOrder(ctx context.Context, userID string, mb int64, months int, amount int64) (*Order, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	d := orderDoc{ID: bson.NewObjectID(), Kind: OrderKindStorage, StorageMB: mb, Months: months, UserID: uid,
		Amount: amount, Status: string(OrderPending), CreatedAt: time.Now().UTC()}
	if _, err := m.orders.InsertOne(ctx, d); err != nil {
		return nil, mapErr(err)
	}
	return d.model(), nil
}

func (m *Mongo) TeacherSales(ctx context.Context, teacherID string, limit int) (*SalesSummary, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	match := bson.M{"teacher_id": tid, "status": string(OrderPaid), "kind": bson.M{"$in": bson.A{OrderKindCourse, OrderKindLesson}}}
	sum := &SalesSummary{Recent: []Sale{}}

	cur, err := m.orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}, "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var agg []struct {
		Total int64 `bson:"total"`
		Count int64 `bson:"count"`
	}
	if err := cur.All(ctx, &agg); err != nil {
		return nil, err
	}
	if len(agg) > 0 {
		sum.TotalAmount, sum.Count = agg[0].Total, agg[0].Count
	}

	cur, err = m.orders.Find(ctx, match, options.Find().SetSort(bson.D{{Key: "paid_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var orders []orderDoc
	if err := cur.All(ctx, &orders); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return sum, nil
	}
	userIDs, courseIDs := make([]bson.ObjectID, 0, len(orders)), make([]bson.ObjectID, 0, len(orders))
	for _, o := range orders {
		userIDs, courseIDs = append(userIDs, o.UserID), append(courseIDs, o.CourseID)
	}
	names := map[bson.ObjectID]string{}
	ucur, err := m.users.Find(ctx, bson.M{"_id": bson.M{"$in": userIDs}}, options.Find().SetProjection(bson.M{"username": 1}))
	if err != nil {
		return nil, err
	}
	var us []userDoc
	if err := ucur.All(ctx, &us); err != nil {
		return nil, err
	}
	for _, u := range us {
		names[u.ID] = u.Username
	}
	titles := map[string]string{}
	cs, err := m.findCourses(ctx, bson.M{"_id": bson.M{"$in": courseIDs}}, options.Find().SetProjection(bson.M{"title": 1, "teacher_id": 1}))
	if err != nil {
		return nil, err
	}
	for _, c := range cs {
		titles[c.ID] = c.Title
	}
	for _, o := range orders {
		title := o.Title
		if title == "" {
			title = titles[o.CourseID.Hex()]
		}
		s := Sale{OrderID: o.ID.Hex(), CourseID: o.CourseID.Hex(), CourseTitle: title,
			BuyerUsername: names[o.UserID], Amount: o.Amount}
		if o.PaidAt != nil {
			s.PaidAt = *o.PaidAt
		}
		sum.Recent = append(sum.Recent, s)
	}
	return sum, nil
}

// ---- чат ----

func (m *Mongo) GetOrCreateConversation(ctx context.Context, teacherID, visitorKey, visitorName, userID string) (*Conversation, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	onInsert := bson.M{"created_at": now, "last_message_at": now, "last_message": ""}
	if userID != "" {
		uid, err := oid(userID)
		if err != nil {
			return nil, err
		}
		onInsert["user_id"] = uid
	}
	var d convDoc
	for attempt := 0; ; attempt++ {
		err = m.conversations.FindOneAndUpdate(ctx,
			bson.M{"teacher_id": tid, "visitor_key": visitorKey},
			bson.M{"$set": bson.M{"visitor_name": visitorName}, "$setOnInsert": onInsert},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
		if mongo.IsDuplicateKeyError(err) && attempt < 2 {
			continue
		}
		break
	}
	if err != nil {
		return nil, mapErr(err)
	}
	c := d.model()
	return &c, nil
}

func (m *Mongo) ConversationByID(ctx context.Context, id string) (*Conversation, error) {
	o, err := oid(id)
	if err != nil {
		return nil, err
	}
	var d convDoc
	if err := m.conversations.FindOne(ctx, bson.M{"_id": o}).Decode(&d); err != nil {
		return nil, mapErr(err)
	}
	c := d.model()
	return &c, nil
}

func (m *Mongo) AddMessage(ctx context.Context, msg *Message) error {
	cid, err := oid(msg.ConversationID)
	if err != nil {
		return err
	}
	var c convDoc
	now := time.Now().UTC()
	// Яриаг шинэчлэхтэй зэрэгцээ багш/зочны түлхүүрийг авна (нэг round-trip).
	err = m.conversations.FindOneAndUpdate(ctx, bson.M{"_id": cid},
		bson.M{"$set": bson.M{"last_message": truncate(msg.Body, 200), "last_message_at": now}}).Decode(&c)
	if err != nil {
		return mapErr(err)
	}
	d := msgDoc{ID: bson.NewObjectID(), ConversationID: cid, TeacherID: c.TeacherID, VisitorKey: c.VisitorKey,
		Sender: msg.Sender, SenderName: msg.SenderName, Body: msg.Body, CreatedAt: now}
	if msg.SenderID != "" {
		if sid, err := oid(msg.SenderID); err == nil {
			d.SenderID = sid
		}
	}
	if _, err := m.messages.InsertOne(ctx, d); err != nil {
		return mapErr(err)
	}
	*msg = d.model()
	return nil
}

func (m *Mongo) Messages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	cid, err := oid(conversationID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"conversation_id": cid}
	if beforeID != "" {
		b, err := oid(beforeID)
		if err != nil {
			return nil, err
		}
		filter["_id"] = bson.M{"$lt": b}
	}
	cur, err := m.messages.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []msgDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Message, len(docs))
	for i := range docs { // шинээс хуучин -> хуучнаас шинэ
		out[len(docs)-1-i] = docs[i].model()
	}
	return out, nil
}

func (m *Mongo) TeacherConversations(ctx context.Context, teacherID string, limit int) ([]Conversation, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	cur, err := m.conversations.Find(ctx, bson.M{"teacher_id": tid, "last_message": bson.M{"$ne": ""}},
		options.Find().SetSort(bson.D{{Key: "last_message_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []convDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Conversation, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

// WatchMessages нь MongoDB change stream-ээр шинэ мессеж бүрийг fn руу дамжуулна.
// Ингэснээр аль ч серверт бичигдсэн мессеж бүх серверийн WebSocket-д хүрнэ.
// Replica set биш бол эхлэхэд алдаа буцаана (дуудагч нь локал горимд шилжинэ).
func (m *Mongo) WatchMessages(ctx context.Context, log *slog.Logger, fn func(Message)) error {
	return watchInserts(ctx, m.messages, log, func(d *msgDoc) { fn(d.model()) })
}

// WatchNotifications нь шинэ мэдэгдлүүдийг бүх сервер рүү түгээнэ.
func (m *Mongo) WatchNotifications(ctx context.Context, log *slog.Logger, fn func(Notification)) error {
	return watchInserts(ctx, m.notifications, log, func(d *notifDoc) { fn(d.model()) })
}

func watchInserts[D any](ctx context.Context, coll *mongo.Collection, log *slog.Logger, fn func(*D)) error {
	pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"operationType": "insert"}}}}
	cs, err := coll.Watch(ctx, pipeline)
	if err != nil {
		return err
	}
	go func() {
		var token bson.Raw
		for {
			for cs.Next(ctx) {
				var ev struct {
					FullDocument D `bson:"fullDocument"`
				}
				if err := cs.Decode(&ev); err == nil {
					fn(&ev.FullDocument)
				}
				token = cs.ResumeToken()
			}
			err := cs.Err()
			_ = cs.Close(context.Background())
			if ctx.Err() != nil {
				return
			}
			log.Warn("change stream тасарлаа, дахин холбогдоно", "err", err)
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
				opts := options.ChangeStream()
				if token != nil {
					opts.SetResumeAfter(token)
				}
				if cs, err = coll.Watch(ctx, pipeline, opts); err == nil {
					break
				}
				log.Warn("change stream дахин нээж чадсангүй", "err", err)
			}
		}
	}()
	return nil
}

// ---- Google Meet ----

func (m *Mongo) SetGoogleToken(ctx context.Context, userID, encrypted string) error {
	uid, err := oid(userID)
	if err != nil {
		return err
	}
	upd := bson.M{"$set": bson.M{"google_token": encrypted}}
	if encrypted == "" {
		upd = bson.M{"$unset": bson.M{"google_token": ""}}
	}
	res, err := m.users.UpdateByID(ctx, uid, upd)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

type meetingDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	TeacherID   bson.ObjectID `bson:"teacher_id"`
	CourseID    bson.ObjectID `bson:"course_id,omitempty"`
	Title       string        `bson:"title"`
	StartsAt    time.Time     `bson:"starts_at"`
	EndsAt      time.Time     `bson:"ends_at"`
	DurationMin int           `bson:"duration_min"`
	MeetURL     string        `bson:"meet_url"`
	EventID     string        `bson:"event_id"`
	CreatedAt   time.Time     `bson:"created_at"`
}

func (m *Mongo) CreateMeeting(ctx context.Context, mt *Meeting) error {
	tid, err := oid(mt.TeacherID)
	if err != nil {
		return err
	}
	d := meetingDoc{ID: bson.NewObjectID(), TeacherID: tid, Title: mt.Title, StartsAt: mt.StartsAt.UTC(),
		EndsAt: mt.StartsAt.Add(time.Duration(mt.DurationMin) * time.Minute).UTC(), DurationMin: mt.DurationMin,
		MeetURL: mt.MeetURL, EventID: mt.EventID, CreatedAt: time.Now().UTC()}
	if mt.CourseID != "" {
		if d.CourseID, err = oid(mt.CourseID); err != nil {
			return err
		}
	}
	if _, err := m.meetings.InsertOne(ctx, d); err != nil {
		return mapErr(err)
	}
	mt.ID, mt.CreatedAt = d.ID.Hex(), d.CreatedAt
	return nil
}

func (m *Mongo) Meetings(ctx context.Context, teacherID, courseID string, from time.Time, limit int) ([]Meeting, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"teacher_id": tid, "ends_at": bson.M{"$gt": from.UTC()}}
	if courseID != "" {
		cid, err := oid(courseID)
		if err != nil {
			return nil, err
		}
		filter["course_id"] = cid
	}
	cur, err := m.meetings.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "starts_at", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []meetingDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Meeting, len(docs))
	for i, d := range docs {
		out[i] = d.model()
	}
	return out, nil
}

// ---- үзэлт ба мэдэгдэл ----

func (m *Mongo) IncViews(ctx context.Context, profile, course map[string]int64) error {
	var um, cm []mongo.WriteModel
	for id, n := range profile {
		if o, err := oid(id); err == nil {
			um = append(um, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": o}).SetUpdate(bson.M{"$inc": bson.M{"profile_views": n}}))
		}
	}
	for id, n := range course {
		if o, err := oid(id); err == nil {
			cm = append(cm, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": o}).SetUpdate(bson.M{"$inc": bson.M{"views": n}}))
		}
	}
	unordered := options.BulkWrite().SetOrdered(false)
	if len(um) > 0 {
		if _, err := m.users.BulkWrite(ctx, um, unordered); err != nil {
			return err
		}
	}
	if len(cm) > 0 {
		if _, err := m.courses.BulkWrite(ctx, cm, unordered); err != nil {
			return err
		}
	}
	return nil
}

type notifDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    bson.ObjectID `bson:"user_id"`
	Type      string        `bson:"type"`
	Title     string        `bson:"title"`
	Body      string        `bson:"body"`
	Link      string        `bson:"link"`
	Count     int64         `bson:"count"`
	Read      bool          `bson:"read"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d *notifDoc) model() Notification {
	return Notification{ID: d.ID.Hex(), UserID: d.UserID.Hex(), Type: d.Type, Title: d.Title, Body: d.Body,
		Link: d.Link, Count: d.Count, Read: d.Read, CreatedAt: d.CreatedAt}
}

func (m *Mongo) AddNotifications(ctx context.Context, ns []*Notification) error {
	docs := make([]any, 0, len(ns))
	now := time.Now().UTC()
	for _, n := range ns {
		uid, err := oid(n.UserID)
		if err != nil {
			continue
		}
		d := notifDoc{ID: bson.NewObjectID(), UserID: uid, Type: n.Type, Title: n.Title, Body: n.Body, Link: n.Link,
			Count: n.Count, CreatedAt: now}
		n.ID, n.CreatedAt = d.ID.Hex(), now
		docs = append(docs, d)
	}
	if len(docs) == 0 {
		return nil
	}
	_, err := m.notifications.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	return err
}

func (m *Mongo) Notifications(ctx context.Context, userID string, limit int) ([]Notification, int64, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, 0, err
	}
	cur, err := m.notifications.Find(ctx, bson.M{"user_id": uid},
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	var docs []notifDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, 0, err
	}
	unread, err := m.notifications.CountDocuments(ctx, bson.M{"user_id": uid, "read": false})
	if err != nil {
		return nil, 0, err
	}
	out := make([]Notification, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, unread, nil
}

func (m *Mongo) MarkNotificationsRead(ctx context.Context, userID string) error {
	uid, err := oid(userID)
	if err != nil {
		return err
	}
	_, err = m.notifications.UpdateMany(ctx, bson.M{"user_id": uid, "read": false}, bson.M{"$set": bson.M{"read": true}})
	return err
}

func (m *Mongo) UpdateLesson(ctx context.Context, l *Lesson) error {
	cid, err := oid(l.CourseID)
	if err != nil {
		return err
	}
	lid, err := oid(l.ID)
	if err != nil {
		return err
	}
	var before lessonDoc
	err = m.lessons.FindOneAndUpdate(ctx, bson.M{"_id": lid, "course_id": cid},
		bson.M{"$set": bson.M{"title": l.Title, "content": l.Content, "video_url": l.VideoURL, "is_free": l.IsFree, "price": l.Price,
			"unlock_after_h": l.UnlockAfterH, "always_open": l.AlwaysOpen, "format": l.Format, "mode": l.Mode, "section": l.Section, "blocks": l.Blocks, "active_min": l.ActiveMin, "exam": l.Exam}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if err != nil {
		return mapErr(err)
	}
	// Үнэгүй хичээлийн тоолуурыг тохируулна.
	if before.IsFree != l.IsFree {
		delta := 1
		if !l.IsFree {
			delta = -1
		}
		if _, err := m.courses.UpdateByID(ctx, cid, bson.M{"$inc": bson.M{"free_lesson_count": delta}, "$set": bson.M{"updated_at": time.Now().UTC()}}); err != nil {
			return err
		}
	}
	l.Position, l.CreatedAt = before.Position, before.CreatedAt
	return nil
}

func (m *Mongo) CreateOrGetPendingLessonOrder(ctx context.Context, userID string, c *Course, l *Lesson) (*Order, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cid, err := oid(c.ID)
	if err != nil {
		return nil, err
	}
	lid, err := oid(l.ID)
	if err != nil {
		return nil, err
	}
	tid, err := oid(c.TeacherID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"user_id": uid, "lesson_id": lid, "status": string(OrderPending), "kind": OrderKindLesson}
	update := bson.M{
		"$set":         bson.M{"amount": l.Price},
		"$setOnInsert": bson.M{"course_id": cid, "teacher_id": tid, "title": c.Title + " — " + l.Title, "created_at": time.Now().UTC()},
	}
	var d orderDoc
	for attempt := 0; ; attempt++ {
		err = m.orders.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
		if mongo.IsDuplicateKeyError(err) && attempt < 2 {
			continue
		}
		break
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return d.model(), nil
}

func (m *Mongo) PurchasedLessons(ctx context.Context, userID, courseID string) ([]string, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	cur, err := m.lessonAccess.Find(ctx, bson.M{"user_id": uid, "course_id": cid}, options.Find().SetProjection(bson.M{"lesson_id": 1}))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		LessonID bson.ObjectID `bson:"lesson_id"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.LessonID.Hex()
	}
	return out, nil
}

// ---- суралцагчийн нүүр хуудас ----

func oids(hexes []string) []bson.ObjectID {
	out := make([]bson.ObjectID, 0, len(hexes))
	for _, h := range hexes {
		if id, err := bson.ObjectIDFromHex(h); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func (m *Mongo) CoursesByIDs(ctx context.Context, ids []string) ([]Course, error) {
	in := oids(ids)
	if len(in) == 0 {
		return []Course{}, nil
	}
	return m.findCourses(ctx, bson.M{"_id": bson.M{"$in": in}})
}

func (m *Mongo) UserOrders(ctx context.Context, userID string, limit int) ([]Order, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cur, err := m.orders.Find(ctx, bson.M{"user_id": uid, "kind": bson.M{"$ne": OrderKindStorage}},
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []orderDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Order, len(docs))
	for i := range docs {
		out[i] = *docs[i].model()
	}
	return out, nil
}

func (m *Mongo) UserConversations(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	// visitor_key-г давхар шалгана: user_id нь зөвхөн тухайн хэрэглэгч зочноор оролцсон яриаг заана.
	cur, err := m.conversations.Find(ctx,
		bson.M{"user_id": uid, "visitor_key": "u:" + userID, "last_message": bson.M{"$ne": ""}},
		options.Find().SetSort(bson.D{{Key: "last_message_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []convDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Conversation, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

func (m *Mongo) MeetingsByCourses(ctx context.Context, courseIDs []string, from time.Time, limit int) ([]Meeting, error) {
	in := oids(courseIDs)
	if len(in) == 0 {
		return []Meeting{}, nil
	}
	cur, err := m.meetings.Find(ctx, bson.M{"course_id": bson.M{"$in": in}, "ends_at": bson.M{"$gt": from.UTC()}},
		options.Find().SetSort(bson.D{{Key: "starts_at", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []meetingDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Meeting, len(docs))
	for i, d := range docs {
		out[i] = d.model()
	}
	return out, nil
}

func (d *meetingDoc) model() Meeting {
	return Meeting{ID: d.ID.Hex(), TeacherID: d.TeacherID.Hex(), CourseID: hexOrEmpty(d.CourseID), Title: d.Title,
		StartsAt: d.StartsAt, DurationMin: d.DurationMin, MeetURL: d.MeetURL, EventID: d.EventID, CreatedAt: d.CreatedAt}
}

func (m *Mongo) UsersByIDs(ctx context.Context, ids []string) ([]User, error) {
	in := oids(ids)
	if len(in) == 0 {
		return []User{}, nil
	}
	cur, err := m.users.Find(ctx, bson.M{"_id": bson.M{"$in": in}})
	if err != nil {
		return nil, err
	}
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]User, len(docs))
	for i := range docs {
		out[i] = *docs[i].model()
	}
	return out, nil
}

// UpdateUsername: давхардлыг username-ийн unique индекс баталгаажуулна (зэрэг хоёр хүсэлтэд ч аюулгүй).
func (m *Mongo) UpdateUsername(ctx context.Context, id, username string) error {
	o, err := oid(id)
	if err != nil {
		return err
	}
	res, err := m.users.UpdateByID(ctx, o, bson.M{"$set": bson.M{"username": username}})
	if err != nil {
		return mapErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) GetOrCreateGroupConversation(ctx context.Context, teacherID, courseID, title string) (*Conversation, error) {
	tid, err := oid(teacherID)
	if err != nil {
		return nil, err
	}
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	// Сургалт энэ багшийнх мөн эсэхийг шалгана.
	if err := m.courses.FindOne(ctx, bson.M{"_id": cid, "teacher_id": tid}, options.FindOne().SetProjection(bson.M{"_id": 1})).Err(); err != nil {
		return nil, mapErr(err)
	}
	now := time.Now().UTC()
	var d convDoc
	for attempt := 0; ; attempt++ {
		err = m.conversations.FindOneAndUpdate(ctx,
			bson.M{"teacher_id": tid, "visitor_key": GroupVisitorKey(courseID)},
			bson.M{"$setOnInsert": bson.M{"visitor_name": title, "kind": ConvGroup, "course_id": cid, "created_at": now, "last_message_at": now, "last_message": ""}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
		if mongo.IsDuplicateKeyError(err) && attempt < 2 {
			continue
		}
		break
	}
	if err != nil {
		return nil, mapErr(err)
	}
	c := d.model()
	return &c, nil
}

func (m *Mongo) UserGroupConversations(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	ids := map[bson.ObjectID]bool{}
	cur, err := m.enrollments.Find(ctx, bson.M{"user_id": uid}, options.Find().SetProjection(bson.M{"course_id": 1}).SetLimit(500))
	if err != nil {
		return nil, err
	}
	var es []struct {
		CourseID bson.ObjectID `bson:"course_id"`
	}
	if err := cur.All(ctx, &es); err != nil {
		return nil, err
	}
	for _, e := range es {
		ids[e.CourseID] = true
	}
	cur, err = m.lessonAccess.Find(ctx, bson.M{"user_id": uid}, options.Find().SetProjection(bson.M{"course_id": 1}).SetLimit(1000))
	if err != nil {
		return nil, err
	}
	if err := cur.All(ctx, &es); err != nil {
		return nil, err
	}
	for _, e := range es {
		ids[e.CourseID] = true
	}
	if len(ids) == 0 {
		return []Conversation{}, nil
	}
	in := make([]bson.ObjectID, 0, len(ids))
	for id := range ids {
		in = append(in, id)
	}
	cur, err = m.conversations.Find(ctx, bson.M{"kind": ConvGroup, "course_id": bson.M{"$in": in}},
		options.Find().SetSort(bson.D{{Key: "last_message_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []convDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]Conversation, len(docs))
	for i := range docs {
		out[i] = docs[i].model()
	}
	return out, nil
}

func (m *Mongo) CourseStudents(ctx context.Context, courseID string) ([]CourseStudent, error) {
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	by := map[string]*CourseStudent{}
	order := []string{}
	get := func(uid string) *CourseStudent {
		if cs, ok := by[uid]; ok {
			return cs
		}
		cs := &CourseStudent{UserID: uid, LessonIDs: []string{}}
		by[uid] = cs
		order = append(order, uid)
		return cs
	}
	cur, err := m.enrollments.Find(ctx, bson.M{"course_id": cid}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(5000))
	if err != nil {
		return nil, err
	}
	var es []enrollmentDoc
	if err := cur.All(ctx, &es); err != nil {
		return nil, err
	}
	for _, e := range es {
		t := e.CreatedAt
		get(e.UserID.Hex()).EnrolledAt = &t
	}
	cur, err = m.lessonAccess.Find(ctx, bson.M{"course_id": cid}, options.Find().SetProjection(bson.M{"user_id": 1, "lesson_id": 1}).SetLimit(20000))
	if err != nil {
		return nil, err
	}
	var ls []struct {
		UserID   bson.ObjectID `bson:"user_id"`
		LessonID bson.ObjectID `bson:"lesson_id"`
	}
	if err := cur.All(ctx, &ls); err != nil {
		return nil, err
	}
	for _, l := range ls {
		cs := get(l.UserID.Hex())
		cs.LessonIDs = append(cs.LessonIDs, l.LessonID.Hex())
	}
	out := make([]CourseStudent, 0, len(order))
	for _, uid := range order {
		out = append(out, *by[uid])
	}
	return out, nil
}

// ---- суралцагчийн явц (дараалсан нээлт) ----

func (m *Mongo) MarkLessonViewed(ctx context.Context, userID, courseID, lessonID string) error {
	uid, err := oid(userID)
	if err != nil {
		return err
	}
	cid, err := oid(courseID)
	if err != nil {
		return err
	}
	lid, err := oid(lessonID)
	if err != nil {
		return err
	}
	_, err = m.progress.UpdateOne(ctx, bson.M{"_id": userID + ":" + lessonID},
		bson.M{"$setOnInsert": bson.M{"user_id": uid, "course_id": cid, "lesson_id": lid, "viewed_at": time.Now().UTC()}},
		options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (m *Mongo) MarkLessonCompleted(ctx context.Context, userID, courseID, lessonID string) error {
	if err := m.MarkLessonViewed(ctx, userID, courseID, lessonID); err != nil {
		return err
	}
	_, err := m.progress.UpdateOne(ctx, bson.M{"_id": userID + ":" + lessonID, "completed_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"completed_at": time.Now().UTC()}})
	return err
}

func (m *Mongo) LessonProgress(ctx context.Context, userID, courseID string) (map[string]LessonProgress, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	cid, err := oid(courseID)
	if err != nil {
		return nil, err
	}
	cur, err := m.progress.Find(ctx, bson.M{"user_id": uid, "course_id": cid})
	if err != nil {
		return nil, err
	}
	var docs []struct {
		LessonID    bson.ObjectID   `bson:"lesson_id"`
		ViewedAt    time.Time       `bson:"viewed_at"`
		CompletedAt *time.Time      `bson:"completed_at,omitempty"`
		Quiz        map[string]bool `bson:"quiz,omitempty"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make(map[string]LessonProgress, len(docs))
	for _, d := range docs {
		out[d.LessonID.Hex()] = LessonProgress{LessonID: d.LessonID.Hex(), ViewedAt: d.ViewedAt, CompletedAt: d.CompletedAt, Quiz: d.Quiz}
	}
	return out, nil
}

func (m *Mongo) ReorderLessons(ctx context.Context, courseID string, items []LessonOrder) error {
	cid, err := oid(courseID)
	if err != nil {
		return err
	}
	if n, err := m.lessons.CountDocuments(ctx, bson.M{"course_id": cid}); err != nil {
		return err
	} else if int(n) != len(items) {
		return ErrNotFound
	}
	models := make([]mongo.WriteModel, 0, len(items))
	seen := make(map[string]bool, len(items))
	for i, it := range items {
		lid, err := oid(it.ID)
		if err != nil || seen[it.ID] {
			return ErrNotFound
		}
		seen[it.ID] = true
		models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": lid, "course_id": cid}).
			SetUpdate(bson.M{"$set": bson.M{"position": i + 1, "section": it.Section}}))
	}
	if len(models) == 0 {
		return nil
	}
	res, err := m.lessons.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return err
	}
	if int(res.MatchedCount) != len(items) {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) SaveQuizResult(ctx context.Context, userID, courseID, lessonID, blockID string, correct bool) error {
	if err := m.MarkLessonViewed(ctx, userID, courseID, lessonID); err != nil {
		return err
	}
	// blockID-г handler шалгасан ([a-z0-9]) тул талбарын нэрэнд аюулгүй.
	_, err := m.progress.UpdateOne(ctx, bson.M{"_id": userID + ":" + lessonID}, bson.M{"$set": bson.M{"quiz." + blockID: correct}})
	return err
}
