package httpapi

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

// requirePrincipal: нэвтэрсэн хэрэглэгч ЭСВЭЛ зочин.
func (s *Server) requirePrincipal(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := s.principal(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "нэвтрэх эсвэл нэрээ оруулна уу")
	}
	return c, ok
}

// convRole нь дуудагч энэ ярианд хэн болохыг (багш/зочин) тодорхойлно.
// Бүлэг чатад: багш эзэмшигч, эсвэл сургалтад элссэн / хичээл худалдаж авсан бүртгэлтэй хэрэглэгч.
func (s *Server) convRole(ctx context.Context, c auth.Claims, conv *store.Conversation) (string, bool) {
	if !c.IsGuest() && c.UID == conv.TeacherID {
		return store.SenderTeacher, true
	}
	if conv.Kind == store.ConvGroup {
		if c.IsGuest() {
			return "", false
		}
		ok, err := s.courseMember(ctx, c.UID, conv.CourseID)
		if err != nil || !ok {
			return "", false
		}
		return store.SenderVisitor, true
	}
	if c.VisitorKey() == conv.VisitorKey {
		return store.SenderVisitor, true
	}
	return "", false
}

// courseMember: элссэн эсвэл дор хаяж нэг хичээл худалдаж авсан.
func (s *Server) courseMember(ctx context.Context, userID, courseID string) (bool, error) {
	if ok, err := s.store.IsEnrolled(ctx, userID, courseID); err != nil || ok {
		return ok, err
	}
	bought, err := s.store.PurchasedLessons(ctx, userID, courseID)
	return len(bought) > 0, err
}

func (s *Server) loadConv(w http.ResponseWriter, r *http.Request) (auth.Claims, *store.Conversation, string, bool) {
	c, ok := s.requirePrincipal(w, r)
	if !ok {
		return c, nil, "", false
	}
	conv, err := s.store.ConversationByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return c, nil, "", false
	}
	role, ok := s.convRole(r.Context(), c, conv)
	if !ok {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return c, nil, "", false
	}
	return c, conv, role, true
}

