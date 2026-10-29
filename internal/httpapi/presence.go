package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"surgalt/internal/chat"
	"surgalt/internal/store"
)

// Багшийн онлайн төлөв: нэвтэрсэн хэрэглэгч сайтын аль ч хуудсанд нэг WebSocket холболттой (чат, мэдэгдэл),
// тиймээс нээлттэй холболт = одоо сайтад байна. Сүүлд гарсан мөчийг санах ойд хадгална (сервер дахин асахад цэвэрлэгдэнэ).
type presence struct{ last sync.Map } // uid -> time.Time

func (p *presence) left(uid string) { p.last.Store(uid, time.Now().UTC()) }

func (p *presence) lastSeen(uid string) (time.Time, bool) {
	v, ok := p.last.Load(uid)
	if !ok {
		return time.Time{}, false
	}
	return v.(time.Time), true
}

// handleTeacherPresence: GET /api/teachers/{username}/presence → {"online": bool, "last_seen"?: time}
func (s *Server) handleTeacherPresence(w http.ResponseWriter, r *http.Request) {
	p, err := s.publicProfile(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	id := p.Data.Teacher.ID
	online := s.hub.Online(chat.TeacherKey(id))
	out := map[string]any{"online": online}
	if t, ok := s.seen.lastSeen(id); ok && !online {
		out["last_seen"] = t
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, out)
}

// handleHideChat: DELETE /api/chat/{id} — «Чат устгах»: яриаг зөвхөн өөрийн жагсаалтаас устгана.
// Өмнөх мессеж танд харагдахгүй болно; нөгөө тал яриагаа хэвээр харна, шинэ мессеж ирвэл яриа дахин гарна.
func (s *Server) handleHideChat(w http.ResponseWriter, r *http.Request) {
	c, conv, _, ok := s.loadConv(w, r)
	if !ok {
		return
	}
	if err := s.store.HideChat(r.Context(), c.UID, conv.ID, time.Now()); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hidden": true})
}

// dropHiddenConvs — хэрэглэгчийн устгасан яриаг (устгаснаас хойш шинэ мессеж ирээгүй бол) жагсаалтаас хасна.
func (s *Server) dropHiddenConvs(ctx context.Context, uid string, convs []store.Conversation) []store.Conversation {
	hides, err := s.store.ChatHides(ctx, uid)
	if err != nil || len(hides) == 0 {
		return convs
	}
	out := make([]store.Conversation, 0, len(convs))
	for _, cv := range convs {
		if at, ok := hides[cv.ID]; ok && !cv.LastMessageAt.After(at) {
			continue
		}
		out = append(out, cv)
	}
	return out
}

// dropHiddenMsgs — устгасан ярианы тэр мөчөөс өмнөх мессежийг хасна (зөвхөн устгасан хүнд).
func (s *Server) dropHiddenMsgs(ctx context.Context, uid, convID string, msgs []store.Message) []store.Message {
	if uid == "" {
		return msgs
	}
	at, err := s.store.ChatHideAt(ctx, uid, convID)
	if err != nil || at.IsZero() {
		return msgs
	}
	out := make([]store.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.CreatedAt.After(at) {
			out = append(out, m)
		}
	}
	return out
}
