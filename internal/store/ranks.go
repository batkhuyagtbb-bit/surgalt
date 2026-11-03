package store

import "context"

// TeacherRank — нүүр хуудасны «Шилдэг багш нар» зэрэглэлийн үзүүлэлт (багш бүрээр).
type TeacherRank struct {
	ID       string
	Views    int64 // профайлын нийт үзэлт
	Courses  int64 // нийтлэгдсэн сургалт
	Students int64 // сургалтад элссэн давтагдаагүй суралцагч
	Enrolls  int64 // нийт элсэлт (сургалт авсан тоо)
	Talks    int64 // харилцсан давтагдаагүй хүн: багшид бичсэн (чат) + хичээлд сэтгэгдэл үлдээсэн
}

func (m *Memory) TeacherRanks(_ context.Context) (map[string]*TeacherRank, error) {
	out := map[string]*TeacherRank{}
	at := func(id string) *TeacherRank {
		r := out[id]
		if r == nil {
			r = &TeacherRank{ID: id}
			out[id] = r
		}
		return r
	}
	talk := map[string]map[string]bool{}
	addTalk := func(teacher, who string) {
		if teacher == "" || who == "" || who == teacher {
			return
		}
		if talk[teacher] == nil {
			talk[teacher] = map[string]bool{}
		}
		talk[teacher][who] = true
	}
	m.mu.RLock()
	for id, u := range m.users {
		if u.ProfileViews > 0 {
			at(id).Views = u.ProfileViews
		}
	}
	for _, c := range m.courses {
		if c.Published {
			at(c.TeacherID).Courses++
		}
	}
	students := map[string]map[string]bool{}
	for k := range m.enrollments {
		c, ok := m.courses[k[1]]
		if !ok {
			continue
		}
		r := at(c.TeacherID)
		r.Enrolls++
		if students[c.TeacherID] == nil {
			students[c.TeacherID] = map[string]bool{}
		}
		students[c.TeacherID][k[0]] = true
	}
	for t, s := range students {
		at(t).Students = int64(len(s))
	}
	for _, msgs := range m.messages {
		for _, msg := range msgs {
			if msg.Sender == SenderTeacher {
				continue
			}
			who := msg.SenderID
			if who == "" {
				who = msg.VisitorKey
			}
			addTalk(msg.TeacherID, who)
		}
	}
	m.mu.RUnlock()
	d := m.dm()
	d.mu.RLock()
	for _, c := range d.comments {
		if !c.Deleted {
			addTalk(c.TeacherID, c.UserID)
		}
	}
	d.mu.RUnlock()
	for t, w := range talk {
		at(t).Talks = int64(len(w))
	}
	return out, nil
}
