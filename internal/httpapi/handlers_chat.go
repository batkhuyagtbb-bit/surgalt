package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
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
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": store.SenderVisitor})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	_, conv, role, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	msgs, err := s.store.Messages(r.Context(), conv.ID, r.URL.Query().Get("before"), 50)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv, "messages": msgs, "me": role})
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
		Body string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > 2000 {
		writeErr(w, http.StatusBadRequest, "мессеж 1-2000 тэмдэгт")
		return
	}
	m := &store.Message{ConversationID: conv.ID, Sender: role, Body: in.Body, SenderName: c.Name}
	if !c.IsGuest() {
		m.SenderID = c.UID
		if u, err := s.store.UserByID(r.Context(), c.UID); err == nil && u.DisplayName != "" {
			m.SenderName = u.DisplayName // токенд хэрэглэгчийн нэр л байдаг; бүлэгт жинхэнэ нэр харагдана
		}
	}
	if err := s.store.AddMessage(r.Context(), m); s.storeErr(w, r, err) {
		return
	}
	if s.Publish != nil {
		s.Publish(*m)
	}
	s.notifyMessage(r.Context(), conv, m)
	writeJSON(w, http.StatusCreated, m)
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
