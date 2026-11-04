package httpapi

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"surgalt/internal/chat"
)

// ---- WebSocket-ийн нэмэлт сувгууд ----
// Хичээл дээрх үүлэн үгийг 5 секунд тутам асуухын оронд (3 000 суралцагч = 600 хүсэлт/с) нээлттэй WebSocket-оор
// «энэ блокийг харж байна» гэж бүртгүүлж, шинэ хариулт ирэхэд нэгтгэлийг түлхэнэ. Түлхэлт нэг блокт cloudPushEvery-д
// нэгээс илүүгүй (олон хүн зэрэг хариулахад хүн бүрт хариулт бүрийг илгээхгүй). Холболт тасарвал клиент 30 секунд
// тутам асууна.

const (
	wsMaxTopics    = 8
	cloudPushEvery = 4 * time.Second
)

type wsTopics struct {
	mu   sync.Mutex
	keys map[string]bool
}

func (t *wsTopics) all() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.keys))
	for k := range t.keys {
		out = append(out, k)
	}
	return out
}

func cloudTopic(lessonID, bid string) string { return "cloud:" + cloudKey(lessonID, bid) }

// wsMessage — {"t":"sub"|"unsub","topic":"cloud","course","lesson","block"}. Бүртгэхийн өмнө хичээлийн эрхийг шалгана.
func (s *Server) wsMessage(ctx context.Context, uid string, client *chat.Client, topics *wsTopics, data []byte) {
	var in struct {
		T, Topic, Course, Lesson, Block string
	}
	if json.Unmarshal(data, &in) != nil || in.Topic != "cloud" || in.Lesson == "" || in.Block == "" {
		return
	}
	key := cloudTopic(in.Lesson, in.Block)
	switch in.T {
	case "unsub":
		topics.mu.Lock()
		delete(topics.keys, key)
		topics.mu.Unlock()
		s.hub.Leave(client, key)
	case "sub":
		topics.mu.Lock()
		have, n := topics.keys[key], len(topics.keys)
		topics.mu.Unlock()
		if have || n >= wsMaxTopics {
			return
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, l, err := s.lessonFor(cctx, uid, in.Course, in.Lesson, false)
		cancel()
		if err != nil || l.ID != in.Lesson || cloudBlock(l, in.Block) == nil {
			return
		}
		topics.mu.Lock()
		topics.keys[key] = true
		topics.mu.Unlock()
		s.hub.Join(client, key)
	}
}

// cloudThrottle — блок бүрийн түлхэлт: эхний хариултыг шууд, дараагийнхыг cloudPushEvery тутам нэгтгэж.
type cloudThrottle struct {
	mu sync.Mutex
	m  map[string]*bool // key → дахин түлхэх шаардлагатай эсэх (map-д байвал цаг явж байна)
}

func (s *Server) cloudChanged(lessonID, bid string) {
	key := cloudKey(lessonID, bid)
	t := &s.cloudPush
	t.mu.Lock()
	if t.m == nil {
		t.m = map[string]*bool{}
	}
	if dirty, armed := t.m[key]; armed {
		*dirty = true
		t.mu.Unlock()
		return
	}
	t.m[key] = new(bool)
	t.mu.Unlock()
	s.pushCloud(lessonID, bid)
	var fire func()
	fire = func() {
		t.mu.Lock()
		dirty := t.m[key]
		if dirty == nil || !*dirty {
			delete(t.m, key)
			t.mu.Unlock()
			return
		}
		*dirty = false
		t.mu.Unlock()
		s.pushCloud(lessonID, bid)
		time.AfterFunc(cloudPushEvery, fire)
	}
	time.AfterFunc(cloudPushEvery, fire)
}

func (s *Server) pushCloud(lessonID, bid string) {
	topic := cloudTopic(lessonID, bid)
	if !s.hub.Online(topic) { // хэн ч харахгүй байна
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	agg, err := s.cloudAgg(ctx, lessonID, bid)
	if err != nil {
		return
	}
	s.hub.Publish(topic, map[string]any{"type": "cloud", "lesson": lessonID, "block": bid, "words": agg.words, "people": len(agg.all)})
}
