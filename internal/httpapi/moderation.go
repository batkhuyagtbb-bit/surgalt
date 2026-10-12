package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// ---- Хууль, итгэл: чатын мэдүүлэг, хориг, хянагч (ADMIN_EMAILS) ----
// Зүй бус мессежийг оролцогч «⚑ Мэдүүлэх»-ээр мэдүүлнэ: мэдүүлсэн мессеж ба түүний өмнөх хэдэн мессежийн хуулбар
// хадгалагдаж, хянагч зөвхөн тэдгээрийг харна. 1:1 ярианд (багш ↔ суралцагч/зочин, ангийн найз) аль нэг тал
// нөгөөгөө хориглож болно — тэгвэл хоёулаа мессеж бичихгүй.

// reportReasons — мэдүүлгийн шалтгаан (түлхүүр → харагдах нэр).
var reportReasons = map[string]string{
	"harass":        "Доромжилсон, дарамталсан",
	"inappropriate": "Зохисгүй агуулга",
	"spam":          "Спам, залилан",
	"other":         "Бусад",
}

const reportContext = 10 // хянагчид харуулах өмнөх мессежийн тоо

// oneToOne — хориг зөвхөн хоёр хүний ярианд үйлчилнэ (бүлэгт багш мессежийг устгаж зохицуулна).
func oneToOne(conv *store.Conversation) bool { return conv.Kind == "" || conv.Kind == store.ConvDM }

// blockView — ярианы хоригийн төлөв: аль тал хориглосон (хоёулаа хориглосон байж болно).
type blockView struct {
	Teacher bool `json:"teacher,omitempty"`
	Visitor bool `json:"visitor,omitempty"`
}

func (s *Server) convBlock(ctx context.Context, conv *store.Conversation) blockView {
	if !oneToOne(conv) {
		return blockView{}
	}
	b, err := s.store.ConversationBlocks(ctx, conv.ID)
	if err != nil {
		s.log.Warn("чатын хориг", "err", err, "conv", conv.ID)
		return blockView{}
	}
	return blockView{Teacher: b[store.SenderTeacher], Visitor: b[store.SenderVisitor]}
}

func (b blockView) any() bool { return b.Teacher || b.Visitor }

var errBlocked = errors.New("энэ ярианд мессеж илгээх боломжгүй")

// messageMine — дуудагч энэ мессежийг өөрөө бичсэн эсэх (handleDeleteMessage-тэй ижил дүрэм).
func messageMine(m *store.Message, uid, key, role string, conv *store.Conversation) bool {
	return (m.SenderID != "" && m.SenderID == uid) || (m.SenderID == "" && m.Sender == role && key == conv.VisitorKey)
}

