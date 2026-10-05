package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Шалгалт, сесс, лог: ID-нууд hex мөрөөр (бусад коллекцтой холбоход хялбар).
func (m *Mongo) attemptsC() *mongo.Collection { return m.db.Collection("exam_attempts") }
func (m *Mongo) sessionsC() *mongo.Collection { return m.db.Collection("study_sessions") }
func (m *Mongo) eventsC() *mongo.Collection   { return m.db.Collection("activity_events") }

func learningIndexes(m *Mongo) map[*mongo.Collection][]mongo.IndexModel {
	idx := func(keys bson.D) mongo.IndexModel { return mongo.IndexModel{Keys: keys} }
	return map[*mongo.Collection][]mongo.IndexModel{
		m.attemptsC(): {idx(bson.D{{Key: "user_id", Value: 1}, {Key: "lesson_id", Value: 1}}), idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "started_at", Value: -1}})},
		m.sessionsC(): {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "last_at", Value: -1}}), idx(bson.D{{Key: "user_id", Value: 1}, {Key: "lesson_id", Value: 1}}), idx(bson.D{{Key: "course_id", Value: 1}, {Key: "last_at", Value: -1}})},
		m.eventsC():   {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "at", Value: -1}}), idx(bson.D{{Key: "course_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "at", Value: -1}})},
	}
}

func (f ActivityFilter) bson(timeField string) bson.M {
	q := bson.M{}
	for k, v := range map[string]string{"teacher_id": f.TeacherID, "course_id": f.CourseID, "user_id": f.UserID, "lesson_id": f.LessonID, "kind": f.Kind} {
		if v != "" {
			q[k] = v
		}
	}
	if !f.Since.IsZero() {
		q[timeField] = bson.M{"$gte": f.Since}
	}
	return q
}

func (m *Mongo) CreateExamAttempt(ctx context.Context, a *ExamAttempt) error {
	a.ID = bson.NewObjectID().Hex()
	_, err := m.attemptsC().InsertOne(ctx, a)
	return err
}

func (m *Mongo) ExamAttemptByID(ctx context.Context, id string) (*ExamAttempt, error) {
	var a ExamAttempt
	if err := m.attemptsC().FindOne(ctx, bson.M{"_id": id}).Decode(&a); err != nil {
		return nil, mapErr(err)
	}
	return &a, nil
}

func (m *Mongo) ExamAttempts(ctx context.Context, f ActivityFilter) ([]ExamAttempt, error) {
	cur, err := m.attemptsC().Find(ctx, f.bson("started_at"), options.Find().SetSort(bson.D{{Key: "started_at", Value: -1}}).SetLimit(5000))
	if err != nil {
		return nil, err
	}
	out := []ExamAttempt{}
	return out, cur.All(ctx, &out)
}

func (m *Mongo) FinishExamAttempt(ctx context.Context, a *ExamAttempt) error {
	res, err := m.attemptsC().UpdateOne(ctx, bson.M{"_id": a.ID, "status": AttemptActive}, bson.M{"$set": bson.M{
		"status": a.Status, "reason": a.Reason, "finished_at": a.FinishedAt, "score": a.Score, "max": a.Max, "pct": a.Pct, "passed": a.Passed, "results": a.Results}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrConflict
	}
	return nil
}

func (m *Mongo) AddAttemptViolation(ctx context.Context, id string) error {
	_, err := m.attemptsC().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"violations": 1}})
	return err
}

func (m *Mongo) CreateSession(ctx context.Context, s *StudySession) error {
	s.ID = bson.NewObjectID().Hex()
	if s.Counts == nil {
		s.Counts = map[string]int{}
	}
	_, err := m.sessionsC().InsertOne(ctx, s)
	return err
}

func (m *Mongo) SessionByID(ctx context.Context, id string) (*StudySession, error) {
	var s StudySession
	if err := m.sessionsC().FindOne(ctx, bson.M{"_id": id}).Decode(&s); err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

func (m *Mongo) AddSessionBeat(ctx context.Context, id string, b SessionBeat) error {
	inc := bson.M{"active_sec": b.ActiveSec, "idle_sec": b.IdleSec, "away_sec": b.AwaySec, "focus_sum": b.FocusSum, "focus_n": b.FocusN}
	for k, v := range b.Counts { // түлхүүрийг handler шалгасан ([a-z_])
		inc["counts."+k] = v
	}
	set := bson.M{"last_at": time.Now().UTC()}
	if b.Camera {
		set["camera"] = true
	}
	if b.End != "" {
		set["ended"], set["end_reason"] = true, b.End
	}
	res, err := m.sessionsC().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": inc, "$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) Sessions(ctx context.Context, f ActivityFilter, limit int) ([]StudySession, error) {
	o := options.Find().SetSort(bson.D{{Key: "last_at", Value: -1}})
	if limit > 0 {
		o.SetLimit(int64(limit))
	}
	cur, err := m.sessionsC().Find(ctx, f.bson("last_at"), o)
	if err != nil {
		return nil, err
	}
	out := []StudySession{}
	return out, cur.All(ctx, &out)
}

func (m *Mongo) AddActivityEvents(ctx context.Context, evs []ActivityEvent) error {
	if len(evs) == 0 {
		return nil
	}
	docs := make([]any, len(evs))
	for i := range evs {
		evs[i].ID = bson.NewObjectID().Hex()
		if evs[i].At.IsZero() {
			evs[i].At = time.Now().UTC()
		}
		docs[i] = evs[i]
	}
	_, err := m.eventsC().InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	return err
}

func (m *Mongo) ActivityEvents(ctx context.Context, f ActivityFilter, limit int) ([]ActivityEvent, error) {
	o := options.Find().SetSort(bson.D{{Key: "at", Value: -1}})
	if limit > 0 {
		o.SetLimit(int64(limit))
	}
	cur, err := m.eventsC().Find(ctx, f.bson("at"), o)
	if err != nil {
		return nil, err
	}
	out := []ActivityEvent{}
	return out, cur.All(ctx, &out)
}
