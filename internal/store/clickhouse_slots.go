package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- цаг захиалга (slots.go) ----

const slotCols = `id, teacher_id, starts_at, duration_min, mode, price, location, student_id, student_name, note,
	hold_by, hold_until, meeting_id, meet_url, created_at`

func scanSlot(r driver.Rows) (Slot, error) {
	var s Slot
	var dur int32
	err := r.Scan(&s.ID, &s.TeacherID, &s.StartsAt, &dur, &s.Mode, &s.Price, &s.Location, &s.StudentID, &s.StudentName, &s.Note,
		&s.HoldBy, &s.HoldUntil, &s.MeetingID, &s.MeetURL, &s.CreatedAt)
	s.DurationMin = int(dur)
	return s, err
}

func (c *ClickHouse) writeSlot(ctx context.Context, s *Slot, deleted bool) error {
	return c.insert(ctx, "slots", []string{"id", "teacher_id", "starts_at", "duration_min", "mode", "price", "location", "student_id",
		"student_name", "note", "hold_by", "hold_until", "meeting_id", "meet_url", "created_at", "ver", "deleted"},
		s.ID, s.TeacherID, s.StartsAt.UTC(), int32(s.DurationMin), s.Mode, s.Price, s.Location, s.StudentID,
		s.StudentName, s.Note, s.HoldBy, nullTime(s.HoldUntil), s.MeetingID, s.MeetURL, s.CreatedAt.UTC(), ver(), deleted)
}

func (c *ClickHouse) slotsWhere(ctx context.Context, where string, args ...any) ([]Slot, error) {
	out := []Slot{}
	err := c.query(ctx, "SELECT "+slotCols+" FROM slots FINAL WHERE deleted = false AND "+where+" ORDER BY starts_at", args, func(r driver.Rows) error {
		s, err := scanSlot(r)
		if err == nil {
			out = append(out, s)
		}
		return err
	})
	return out, err
}

func (c *ClickHouse) AddSlots(ctx context.Context, slots []*Slot) error {
	for _, s := range slots {
		s.ID, s.CreatedAt = NewID(), time.Now()
		if err := c.writeSlot(ctx, s, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClickHouse) SlotByID(ctx context.Context, id string) (*Slot, error) {
	ss, err := c.slotsWhere(ctx, "id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(ss) == 0 {
		return nil, ErrNotFound
	}
	return &ss[0], nil
}

func (c *ClickHouse) Slots(ctx context.Context, teacherID string, from, to time.Time) ([]Slot, error) {
	return c.slotsWhere(ctx, "teacher_id = ? AND starts_at >= ? AND starts_at < ?", teacherID, from.UTC(), to.UTC())
}

func (c *ClickHouse) StudentSlots(ctx context.Context, userID string, from time.Time) ([]Slot, error) {
	return c.slotsWhere(ctx, "student_id = ? AND starts_at >= ?", userID, from.UTC())
}

func (c *ClickHouse) UpdateSlot(ctx context.Context, id string, fn func(*Slot) error) (*Slot, error) {
	unlock, err := c.lock(ctx, "slot:"+id)
	if err != nil {
		return nil, err
	}
	defer unlock()
	s, err := c.SlotByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := fn(s); err != nil {
		return nil, err
	}
	return s, c.writeSlot(ctx, s, false)
}

func (c *ClickHouse) DeleteSlot(ctx context.Context, id string) error {
	unlock, err := c.lock(ctx, "slot:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	s, err := c.SlotByID(ctx, id)
	if err != nil {
		return err
	}
	return c.writeSlot(ctx, s, true)
}

func (c *ClickHouse) CreateOrGetPendingSlotOrder(ctx context.Context, userID string, s *Slot, title string) (*Order, error) {
	return c.pendingOrder(ctx, userID+":s:"+s.ID, "user_id = ? AND slot_id = ? AND kind = ?", []any{userID, s.ID, OrderKindSlot},
		&Order{Kind: OrderKindSlot, Title: title, UserID: userID, SlotID: s.ID, TeacherID: s.TeacherID, Amount: s.Price})
}

// DeleteMeeting — цуцлагдсан цаг захиалгын уулзалтыг «Шууд хичээл»-ээс хасна (хөнгөн DELETE).
func (c *ClickHouse) DeleteMeeting(ctx context.Context, id string) error {
	return c.conn.Exec(ctx, "DELETE FROM meetings WHERE id = ?", id)
}
