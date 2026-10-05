// Package chat нь WebSocket холболтуудыг удирдаж, шинэ мессежийг
// тухайн яриаг сонсож буй бүх холболт руу түгээнэ.
package chat

import (
	"encoding/json"
	"hash/maphash"
	"strings"
	"sync"
	"sync/atomic"

	"surgalt/internal/store"
)

// Сувгийн түлхүүрүүд:
//
//	"t:<teacherID>" — багшийн бүх яриа
//	"v:<visitorKey>" — зочны өөрийн яриа
func TeacherKey(teacherID string) string    { return "t:" + teacherID }
func GroupKey(conversationID string) string { return "grp:" + conversationID } // бүлэг чатын суваг
func VisitorKey(visitorKey string) string   { return "v:" + visitorKey }

// Client нь нэг WebSocket холболт. Send дүүрвэл (удаан клиент) холболтыг хаана.
type Client struct {
	Send   chan []byte
	closed atomic.Bool
	Done   chan struct{}
}

func NewClient() *Client { return &Client{Send: make(chan []byte, 64), Done: make(chan struct{})} }

func (c *Client) kill() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.Done)
	}
}

const hubShards = 32

type hubShard struct {
	mu   sync.RWMutex
	subs map[string]map[*Client]struct{}
}

type Hub struct {
	seed   maphash.Seed
	shards [hubShards]hubShard
	conns  atomic.Int64
}

func NewHub() *Hub {
	h := &Hub{seed: maphash.MakeSeed()}
	for i := range h.shards {
		h.shards[i].subs = make(map[string]map[*Client]struct{})
	}
	return h
}

func (h *Hub) shard(key string) *hubShard { return &h.shards[maphash.String(h.seed, key)%hubShards] }

func (h *Hub) Subscribe(c *Client, keys ...string) {
	h.conns.Add(1)
	for _, k := range keys {
		s := h.shard(k)
		s.mu.Lock()
		set := s.subs[k]
		if set == nil {
			set = make(map[*Client]struct{})
			s.subs[k] = set
		}
		set[c] = struct{}{}
		s.mu.Unlock()
	}
}

func (h *Hub) Unsubscribe(c *Client, keys ...string) {
	h.conns.Add(-1)
	for _, k := range keys {
		s := h.shard(k)
		s.mu.Lock()
		if set := s.subs[k]; set != nil {
			delete(set, c)
			if len(set) == 0 {
				delete(s.subs, k)
			}
		}
		s.mu.Unlock()
	}
	c.kill()
}

func (h *Hub) Connections() int64 { return h.conns.Load() }

func (h *Hub) publish(key string, payload []byte) {
	s := h.shard(key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.subs[key] {
		select {
		case c.Send <- payload:
		default:
			c.kill() // удаан клиент бусдыг саатуулахгүй
		}
	}
}

type Event struct {
	Type    string        `json:"type"`
	Message store.Message `json:"message"`
}

// UserKey нь бүртгэлтэй хэрэглэгчийн хувийн суваг (мэдэгдэл).
func UserKey(userID string) string { return VisitorKey("u:" + userID) }

// Notify нь мэдэгдлийг тухайн хэрэглэгчийн бүх нээлттэй холболт руу түлхэнэ.
func (h *Hub) Notify(n store.Notification) {
	payload, err := json.Marshal(struct {
		Type         string             `json:"type"`
		Notification store.Notification `json:"notification"`
	}{"notification", n})
	if err != nil {
		return
	}
	h.publish(UserKey(n.UserID), payload)
}

// Deliver нь мессежийг багш болон зочны сувгууд руу түгээнэ.
func (h *Hub) Deliver(m store.Message) {
	payload, err := json.Marshal(Event{Type: "message", Message: m})
	if err != nil {
		return
	}
	h.publish(TeacherKey(m.TeacherID), payload)
	if strings.HasPrefix(m.VisitorKey, "course:") {
		h.publish(GroupKey(m.ConversationID), payload) // бүлгийн бүх гишүүнд
		return
	}
	h.publish(VisitorKey(m.VisitorKey), payload)
}