// handleStartChat нь зочны тухайн багштай ярианы өрөөг нээж, сүүлийн мессежүүдийг буцаана.
func (s *Server) handleStartChat(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requirePrincipal(w, r)
	if !ok {
		return
	}
	p, err := s.publicProfile(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	if !c.IsGuest() && c.UID == p.Data.Teacher.ID {
		writeErr(w, http.StatusBadRequest, "өөртэйгөө чатлах боломжгүй — профайлынхаа Чат хэсгээс хариулна уу")
		return
	}
	conv, err := s.store.GetOrCreateConversation(r.Context(), p.Data.Teacher.ID, c.VisitorKey(), c.Name, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	msgs, err := s.store.Messages(r.Context(), conv.ID, "", 50)
	if s.storeErr(w, r, err) {
		return
	}
	if err := s.attachReactions(r, msgs); s.storeErr(w, r, err) {
		return
	}
	s.signAttachments(msgs)
	reads, _ := s.store.ConversationReads(r.Context(), conv.ID)
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": store.SenderVisitor, "me_key": c.VisitorKey(), "reads": reads})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	c, conv, role, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	msgs, err := s.store.Messages(r.Context(), conv.ID, r.URL.Query().Get("before"), 50)
	if s.storeErr(w, r, err) {
		return
	}
	if err := s.attachReactions(r, msgs); s.storeErr(w, r, err) {
		return
	}
	s.signAttachments(msgs)
	reads, err := s.store.ConversationReads(r.Context(), conv.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": role, "me_key": c.VisitorKey(), "reads": reads})
}

// attachReactions — мессежүүдэд реакцуудыг нь бөглөнө.
func (s *Server) attachReactions(r *http.Request, msgs []store.Message) error {
	ids := make([]string, len(msgs))
	for i := range msgs {
		ids[i] = msgs[i].ID
	}
	rx, err := s.store.MessageReactions(r.Context(), ids)
	if err != nil {
		return err
	}
	for i := range msgs {
		msgs[i].Reactions = rx[msgs[i].ID]
	}
	return nil
}

// chatEvent — ярианы оролцогчдод ердийн мессежээс өөр үйл явдал (реакц, уншсан, бичиж байна).
func (s *Server) chatEvent(conv *store.Conversation, ev map[string]any) {
	ev["conversation_id"] = conv.ID
	s.hub.Fanout(conv.TeacherID, conv.VisitorKey, conv.ID, ev)
}

// allowedReactions — Messenger маягийн 6 реакц.
var allowedReactions = map[string]bool{"👍": true, "❤️": true, "😂": true, "😮": true, "😢": true, "🙏": true}

// handleReact: POST /api/chat/{id}/react {message_id, emoji} — ижил эможи дахин дарвал хасна.
func (s *Server) handleReact(w http.ResponseWriter, r *http.Request) {
	c, conv, _, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	var in struct {
		MessageID string `json:"message_id"`
		Emoji     string `json:"emoji"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Emoji != "" && !allowedReactions[in.Emoji] {
		writeErr(w, http.StatusBadRequest, "реакц: 👍 ❤️ 😂 😮 😢 🙏")
		return
	}
	if _, err := s.store.MessageByID(r.Context(), conv.ID, in.MessageID); err != nil {
		writeErr(w, http.StatusNotFound, "мессеж олдсонгүй")
		return
	}
	key := c.VisitorKey()
	cur, _ := s.store.MessageReactions(r.Context(), []string{in.MessageID})
	for e, users := range cur[in.MessageID] {
		for _, u := range users {
			if u.Key == key && e == in.Emoji {
				in.Emoji = "" // ижил реакц → хасна
			}
		}
	}
	name := c.Name
	if !c.IsGuest() {
		name = s.displayName(r.Context(), c.UID, c.Name)
	}
	if err := s.store.ReactMessage(r.Context(), conv.ID, in.MessageID, key, name, in.Emoji); s.storeErr(w, r, err) {
		return
	}
	rx, _ := s.store.MessageReactions(r.Context(), []string{in.MessageID})
	out := rx[in.MessageID]
	if out == nil {
		out = map[string][]store.ReactUser{}
	}
	s.chatEvent(conv, map[string]any{"type": "reaction", "message_id": in.MessageID, "reactions": out})
	writeJSON(w, http.StatusOK, map[string]any{"message_id": in.MessageID, "reactions": out, "mine": in.Emoji})
}

// handleRead: POST /api/chat/{id}/read {message_id} — энэ хүртэл үзсэн ("Үзсэн" тэмдэг).
func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	c, conv, _, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	var in struct {
		MessageID string `json:"message_id"`
	}
	if !decode(w, r, &in) || in.MessageID == "" {
		writeErr(w, http.StatusBadRequest, "message_id хэрэгтэй")
		return
	}
	if err := s.store.MarkRead(r.Context(), conv.ID, c.VisitorKey(), in.MessageID); s.storeErr(w, r, err) {
		return
	}
	s.chatEvent(conv, map[string]any{"type": "read", "key": c.VisitorKey(), "name": c.Name, "last_id": in.MessageID})
	w.WriteHeader(http.StatusNoContent)
}

// handleTyping: POST /api/chat/{id}/typing — "бичиж байна…" (хадгалагдахгүй, тухайн агшинд түгээнэ).
func (s *Server) handleTyping(w http.ResponseWriter, r *http.Request) {
	c, conv, _, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	name := c.Name
	if !c.IsGuest() {
		name = s.displayName(r.Context(), c.UID, c.Name)
	}
	s.chatEvent(conv, map[string]any{"type": "typing", "key": c.VisitorKey(), "name": name})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	c, conv, role, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	if !s.chatLim.Allow(c.VisitorKey()) {
		writeErr(w, http.StatusTooManyRequests, "хэт хурдан бичиж байна")
		return
	}
	var in struct {
		Body       string `json:"body"`
		ReplyTo    string `json:"reply_to"`
		Attachment string `json:"attachment"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Attachment != "" && !(files.OwnedBy(in.Attachment, conv.TeacherID) && strings.Contains(in.Attachment, "/"+files.Private+"/") && strings.Contains(in.Attachment, "__chat_")) {
		writeErr(w, http.StatusBadRequest, "хавсралт буруу")
		return
	}
	if in.Attachment != "" && in.Body == "" {
		in.Body = "📷 Зураг"
	}
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > 2000 {
		writeErr(w, http.StatusBadRequest, "мессеж 1-2000 тэмдэгт")
		return
	}
	m := &store.Message{ConversationID: conv.ID, Sender: role, Body: in.Body, SenderName: c.Name, Attachment: in.Attachment}
	if in.ReplyTo != "" { // хариулж буй мессежийн товч хуулбарыг хамт хадгална
		orig, err := s.store.MessageByID(r.Context(), conv.ID, in.ReplyTo)
		if err != nil {
			writeErr(w, http.StatusNotFound, "хариулах мессеж олдсонгүй")
			return
		}
		m.ReplyTo, m.ReplyBody, m.ReplyName = orig.ID, short(orig.Body), orig.SenderName
	}
	if !c.IsGuest() {
		m.SenderID = c.UID
		if u, err := s.store.UserByID(r.Context(), c.UID); err == nil && u.DisplayName != "" {
			m.SenderName = u.DisplayName // токенд хэрэглэгчийн нэр л байдаг; бүлэгт жинхэнэ нэр харагдана
		}
	}
	if err := s.store.AddMessage(r.Context(), m); s.storeErr(w, r, err) {
		return
	}
	if m.Attachment != "" {
		m.AttachmentURL = s.media(m.Attachment)
	}
	if s.Publish != nil {
		s.Publish(*m)
	}
	s.notifyMessage(r.Context(), conv, m)
	writeJSON(w, http.StatusCreated, m)
}

// signAttachments — зургийн хавсралтад гарын үсэгтэй холбоос.
func (s *Server) signAttachments(msgs []store.Message) {
	for i := range msgs {
		if msgs[i].Attachment != "" {
			msgs[i].AttachmentURL = s.media(msgs[i].Attachment)
		}
	}
}

// handleChatUpload: POST /api/chat/{id}/upload — зураг (5MB хүртэл) багшийн санд chat_ угтвартай хадгална.
func (s *Server) handleChatUpload(w http.ResponseWriter, r *http.Request) {
	c, conv, _, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	if !s.chatLim.Allow(c.VisitorKey()) {
		writeErr(w, http.StatusTooManyRequests, "хэт хурдан илгээж байна")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "multipart/form-data хэлбэрээр илгээнэ үү")
		return
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			writeErr(w, http.StatusBadRequest, "файл олдсонгүй")
			return
		}
		if part.FileName() == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(part.FileName()))
		if !map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}[ext] {
			writeErr(w, http.StatusBadRequest, "зөвхөн зураг (JPG, PNG, GIF, WebP)")
			return
		}
		u, err := s.store.UserByID(r.Context(), conv.TeacherID)
		if s.storeErr(w, r, err) {
			return
		}
		info, err := s.files.Save(conv.TeacherID, files.Private, "chat_"+tail(conv.ID, 6)+"_"+part.FileName(), part, s.quotaOf(u))
		if err != nil {
			s.filesErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"path": info.Path, "url": s.media(info.Path)})
		return
	}
}

