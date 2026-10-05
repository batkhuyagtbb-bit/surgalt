package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const convCols = `id, teacher_id, visitor_key, visitor_name, user_id, kind, course_id, last_message, created_at, last_message_at`

func scanConv(r driver.Rows) (Conversation, error) {
	var cv Conversation
	err := r.Scan(&cv.ID, &cv.TeacherID, &cv.VisitorKey, &cv.VisitorName, &cv.UserID, &cv.Kind, &cv.CourseID,
		&cv.LastMessage, &cv.CreatedAt, &cv.LastMessageAt)
	return cv, err
}

func (c *ClickHouse) writeConv(ctx context.Context, cv *Conversation) error {
	return c.insert(ctx, "conversations", []string{"id", "teacher_id", "visitor_key", "visitor_name", "user_id", "kind", "course_id",
		"last_message", "created_at", "last_message_at", "ver"},
		cv.ID, cv.TeacherID, cv.VisitorKey, cv.VisitorName, cv.UserID, cv.Kind, cv.CourseID, cv.LastMessage,
		cv.CreatedAt.UTC(), cv.LastMessageAt.UTC(), ver())
}

func (c *ClickHouse) convsWhere(ctx context.Context, where, order string, args ...any) ([]Conversation, error) {
	out := []Conversation{}
	err := c.query(ctx, "SELECT "+convCols+" FROM conversations FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		cv, err := scanConv(r)
		if err != nil {
			return err
		}
		out = append(out, cv)
		return nil
	})
	return out, err
}

func (c *ClickHouse) GetOrCreateConversation(ctx context.Context, teacherID, visitorKey, visitorName, userID string) (*Conversation, error) {
	unlock := c.lock("conv:" + teacherID + "|" + visitorKey)
	defer unlock()
	if _, err := c.UserByID(ctx, teacherID); err != nil {
		return nil, err
	}
	cs, err := c.convsWhere(ctx, "teacher_id = ? AND visitor_key = ? AND kind = ''", "LIMIT 1", teacherID, visitorKey)
	if err != nil {
		return nil, err
	}
	if len(cs) > 0 {
		return &cs[0], nil
	}
	now := time.Now()
	cv := &Conversation{ID: NewID(), TeacherID: teacherID, VisitorKey: visitorKey, VisitorName: visitorName, UserID: userID, CreatedAt: now, LastMessageAt: now}
	return cv, c.writeConv(ctx, cv)
}

func (c *ClickHouse) GetOrCreateGroupConversation(ctx context.Context, teacherID, courseID, title string) (*Conversation, error) {
	unlock := c.lock("conv:group:" + courseID)
	defer unlock()
	cs, err := c.convsWhere(ctx, "kind = ? AND course_id = ?", "LIMIT 1", ConvGroup, courseID)
	if err != nil {
		return nil, err
	}
	if len(cs) > 0 {
		return &cs[0], nil
	}
	now := time.Now()
	cv := &Conversation{ID: NewID(), TeacherID: teacherID, VisitorKey: GroupVisitorKey(courseID), VisitorName: title, Kind: ConvGroup,
		CourseID: courseID, CreatedAt: now, LastMessageAt: now}
	return cv, c.writeConv(ctx, cv)
}

// memberCourses — хэрэглэгчийн элссэн эсвэл хичээл худалдаж авсан сургалтууд.
func (c *ClickHouse) memberCourses(ctx context.Context, userID string) ([]string, error) {
	out := []string{}
	err := c.query(ctx, `SELECT DISTINCT course_id FROM (
		SELECT course_id FROM enrollments FINAL WHERE user_id = ?
		UNION ALL SELECT course_id FROM lesson_access FINAL WHERE user_id = ?)`, []any{userID, userID}, func(r driver.Rows) error {
		var id string
		if err := r.Scan(&id); err != nil {
			return err
		}
		out = append(out, id)
		return nil
	})
	return out, err
}

func (c *ClickHouse) UserGroupConversations(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	ids, err := c.memberCourses(ctx, userID)
	if err != nil || len(ids) == 0 {
		return []Conversation{}, err
	}
	return c.convsWhere(ctx, "kind = ? AND has(?, course_id)", "ORDER BY last_message_at DESC LIMIT ?", ConvGroup, ids, limit)
}

func (c *ClickHouse) ConversationByID(ctx context.Context, id string) (*Conversation, error) {
	cs, err := c.convsWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, ErrNotFound
	}
	return &cs[0], nil
}

func (c *ClickHouse) TeacherConversations(ctx context.Context, teacherID string, limit int) ([]Conversation, error) {
	return c.convsWhere(ctx, "teacher_id = ?", "ORDER BY last_message_at DESC LIMIT ?", teacherID, limit)
}

func (c *ClickHouse) UserConversations(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	return c.convsWhere(ctx, "user_id = ? AND kind = ''", "ORDER BY last_message_at DESC LIMIT ?", userID, limit)
}

func (c *ClickHouse) AddMessage(ctx context.Context, m *Message) error {
	unlock := c.lock("conv:" + m.ConversationID)
	defer unlock()
	cv, err := c.ConversationByID(ctx, m.ConversationID)
	if err != nil {
		return err
	}
	m.ID, m.CreatedAt, m.TeacherID, m.VisitorKey = NewID(), time.Now(), cv.TeacherID, cv.VisitorKey
	if err := c.insert(ctx, "messages", []string{"id", "conversation_id", "teacher_id", "visitor_key", "sender", "sender_id", "sender_name", "body", "created_at"},
		m.ID, m.ConversationID, m.TeacherID, m.VisitorKey, m.Sender, m.SenderID, m.SenderName, m.Body, m.CreatedAt.UTC()); err != nil {
		return err
	}
	cv.LastMessage, cv.LastMessageAt = m.Body, m.CreatedAt
	return c.writeConv(ctx, cv)
}

func (c *ClickHouse) Messages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	out := []Message{}
	err := c.query(ctx, `SELECT id, conversation_id, teacher_id, visitor_key, sender, sender_id, sender_name, body, created_at
		FROM messages WHERE conversation_id = ? AND (? = '' OR id < ?) ORDER BY id DESC LIMIT ?`,
		[]any{conversationID, beforeID, beforeID, limit}, func(r driver.Rows) error {
			var m Message
			if err := r.Scan(&m.ID, &m.ConversationID, &m.TeacherID, &m.VisitorKey, &m.Sender, &m.SenderID, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
				return err
			}
			out = append(out, m)
			return nil
		})
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 { // хуучнаас шинэ рүү
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
