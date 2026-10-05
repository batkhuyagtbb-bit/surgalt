package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (m *Mongo) quizLogsC() *mongo.Collection { return m.db.Collection("quiz_logs") }
func (m *Mongo) reflC() *mongo.Collection     { return m.db.Collection("reflections") }
func (m *Mongo) watchesC() *mongo.Collection  { return m.db.Collection("video_watches") }

func engagementIndexes(m *Mongo) map[*mongo.Collection][]mongo.IndexModel {
	idx := func(keys bson.D) mongo.IndexModel { return mongo.IndexModel{Keys: keys} }
	return map[*mongo.Collection][]mongo.IndexModel{
		m.quizLogsC(): {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "at", Value: -1}}), idx(bson.D{{Key: "user_id", Value: 1}, {Key: "lesson_id", Value: 1}})},
		m.reflC():     {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "at", Value: -1}})},
		m.watchesC():  {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "at", Value: -1}}), idx(bson.D{{Key: "user_id", Value: 1}, {Key: "lesson_id", Value: 1}})},
	}
}

func (m *Mongo) AddQuizLog(ctx context.Context, l QuizLog) error {
	l.ID = bson.NewObjectID().Hex()
	if l.At.IsZero() {
		l.At = time.Now().UTC()
	}
	_, err := m.quizLogsC().InsertOne(ctx, l)
	return err
}

func (m *Mongo) QuizLogs(ctx context.Context, f ActivityFilter, limit int) ([]QuizLog, error) {
	o := options.Find().SetSort(bson.D{{Key: "at", Value: -1}})
	if limit > 0 {
		o.SetLimit(int64(limit))
	}
	cur, err := m.quizLogsC().Find(ctx, f.bson("at"), o)
	if err != nil {
		return nil, err
	}
	out := []QuizLog{}
	return out, cur.All(ctx, &out)
}

func (m *Mongo) SaveReflection(ctx context.Context, r *Reflection) error {
	r.ID = r.UserID + ":" + r.LessonID
	if r.At.IsZero() {
		r.At = time.Now().UTC()
	}
	_, err := m.reflC().ReplaceOne(ctx, bson.M{"_id": r.ID}, r, options.Replace().SetUpsert(true))
	return err
}

func (m *Mongo) Reflections(ctx context.Context, f ActivityFilter, limit int) ([]Reflection, error) {
	o := options.Find().SetSort(bson.D{{Key: "at", Value: -1}})
	if limit > 0 {
		o.SetLimit(int64(limit))
	}
	cur, err := m.reflC().Find(ctx, f.bson("at"), o)
	if err != nil {
		return nil, err
	}
	out := []Reflection{}
	return out, cur.All(ctx, &out)
}

func (m *Mongo) AddVideoWatch(ctx context.Context, v VideoWatch) error {
	id := v.UserID + ":" + v.LessonID + ":" + v.BlockID
	inc := bson.M{}
	for k, n := range v.Buckets { // handler k-г int→string хөрвүүлсэн ([0-9]) тул талбарын нэрэнд аюулгүй
		inc["buckets."+itoa(k)] = n
	}
	update := bson.M{"$set": bson.M{"at": time.Now().UTC(), "user_id": v.UserID, "course_id": v.CourseID, "lesson_id": v.LessonID, "teacher_id": v.TeacherID, "block_id": v.BlockID},
		"$max": bson.M{"duration": v.Duration}}
	if len(inc) > 0 {
		update["$inc"] = inc
	}
	_, err := m.watchesC().UpdateOne(ctx, bson.M{"_id": id}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) VideoWatches(ctx context.Context, f ActivityFilter) ([]VideoWatch, error) {
	cur, err := m.watchesC().Find(ctx, f.bson("at"))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		VideoWatch `bson:",inline"`
		Buckets    map[string]int `bson:"buckets"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]VideoWatch, len(docs))
	for i, d := range docs {
		out[i] = d.VideoWatch
		out[i].Buckets = map[int]int{}
		for k, n := range d.Buckets {
			out[i].Buckets[atoi(k)] = n
		}
	}
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func atoi(s string) int {
	n, neg := 0, false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n
}