// handleCourseChat: сургалтын бүлэг чат. Багш нээнэ (байхгүй бол үүсгэнэ), элссэн суралцагч орно.
func (s *Server) handleCourseChat(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	role := store.SenderVisitor
	if course.TeacherID == c.UID {
		role = store.SenderTeacher
	} else {
		member, err := s.courseMember(r.Context(), c.UID, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		if !member || !course.Published {
			writeErr(w, http.StatusForbidden, "бүлэг чатад зөвхөн сургалтад элссэн суралцагчид орно")
			return
		}
	}
	// Бүлэг нь сургалт бүрт нэг; багш эсвэл элссэн суралцагчийн аль нь түрүүлж орсон ч үүснэ.
	conv, err := s.store.GetOrCreateGroupConversation(r.Context(), course.TeacherID, course.ID, course.Title)
	if s.storeErr(w, r, err) {
		return
	}
	if conv.LastMessage == "" { // шинээр нээгдсэн: эхний мессеж багшийн нэрээр, юу болохыг хэлнэ
		name := "Багш"
		if t, err := s.store.UserByID(r.Context(), course.TeacherID); err == nil && t.DisplayName != "" {
			name = t.DisplayName
		}
		m := &store.Message{ConversationID: conv.ID, Sender: store.SenderTeacher, SenderID: course.TeacherID, SenderName: name,
			Body: "👋 «" + course.Title + "» сургалтын бүлэг чат. Хичээлтэй холбоотой асуулт, санал бодлоо энд хуваалцаарай."}
		if err := s.store.AddMessage(r.Context(), m); s.storeErr(w, r, err) {
			return
		}
		if s.Publish != nil {
			s.Publish(*m)
		}
		conv.LastMessage, conv.LastMessageAt = m.Body, m.CreatedAt
	}
	msgs, err := s.store.Messages(r.Context(), conv.ID, "", 50)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": role})
}

func (s *Server) handleMyConversations(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	convs, err := s.store.TeacherConversations(r.Context(), c.UID, 100)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, convs)
}