// handleReport: POST /api/chat/{id}/report {message_id, reason, note, block} — мессежийг мэдүүлнэ (сонговол хориглоно).
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	c, conv, role, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	if !s.chatLim.Allow("report|" + c.VisitorKey()) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	var in struct {
		MessageID string `json:"message_id"`
		Reason    string `json:"reason"`
		Note      string `json:"note"`
		Block     bool   `json:"block"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	switch {
	case reportReasons[in.Reason] == "":
		writeErr(w, http.StatusBadRequest, "шалтгаанаа сонгоно уу")
		return
	case utf8.RuneCountInString(in.Note) > 500:
		writeErr(w, http.StatusBadRequest, "тайлбар 500 тэмдэгтээс ихгүй")
		return
	}
	ctx := r.Context()
	m, err := s.store.MessageByID(ctx, conv.ID, in.MessageID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "мессеж олдсонгүй")
		return
	}
	uid := ""
	if !c.IsGuest() {
		uid = c.UID
	}
	if messageMine(m, uid, c.VisitorKey(), role, conv) {
		writeErr(w, http.StatusBadRequest, "өөрийн мессежийг мэдүүлэх боломжгүй")
		return
	}
	rep := &store.Report{ConversationID: conv.ID, ConvKind: conv.Kind, TeacherID: conv.TeacherID, MessageID: m.ID,
		ReporterKey: c.VisitorKey(), ReporterID: uid, ReporterName: s.displayName(ctx, uid, c.Name),
		TargetName: m.SenderName, Reason: in.Reason, Note: in.Note, Context: s.reportContext(ctx, conv, m)}
	switch {
	case m.SenderID != "":
		rep.TargetKey, rep.TargetID = "u:"+m.SenderID, m.SenderID
	case m.Sender == store.SenderTeacher:
		rep.TargetKey, rep.TargetID = "u:"+conv.TeacherID, conv.TeacherID
	default:
		rep.TargetKey = conv.VisitorKey
	}
	if rep.TargetName == "" {
		rep.TargetName = s.displayName(ctx, rep.TargetID, "Зочин")
	}
	dup := false
	if err := s.store.CreateReport(ctx, rep); errors.Is(err, store.ErrConflict) {
		dup = true // нэг мессежийг давтан мэдүүлбэл дахин бүртгэхгүй
	} else if s.storeErr(w, r, err) {
		return
	}
	if in.Block && oneToOne(conv) {
		if err := s.store.SetConversationBlock(ctx, conv.ID, role, true); s.storeErr(w, r, err) {
			return
		}
		s.chatEvent(conv, map[string]any{"type": "block", "block": s.convBlock(ctx, conv)})
	}
	if !dup {
		s.notifyAdmins(ctx, &store.Notification{Type: NotifReport, Link: "/admin",
			Title: "🚩 Шинэ мэдүүлэг · " + reportReasons[in.Reason],
			Body:  rep.ReporterName + " → " + rep.TargetName + ": «" + short(m.Body) + "»"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "duplicate": dup, "block": s.convBlock(ctx, conv)})
}

// reportContext — мэдүүлсэн мессеж ба түүний өмнөх (мөн дараах 2 хүртэлх) мессежийн хуулбар.
func (s *Server) reportContext(ctx context.Context, conv *store.Conversation, m *store.Message) []store.ReportMsg {
	msgs, err := s.store.Messages(ctx, conv.ID, "", 200)
	at := -1
	for i := range msgs {
		if msgs[i].ID == m.ID {
			at = i
		}
	}
	if err != nil || at < 0 {
		msgs, at = []store.Message{*m}, 0
	}
	lo, hi := max(0, at-reportContext), min(len(msgs), at+3)
	out := make([]store.ReportMsg, 0, hi-lo)
	teacher := ""
	for _, x := range msgs[lo:hi] {
		if x.Deleted {
			continue
		}
		name := x.SenderName
		switch {
		case name != "":
		case x.Sender == store.SenderTeacher:
			if teacher == "" {
				teacher = s.displayName(ctx, conv.TeacherID, "Багш")
			}
			name = teacher
		default:
			name = conv.VisitorName
		}
		out = append(out, store.ReportMsg{ID: x.ID, Name: name, Body: x.Body, At: x.CreatedAt, Reported: x.ID == m.ID})
	}
	return out
}

// handleBlock: POST (хориглох) / DELETE (хоригоо цуцлах) /api/chat/{id}/block — 1:1 ярианд.
func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	_, conv, role, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	if !oneToOne(conv) {
		writeErr(w, http.StatusBadRequest, "бүлэг чатад хориглохгүй — мессежийг мэдүүлнэ үү")
		return
	}
	if err := s.store.SetConversationBlock(r.Context(), conv.ID, role, r.Method == http.MethodPost); s.storeErr(w, r, err) {
		return
	}
	b := s.convBlock(r.Context(), conv)
	s.chatEvent(conv, map[string]any{"type": "block", "block": b})
	writeJSON(w, http.StatusOK, map[string]any{"block": b})
}

// ---- хянагч ----

// isAdmin — ADMIN_EMAILS-д байгаа, Google/Microsoft-оор баталгаажсан имэйлээр нэвтэрсэн (нууц үггүй) бүртгэл.
// Нууц үгтэй бүртгэлийг тооцохгүй: эс бөгөөс хэн нэгэн админы имэйлээр түрүүлж бүртгүүлээд хянагч болно.
// Демо горимд (DEV_PAYMENTS) туршихад нууц үгтэй бүртгэл ч болно.
func (s *Server) isAdmin(u *store.User) bool {
	if u == nil || (u.PasswordHash != "" && !s.cfg.DevPayments) {
		return false
	}
	e := strings.ToLower(strings.TrimSpace(u.Email))
	for _, a := range s.cfg.AdminEmails {
		if e != "" && a == e {
			return true
		}
	}
	return false
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (*store.User, bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return nil, false
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if err != nil || !s.isAdmin(u) {
		writeErr(w, http.StatusForbidden, "зөвхөн хянагч")
		return nil, false
	}
	return u, true
}

// notifyAdmins — бүх хянагчид мэдэгдэнэ.
func (s *Server) notifyAdmins(ctx context.Context, tpl *store.Notification) {
	var ns []*store.Notification
	for _, e := range s.cfg.AdminEmails {
		u, err := s.store.UserByEmail(ctx, e)
		if err != nil || !s.isAdmin(u) {
			continue
		}
		n := *tpl
		n.UserID = u.ID
		ns = append(ns, &n)
	}
	if len(ns) > 0 {
		s.notify(ctx, ns...)
	}
}

// handleAdminReports: GET /api/admin/reports?status=open|resolved|all
func (s *Server) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "", store.ReportOpen:
		status = store.ReportOpen
	case "all":
		status = ""
	case store.ReportResolved:
	default:
		writeErr(w, http.StatusBadRequest, "төлөв буруу")
		return
	}
	rs, err := s.store.Reports(r.Context(), status, 200)
	if s.storeErr(w, r, err) {
		return
	}
	type row struct {
		store.Report
		ReasonLabel string `json:"reason_label"`
	}
	out := make([]row, len(rs))
	for i := range rs {
		out[i] = row{Report: rs[i], ReasonLabel: reportReasons[rs[i].Reason]}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminResolve: POST /api/admin/reports/{id}/resolve {resolution, delete_message} — шийдвэрлэнэ
// (сонговол мэдүүлсэн мессежийг устгана); бүртгэлтэй мэдүүлэгчид мэдэгдэнэ.
func (s *Server) handleAdminResolve(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		Resolution    string `json:"resolution"`
		DeleteMessage bool   `json:"delete_message"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Resolution = strings.TrimSpace(in.Resolution)
	if utf8.RuneCountInString(in.Resolution) > 500 {
		writeErr(w, http.StatusBadRequest, "тайлбар 500 тэмдэгтээс ихгүй")
		return
	}
	ctx, now := r.Context(), time.Now()
	rep, err := s.store.UpdateReport(ctx, r.PathValue("id"), func(x *store.Report) error {
		x.Status, x.Resolution, x.ResolvedBy, x.ResolvedAt = store.ReportResolved, in.Resolution, admin.DisplayName, &now
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	if in.DeleteMessage {
		if err := s.store.DeleteMessage(ctx, rep.ConversationID, rep.MessageID); err == nil {
			if conv, err := s.store.ConversationByID(ctx, rep.ConversationID); err == nil {
				s.chatEvent(conv, map[string]any{"type": "delete", "message_id": rep.MessageID})
			}
		} else if !errors.Is(err, store.ErrNotFound) {
			s.log.Warn("мэдүүлсэн мессеж устгах", "err", err)
		}
	}
	if rep.ReporterID != "" {
		s.notify(ctx, &store.Notification{UserID: rep.ReporterID, Type: NotifReport, Link: "/",
			Title: "✓ Таны мэдүүлгийг хянаж шийдвэрлэлээ", Body: "Мэдүүлсэнд баярлалаа. " + in.Resolution})
	}
	writeJSON(w, http.StatusOK, rep)
}
