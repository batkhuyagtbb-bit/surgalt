package store

import "time"

// ---- Чатын мэдүүлэг ба хориг: зүй бус мессежийг мэдүүлж хянагч (ADMIN_EMAILS) шийдвэрлэнэ; 1:1 ярианд
// аль нэг тал нөгөөгөө хориглож болно (тэгвэл хоёулаа мессеж бичихгүй). ----

const (
	ReportOpen     = "open"
	ReportResolved = "resolved"
)

// ReportMsg — мэдүүлэх үед хадгалсан мессежийн хуулбар: хянагч бүх чатыг биш, зөвхөн эдгээрийг харна.
type ReportMsg struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Body     string    `json:"body"`
	At       time.Time `json:"at"`
	Reported bool      `json:"reported,omitempty"` // мэдүүлсэн мессеж өөрөө
}

// Report — чатын нэг мессежийн тухай мэдүүлэг. Нэг хүн нэг мессежийг нэг л удаа мэдүүлнэ.
type Report struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversation_id"`
	ConvKind       string      `json:"conv_kind"`
	TeacherID      string      `json:"teacher_id"`
	MessageID      string      `json:"message_id"`
	ReporterKey    string      `json:"-"` // "u:<id>" эсвэл зочны түлхүүр
	ReporterID     string      `json:"reporter_id,omitempty"`
	ReporterName   string      `json:"reporter_name"`
	TargetKey      string      `json:"-"` // мэдүүлэгдсэн мессежийн илгээгч
	TargetID       string      `json:"target_id,omitempty"`
	TargetName     string      `json:"target_name"`
	Reason         string      `json:"reason"` // harass | spam | inappropriate | other
	Note           string      `json:"note,omitempty"`
	Context        []ReportMsg `json:"context"`
	Status         string      `json:"status"`
	Resolution     string      `json:"resolution,omitempty"`
	ResolvedBy     string      `json:"resolved_by,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	ResolvedAt     *time.Time  `json:"resolved_at,omitempty"`
}
