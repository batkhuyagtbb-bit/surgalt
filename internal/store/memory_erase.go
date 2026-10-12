package store

import (
	"context"
	"strings"
)

func (m *Memory) DeleteUserData(_ context.Context, uid string) error {
	key := "u:" + uid
	mine := func(msg *Message) bool {
		return msg.SenderID == uid || (msg.SenderID == "" && msg.Sender == SenderVisitor && msg.VisitorKey == key)
	}
	m.mu.Lock()
	u, ok := m.users[uid]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	delete(m.byEmail, strings.ToLower(u.Email))
	delete(m.byUsername, u.Username)
	delete(m.users, uid)
	for k, v := range m.identities {
		if v == uid {
			delete(m.identities, k)
		}
	}
	for k := range m.enrollments {
		if k[0] == uid {
			delete(m.enrollments, k)
		}
	}
	for _, mp := range []map[[2]string]string{m.lessonAcc, m.pendingL, m.pending, m.meetAcc} {
		for k := range mp {
			if k[0] == uid {
				delete(mp, k)
			}
		}
	}
	for k := range m.progress {
		if k[0] == uid {
			delete(m.progress, k)
		}
	}
	delete(m.notifs, uid)
	for _, members := range m.teamMembers {
		delete(members, uid)
	}
	gone := map[string]bool{}
	for _, list := range m.messages {
		for _, msg := range list {
			if mine(msg) {
				gone[msg.ID] = true
				msg.Body, msg.Attachment, msg.ReplyBody, msg.SenderName, msg.Deleted = "", "", "", DeletedName, true
			}
		}
	}
	for _, list := range m.messages {
		for _, msg := range list {
			if gone[msg.ReplyTo] {
				msg.ReplyBody, msg.ReplyName = "", DeletedName
			}
		}
	}
	for _, c := range m.convs {
		if c.UserID == uid && c.Kind == "" {
			c.VisitorName, c.LastMessage = DeletedName, ""
		}
	}
	for _, s := range m.slots {
		if s.StudentID == uid {
			s.StudentName, s.Note = DeletedName, ""
		}
	}
	for id, v := range m.visits {
		if v.VisitorID == uid {
			delete(m.visits, id)
		}
	}
	m.mu.Unlock()

	l := &m.learn
	l.mu.Lock()
	for id, a := range l.attempts {
		if a.UserID == uid {
			delete(l.attempts, id)
		}
	}
	for id, s := range l.sessions {
		if s.UserID == uid {
			delete(l.sessions, id)
		}
	}
	evs := l.events[:0]
	for _, e := range l.events {
		if e.UserID != uid {
			evs = append(evs, e)
		}
	}
	l.events = evs
	for k := range l.rankPts {
		if k[0] == uid {
			delete(l.rankPts, k)
		}
	}
	delete(l.rankLvl, uid)
	for id, s := range l.subs {
		if s.UserID == uid {
			delete(l.subs, id)
		}
	}
	l.mu.Unlock()

	e := &m.engage
	e.mu.Lock()
	qs := e.quiz[:0]
	for _, q := range e.quiz {
		if q.UserID != uid {
			qs = append(qs, q)
		}
	}
	e.quiz = qs
	for id, r := range e.refl {
		if r.UserID == uid {
			delete(e.refl, id)
		}
	}
	for id, w := range e.watches {
		if w.UserID == uid {
			delete(e.watches, id)
		}
	}
	e.mu.Unlock()

	b := &m.books
	b.mu.Lock()
	for k := range b.access {
		if k[0] == uid {
			delete(b.access, k)
		}
	}
	for k := range b.pending {
		if k[0] == uid {
			delete(b.pending, k)
		}
	}
	bes := b.events[:0]
	for _, ev := range b.events {
		if ev.UserID != uid {
			bes = append(bes, ev)
		}
	}
	b.events = bes
	b.mu.Unlock()

	d := m.dm()
	d.mu.Lock()
	for id, c := range d.comments {
		if c.UserID == uid {
			delete(d.comments, id)
		}
	}
	for k := range d.likes {
		if k[2] == uid {
			delete(d.likes, k)
		}
	}
	d.mu.Unlock()

	x := m.cx()
	x.mu.Lock()
	for _, users := range x.reactions {
		delete(users, key)
	}
	for k := range x.emoji {
		if k[1] == key {
			delete(x.emoji, k)
		}
	}
	for k := range x.reads {
		if k[1] == key {
			delete(x.reads, k)
		}
	}
	x.mu.Unlock()
	return nil
}