// handleWS: /api/chat/ws?token=... — шинэ мессежийг бодит цагт түлхэнэ.
// Мессеж илгээхдээ HTTP POST ашиглана (баталгаатай, rate-limit-тэй).
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := s.tokens.Verify(r.URL.Query().Get("token"))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "токен хүчингүй")
		return
	}
	// Сервер түвшний Read/WriteTimeout урт хугацааны WebSocket-ийг таслахгүй байх.
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Time{})
	_ = rc.SetWriteDeadline(time.Time{})
	conn, err := websocket.Accept(w, r, nil) // ижил origin-оос л зөвшөөрнө
	if err != nil {
		return
	}
	conn.SetReadLimit(1024)
	// Бүртгэлтэй хэрэглэгчийн VisitorKey нь UserKey-тэй ижил тул мэдэгдэл ч энэ сувгаар ирнэ.
	keys := []string{chat.VisitorKey(c.VisitorKey())}
	if c.Role == string(store.RoleTeacher) {
		keys = append(keys, chat.TeacherKey(c.UID))
	}
	// Бүлэг чатууд: холбогдох үеийн гишүүнчлэлээр бүртгэнэ (шинэ бүлэгт орвол клиент дахин холбогдоно).
	if !c.IsGuest() {
		gctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		groups, err := s.store.UserGroupConversations(gctx, c.UID, 200)
		if err == nil && c.Role == string(store.RoleTeacher) {
			var mine []store.Conversation
			if mine, err = s.store.TeacherConversations(gctx, c.UID, 200); err == nil {
				groups = append(groups, mine...)
			}
		}
		cancel()
		if err != nil {
			s.log.Warn("ws: бүлэг чат уншиж чадсангүй", "err", err)
		}
		for _, g := range groups {
			if g.Kind == store.ConvGroup {
				keys = append(keys, chat.GroupKey(g.ID))
			}
		}
	}
	client := chat.NewClient()
	s.hub.Subscribe(client, keys...)
	defer s.hub.Unsubscribe(client, keys...)

	ctx := conn.CloseRead(context.Background()) // клиент хаахад ctx цуцлагдана
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done:
			conn.Close(websocket.StatusPolicyViolation, "slow consumer")
			return
		case msg := <-client.Send:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		case <-ping.C:
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
