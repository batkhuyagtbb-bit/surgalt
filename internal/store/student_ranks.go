package store

import (
	"context"
	"strings"
)

// StudentRank — нүүр хуудасны «Идэвхтэй суралцагчид» зэрэглэлийн үзүүлэлт (хэрэглэгч бүрээр). Өөрийн сургалт дахь
// үйлдлийг (багш өөрийнхөө хичээлийг туршсан) тооцохгүй.
type StudentRank struct {
	ID        string
	Courses   int64 // суралцсан (хичээл үзсэн) сургалт
	Points    int64 // цолын оноо — сургалтуудын онооны дундаж
	Creative  int64 // бүтээл: хичээлийн дүгнэлт + даалгавар + үүлэн үгийн хариулт + тодруулга
	ActiveMin int64 // хичээл дээрх нийт идэвхтэй минут
	CleanMin  int64 // сануулгагүй (зөрчилгүй) хичээлийн идэвхтэй минут
	Warnings  int64 // сануулга авсан тоо
	Talks     int64 // бичсэн мессеж + хичээлийн сэтгэгдэл
}

// isWarn — study_sessions.counts-ийн түлхүүр сануулгад тооцогдох эсэх.
func isWarn(k string, warn []string) bool {
	for _, w := range warn {
		if strings.EqualFold(k, w) {
			return true
		}
	}
	return false
}

func (m *Memory) StudentRanks(_ context.Context, warn []string) (map[string]*StudentRank, error) {
	out := map[string]*StudentRank{}
	at := func(id string) *StudentRank {
		r := out[id]
		if r == nil {
			r = &StudentRank{ID: id}
			out[id] = r
		}
		return r
	}
	m.mu.RLock()
	courseT := map[string]string{}
	for id, c := range m.courses {
		courseT[id] = c.TeacherID
	}
	lessonT := map[string]string{}
	for cid, ls := range m.lessons {
		for _, l := range ls {
			lessonT[l.ID] = courseT[cid]
		}
	}
	lessonC := map[string]string{}
	for cid, ls := range m.lessons {
		for _, l := range ls {
			lessonC[l.ID] = cid
		}
	}
	seen := map[[2]string]bool{}
	for k := range m.progress {
		cid := lessonC[k[1]]
		if cid != "" && courseT[cid] != k[0] && !seen[[2]string{k[0], cid}] {
			seen[[2]string{k[0], cid}] = true
			at(k[0]).Courses++
		}
	}
	for k, hs := range m.hl {
		if lessonT[k[1]] != k[0] {
			at(k[0]).Creative += int64(len(hs))
		}
	}
	for k := range m.cloud {
		if lessonT[k[0]] != k[2] {
			at(k[2]).Creative++
		}
	}
	for _, msgs := range m.messages {
		for _, msg := range msgs {
			if msg.Sender != SenderTeacher && msg.SenderID != "" && msg.SenderID != msg.TeacherID {
				at(msg.SenderID).Talks++
			}
		}
	}
	m.mu.RUnlock()
	l := &m.learn
	l.mu.RLock()
	pts := map[string][2]int64{}
	for k, p := range l.rankPts {
		if p > 0 && courseT[k[1]] != k[0] {
			v := pts[k[0]]
			pts[k[0]] = [2]int64{v[0] + int64(p), v[1] + 1}
		}
	}
	for _, s := range l.subs {
		if s.TeacherID != s.UserID {
			at(s.UserID).Creative++
		}
	}
	for _, s := range l.sessions {
		if s.TeacherID == s.UserID {
			continue
		}
		r, clean := at(s.UserID), true
		r.ActiveMin += int64(s.ActiveSec)
		for k, n := range s.Counts {
			if n > 0 && isWarn(k, warn) {
				clean = false
			}
		}
		if clean {
			r.CleanMin += int64(s.ActiveSec)
		}
	}
	for _, e := range l.events {
		if e.TeacherID != e.UserID && isWarn(e.Type, warn) {
			at(e.UserID).Warnings++
		}
	}
	l.mu.RUnlock()
	for u, v := range pts {
		at(u).Points = v[0] / v[1]
	}
	e := &m.engage
	e.mu.RLock()
	for _, r := range e.refl {
		if r.TeacherID != r.UserID {
			at(r.UserID).Creative++
		}
	}
	e.mu.RUnlock()
	d := m.dm()
	d.mu.RLock()
	for _, c := range d.comments {
		if !c.Deleted && c.UserID != c.TeacherID {
			at(c.UserID).Talks++
		}
	}
	d.mu.RUnlock()
	for _, r := range out {
		r.ActiveMin /= 60
		r.CleanMin /= 60
	}
	return out, nil
}
