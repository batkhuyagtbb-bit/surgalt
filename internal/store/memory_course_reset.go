package store

import (
	"context"
	"strings"
	"time"
)

// ResetCourseStudent — course_reset.go-гийн санах ойн хувилбар: ижил урьдчилсан нөхцөл, ижил дараалал (runReset —
// устгал эхэлмэгц төлсөн эрх заавал сэргэнэ), sub-store бүрийн түгжээн дор дараалан (хэзээ ч давхарлахгүй) устгана.
// Ялгаа: санах ойн store устгасан хичээлийг хадгалдаггүй тул зөвхөн одоогийн хичээлүүдийн (m.lessons) явц, тэмдэглэл,
// үүлэн үг, самбар, сануулгыг устгана (ClickHouse устгагдсан хичээлийнхийг ч устгана); тэдгээрийг санах ойн уншигчид
// m.lessons-оор хайдаг тул харагддаггүй.
func (m *Memory) ResetCourseStudent(ctx context.Context, courseID, userID, by string) (ResetResult, error) {
	var res ResetResult
	m.cmetaMu.Lock() // "cmeta:"+courseID түгжээтэй адил: шалгалтаас эцсийн бичилт хүртэл архивын төлөв өөрчлөгдөхгүй
	defer m.cmetaMu.Unlock()

	m.mu.RLock()
	cur, ok := m.cmeta[[2]string{courseID, userID}]
	lids := map[string]bool{}
	for _, l := range m.lessons[courseID] {
		lids[l.ID] = true
	}
	passes := m.paidPassesLocked(courseID, userID) // устгалаас ӨМНӨ
	m.mu.RUnlock()
	if err := resetAllowed(cur, ok, time.Now()); err != nil {
		return res, err
	}
	err := runReset(ctx, resetSteps{
		wipe: func(context.Context) error {
			n, err := m.resetWipe(courseID, userID, lids)
			res.Wiped = n > 0
			return err
		},
		release: func() {},
		// Санах ойд устгал синхрон тул force-ийн ялгаа гарахгүй (хойшилсон mutation байхгүй); ClickHouse-тэй адил
		// устгалын хооронд төлөгдсөн захиалгыг дахин уншиж нэгтгэнэ.
		restore: func(ctx context.Context, force bool) error {
			m.mu.RLock()
			all := unionPasses(passes, m.paidPassesLocked(courseID, userID))
			m.mu.RUnlock()
			return restorePasses(ctx, func(ctx context.Context, lid string, _ bool, keys []string) error {
				return m.GrantLessonPass(ctx, userID, courseID, lid, keys...)
			}, all, force)
		},
		finish: func(context.Context) error {
			mt := resetMeta(userID, by, time.Now())
			res.At = *mt.ResetAt
			m.mu.Lock()
			m.putMetaLocked(courseID, mt)
			m.mu.Unlock()
			return nil
		},
	})
	if err != nil {
		return ResetResult{}, err
	}
	return res, nil
}

// paidPassesLocked — m.mu түгжээтэй үед: суралцагчийн энэ сургалтад төлсөн хоцролт/оролцооны захиалгууд.
func (m *Memory) paidPassesLocked(courseID, userID string) []paidPass {
	var out []paidPass
	for _, o := range m.orders {
		if o.UserID == userID && o.CourseID == courseID && o.Status == OrderPaid && o.LessonID != "" &&
			(o.Kind == OrderKindLate || o.Kind == OrderKindFee) {
			out = append(out, paidPass{o.LessonID, o.Kind})
		}
	}
	return out
}

// resetFault — тестэд: устгалын алхмын (step) дараа алдаа оруулах (nil бол үгүй).
func (m *Memory) resetFault(step string) error {
	if m.resetFaults == nil {
		return nil
	}
	return m.resetFaults(step)
}

// resetWipe — устгасан зүйлийн тоог буцаана. Эхэнд ("start") ба алхам бүрийн дараа resetFault шалгана ("lesson_progress",
// "learning", "engagement").
func (m *Memory) resetWipe(courseID, userID string, lids map[string]bool) (int, error) {
	if err := m.resetFault("start"); err != nil { // юу ч устгаагүй үеийн алдаа
		return 0, err
	}
	n := 0
	m.mu.Lock()
	for lid := range lids {
		k := [2]string{userID, lid}
		if p, ok := m.progress[k]; ok {
			delete(m.progress, k) // зөвхөн эрхийн мөрийг ч (дараа нь сэргээнэ), гэхдээ өгөгдөлд тооцохгүй (ClickHouse-ийн probe-той адил)
			if p.Viewed() {
				n++
			}
		}
		if _, ok := m.hl[k]; ok {
			delete(m.hl, k)
			n++
		}
		delete(m.hlAt, k)
	}
	for k := range m.cloud { // (хичээл, хэсэг, суралцагч)
		if k[2] == userID && lids[k[0]] {
			delete(m.cloud, k)
			n++
		}
	}
	for k := range m.boards { // (хичээл, хэсэг, өрөө)
		if k[2] == "u:"+userID && lids[k[0]] {
			delete(m.boards, k)
			n++
		}
	}
	for k := range m.reviews { // user|lesson|stage
		if parts := strings.Split(k, "|"); len(parts) == 3 && parts[0] == userID && lids[parts[1]] {
			delete(m.reviews, k)
			n++
		}
	}
	// Бүлэг чатын харилцан үнэлгээ: тэр суралцагчийн энэ сургалтад өгсөн ба авсан.
	n += m.deletePeersLocked(func(k peerKey) bool { return k[0] == courseID && (k[1] == userID || k[2] == userID) })
	m.mu.Unlock()
	if err := m.resetFault("lesson_progress"); err != nil {
		return n, err
	}

	l := &m.learn
	l.mu.Lock()
	for id, a := range l.attempts {
		if a.UserID == userID && a.CourseID == courseID {
			delete(l.attempts, id)
			n++
		}
	}
	for id, s := range l.sessions {
		if s.UserID == userID && s.CourseID == courseID {
			delete(l.sessions, id)
			n++
		}
	}
	evs := l.events[:0]
	for _, e := range l.events {
		if e.UserID != userID || e.CourseID != courseID {
			evs = append(evs, e)
		}
	}
	n += len(l.events) - len(evs)
	clear(l.events[len(evs):])
	l.events = evs
	for id, s := range l.subs {
		if s.UserID == userID && (s.CourseID == courseID || lids[s.LessonID]) {
			delete(l.subs, id)
			n++
		}
	}
	if _, ok := l.rankPts[[2]string{userID, courseID}]; ok {
		delete(l.rankPts, [2]string{userID, courseID})
		n++
	}
	l.mu.Unlock()
	if err := m.resetFault("learning"); err != nil {
		return n, err
	}

	e := &m.engage
	e.mu.Lock()
	qs := e.quiz[:0]
	for _, q := range e.quiz {
		if q.UserID != userID || q.CourseID != courseID {
			qs = append(qs, q)
		}
	}
	n += len(e.quiz) - len(qs)
	clear(e.quiz[len(qs):])
	e.quiz = qs
	for id, r := range e.refl {
		if r.UserID == userID && r.CourseID == courseID {
			delete(e.refl, id)
			n++
		}
	}
	for id, w := range e.watches {
		if w.UserID == userID && w.CourseID == courseID {
			delete(e.watches, id)
			n++
		}
	}
	e.mu.Unlock()
	return n, m.resetFault("engagement")
}
