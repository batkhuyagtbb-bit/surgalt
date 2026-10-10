package store

import "time"

// ---- Цаг захиалга: багш календарьтаа сул цагаа тэмдэглэж, суралцагч сонгож захиална ----

const (
	OrderKindSlot = "slot" // багштай уулзах төлбөртэй цаг

	SlotOnline  = "online"  // Google Meet холбоостой
	SlotOffline = "offline" // биечлэн уулзана (Location-д)
)

// Slot — багшийн тэмдэглэсэн нэг сул цаг ба (захиалсан бол) түүний захиалга: нэг цаг = нэг хүн.
// Төлбөртэй цагийг захиалж эхлэхэд төлбөр хүлээх хугацаанд (HoldUntil) өөр хүн авч чадахгүй.
type Slot struct {
	ID          string     `json:"id"`
	TeacherID   string     `json:"teacher_id"`
	StartsAt    time.Time  `json:"starts_at"`
	DurationMin int        `json:"duration_min"`
	Mode        string     `json:"mode"`               // online | offline
	Price       int64      `json:"price"`              // 0 = үнэгүй
	Location    string     `json:"location,omitempty"` // биечлэн уулзах газар
	StudentID   string     `json:"student_id,omitempty"`
	StudentName string     `json:"student_name,omitempty"`
	Note        string     `json:"note,omitempty"` // захиалагчийн бичсэн (юу ярилцах)
	HoldBy      string     `json:"-"`              // төлбөр хүлээгдэж буй хүн
	HoldUntil   *time.Time `json:"-"`
	SeriesID    string     `json:"series_id,omitempty"`  // долоо хоног бүр давтагдах цуврал (улирал, жил)
	MeetingID   string     `json:"meeting_id,omitempty"` // баталгаажсаны дараа «Шууд хичээл»-д үүссэн уулзалт
	MeetURL     string     `json:"meet_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Free — захиалаагүй, төлбөр хүлээгдээгүй (эсвэл хүлээх хугацаа нь дууссан).
func (s *Slot) Free(now time.Time) bool {
	return s.StudentID == "" && (s.HoldBy == "" || s.HoldUntil == nil || now.After(*s.HoldUntil))
}

// End — уулзалт дуусах цаг.
func (s *Slot) End() time.Time { return s.StartsAt.Add(time.Duration(s.DurationMin) * time.Minute) }
