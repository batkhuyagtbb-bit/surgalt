package httpapi

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"surgalt/internal/chat"
	"surgalt/internal/store"
)

// Багшийн онлайн төлөв: нэвтэрсэн хэрэглэгч сайтын аль ч хуудсанд нэг WebSocket холболттой (чат, мэдэгдэл),
// тиймээс нээлттэй холболт = одоо сайтад байна. Сүүлд гарсан мөчийг санах ойд хадгална (сервер дахин асахад цэвэрлэгдэнэ).
type presence struct {
	last sync.Map // uid -> time.Time
	book contactBook
}

func (p *presence) left(uid string) { p.last.Store(uid, time.Now().UTC()) }

func (p *presence) lastSeen(uid string) (time.Time, bool) {
	v, ok := p.last.Load(uid)
	if !ok {
		return time.Time{}, false
	}
	return v.(time.Time), true
}

// contactBook — хэрэглэгч бүрд чатын жагсаалтад нь аль хэдийн харуулсан хүмүүс (яриа, суралцагчид, багш нар,
// ангийн найзууд). Онлайн төлөвийг зөвхөн тэдэнд нь өгнө: гадны хүн ID таамаглаж хэн сайтад байгааг шалгаж чадахгүй.
// Санах ойд л хадгална; 24 цаг жагсаалтаа ачаалаагүй хэрэглэгчийнх цэвэрлэгдэнэ.
type contactBook struct {
	mu    sync.Mutex
	m     map[string]*contactSet
	prune time.Time
}

type contactSet struct {
	ids map[string]struct{}
	at  time.Time
}

const (
	contactTTL  = 24 * time.Hour
	maxContacts = 5000
)

func (b *contactBook) add(uid string, ids ...string) {
	if uid == "" || len(ids) == 0 {
		return
	}
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		b.m = map[string]*contactSet{}
	}
	if now.Sub(b.prune) > 10*time.Minute {
		b.prune = now
		for k, cs := range b.m {
			if now.Sub(cs.at) > contactTTL {
				delete(b.m, k)
			}
		}
	}
	cs := b.m[uid]
	if cs == nil {
		cs = &contactSet{ids: map[string]struct{}{}}
		b.m[uid] = cs
	}
	cs.at = now
	for _, id := range ids {
		if id != "" && id != uid && len(cs.ids) < maxContacts {
			cs.ids[id] = struct{}{}
		}
	}
}

func (b *contactBook) list(uid string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	cs := b.m[uid]
	if cs == nil || time.Since(cs.at) > contactTTL {
		return nil
	}
	out := make([]string, 0, len(cs.ids))
	for id := range cs.ids {
		out = append(out, id)
	}
	return out
}

// knowConvs — хувийн яриа, ангийн найзын ярианы нөгөө талыг харилцагчид нэмнэ (бүлэг чатыг үл тооно).
func (s *Server) knowConvs(uid string, convs []store.Conversation) {
	ids := make([]string, 0, len(convs))
	for _, cv := range convs {
		switch cv.Kind {
		case "", store.ConvDM:
			if cv.TeacherID == uid {
				ids = append(ids, cv.UserID)
			} else {
				ids = append(ids, cv.TeacherID)
			}
		}
	}
	s.seen.book.add(uid, ids...)
}

// handleMyPresence: GET /api/me/presence → {"online": [uid…], "seen": {uid: time}} — чатын жагсаалтын хүмүүсээс
// хэн одоо сайтад байгаа (нээлттэй WebSocket), хэн сүүлийн 24 цагт идэвхтэй байсан.
func (s *Server) handleMyPresence(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	online, seen := []string{}, map[string]time.Time{}
	for _, id := range s.seen.book.list(c.UID) {
		if s.hub.Online(chat.TeacherKey(id)) { // бүртгэлтэй хэрэглэгч бүр өөрийн ID-н сувагт холбогддог
			online = append(online, id)
		} else if t, ok := s.seen.lastSeen(id); ok && time.Since(t) < 24*time.Hour {
			seen[id] = t
		}
	}
	sort.Strings(online)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"online": online, "seen": seen})
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
