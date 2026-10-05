package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (m *Mongo) booksC() *mongo.Collection      { return m.db.Collection("books") }
func (m *Mongo) bookAccessC() *mongo.Collection { return m.db.Collection("book_access") }
func (m *Mongo) bookEventsC() *mongo.Collection { return m.db.Collection("book_events") }

func bookIndexes(m *Mongo) map[*mongo.Collection][]mongo.IndexModel {
	idx := func(keys bson.D) mongo.IndexModel { return mongo.IndexModel{Keys: keys} }
	return map[*mongo.Collection][]mongo.IndexModel{
		m.booksC():      {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "created_at", Value: -1}})},
		m.bookEventsC(): {idx(bson.D{{Key: "teacher_id", Value: 1}, {Key: "at", Value: -1}}), idx(bson.D{{Key: "book_id", Value: 1}, {Key: "at", Value: -1}})},
	}
}

func (m *Mongo) CreateBook(ctx context.Context, b *Book) error {
	b.ID = bson.NewObjectID().Hex()
	b.CreatedAt, b.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	_, err := m.booksC().InsertOne(ctx, b)
	return err
}

func (m *Mongo) UpdateBook(ctx context.Context, b *Book) error {
	err := m.booksC().FindOneAndUpdate(ctx, bson.M{"_id": b.ID, "teacher_id": b.TeacherID}, bson.M{"$set": bson.M{
		"kind": b.Kind, "title": b.Title, "author": b.Author, "description": b.Description, "cover_url": b.CoverURL,
		"price": b.Price, "published": b.Published, "preview_n": b.PreviewN, "updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(b)
	return mapErr(err)
}

func (m *Mongo) SetBookPages(ctx context.Context, id, teacherID string, pages int) error {
	res, err := m.booksC().UpdateOne(ctx, bson.M{"_id": id, "teacher_id": teacherID}, bson.M{"$set": bson.M{"pages": pages, "updated_at": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) DeleteBook(ctx context.Context, id, teacherID string) error {
	res, err := m.booksC().DeleteOne(ctx, bson.M{"_id": id, "teacher_id": teacherID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *Mongo) BookByID(ctx context.Context, id string) (*Book, error) {
	var b Book
	if err := m.booksC().FindOne(ctx, bson.M{"_id": id}).Decode(&b); err != nil {
		return nil, mapErr(err)
	}
	return &b, nil
}

func (m *Mongo) BooksByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Book, error) {
	f := bson.M{"teacher_id": teacherID}
	if onlyPublished {
		f["published"] = true
	}
	cur, err := m.booksC().Find(ctx, f, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(500))
	if err != nil {
		return nil, err
	}
	out := []Book{}
	return out, cur.All(ctx, &out)
}

func (m *Mongo) HasBookAccess(ctx context.Context, userID, bookID string) (bool, error) {
	n, err := m.bookAccessC().CountDocuments(ctx, bson.M{"_id": userID + ":" + bookID}, options.Count().SetLimit(1))
	return n > 0, err
}

// grantBook — захиалга төлөгдөхөд (идемпотент: давтагдсан webhook-д дахин тоолохгүй).
func (m *Mongo) grantBook(ctx context.Context, userID, bookID string, amount int64) error {
	res, err := m.bookAccessC().UpdateOne(ctx, bson.M{"_id": userID + ":" + bookID},
		bson.M{"$setOnInsert": bson.M{"user_id": userID, "book_id": bookID, "created_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return err
	}
	if res != nil && res.UpsertedCount == 1 {
		_, err = m.booksC().UpdateOne(ctx, bson.M{"_id": bookID}, bson.M{"$inc": bson.M{"revenue": amount}})
	}
	return err
}

func (m *Mongo) CreateOrGetPendingBookOrder(ctx context.Context, userID string, b *Book) (*Order, error) {
	uid, err := oid(userID)
	if err != nil {
		return nil, err
	}
	tid, err := oid(b.TeacherID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"user_id": uid, "book_id": b.ID, "status": string(OrderPending), "kind": OrderKindBook}
	update := bson.M{"$set": bson.M{"amount": b.Price}, "$setOnInsert": bson.M{"teacher_id": tid, "title": b.Title, "created_at": time.Now().UTC()}}
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

func (m *Mongo) AddBookEvent(ctx context.Context, e BookEvent) error {
	e.ID = bson.NewObjectID().Hex()
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if _, err := m.bookEventsC().InsertOne(ctx, e); err != nil {
		return err
	}
	if f := bookCounter[e.Type]; f != "" {
		_, err := m.booksC().UpdateOne(ctx, bson.M{"_id": e.BookID}, bson.M{"$inc": bson.M{f: 1}})
		return err
	}
	return nil
}

func (m *Mongo) BookEvents(ctx context.Context, teacherID, bookID string, limit int) ([]BookEvent, error) {
	f := bson.M{"teacher_id": teacherID}
	if bookID != "" {
		f["book_id"] = bookID
	}
	o := options.Find().SetSort(bson.D{{Key: "at", Value: -1}})
	if limit > 0 {
		o.SetLimit(int64(limit))
	}
	cur, err := m.bookEventsC().Find(ctx, f, o)
	if err != nil {
		return nil, err
	}
	out := []BookEvent{}
	return out, cur.All(ctx, &out)
}
