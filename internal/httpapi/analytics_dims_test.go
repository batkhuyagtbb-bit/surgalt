package httpapi

import (
	"testing"
	"time"
)

// 8 хэмжүүр: өгөгдөлгүй тал −1 (дундажид орохгүй), утга 0-100 хооронд; ангийн дундаж зөвхөн өгөгдөлтэй хүмүүсээр.
func TestDimScores(t *testing.T) {
	none := (&StudentStat{ExamBest: -1}).dimScores()
	for i, v := range none {
		if v != -1 {
			t.Fatalf("өгөгдөлгүй суралцагчийн %s = %d, −1 байх ёстой", dimNames[i], v)
		}
	}
	a := &StudentStat{TotalSec: 600, ActivePct: 80, ActiveDays: 5, Attention: 90, QuizTotal: 10, QuizAccuracy: 90, QuizQuestions: 8, QuizFirstTry: 75,
		QuizGuesses: 1, VideoClips: 2, VideoCoverage: 70, ExamBest: 85, Lessons: 4, Reflections: 2, ReflectionWords: 24, BoardDays: 1, CollabScore: 60, ReviewDue: 4, ReviewDone: 3}
	d := a.dimScores()
	want := []int{88, 90, 79, 70, 85, 50, 60, 75} // 0.6·80+0.4·100; 90; 0.6·90+0.4·75−5; 70; 85; 2/4·100·1; 60; 3/4
	for i := range want {
		if d[i] != want[i] {
			t.Fatalf("%s = %d, хүлээсэн %d (%v)", dimNames[i], d[i], want[i], d)
		}
	}
	a.Dims = d
	b := &StudentStat{ExamBest: -1, TotalSec: 300, ActivePct: 40, Attention: 30}
	b.Dims = b.dimScores()
	avg := avgDims([]*StudentStat{a, b})
	if avg[1] != 60 || avg[3] != 70 || avg[6] != 60 {
		t.Fatalf("ангийн дундаж: %v (анхаарал 60, видео зөвхөн a-гаар 70, багаар 60)", avg)
	}
	if got := avgDims(nil); got[0] != -1 {
		t.Fatalf("хоосон анги: %v", got)
	}
}

// Мартах муруй: анх үзсэнээс 1, 3, 7… хоногийн дараах үе шат бүрт тэр хичээлдээ эргэж орсон эсэх. Хугацаа нь болоогүй,
// эсвэл тайлант хугацаанаас өмнөх үе шат тоологдохгүй; сануулгаас 12 цагийн өмнөөс дараагийн үе шат хүртэл орсон бол давтсан.
func TestCurveReviews(t *testing.T) {
	viewed := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	now := viewed.AddDate(0, 0, 10) // 1, 3, 7 хоногийн үе шат болсон; 14, 30 болоогүй
	visits := []time.Time{viewed.AddDate(0, 0, 1).Add(2 * time.Hour), viewed.AddDate(0, 0, 8)} // 1-р үе шат, 7-р үе шат
	if due, done := curveReviews(viewed, visits, viewed.AddDate(0, 0, -30), now); due != 3 || done != 2 {
		t.Fatalf("due=%d done=%d, хүлээсэн 3/2", due, done)
	}
	if due, _ := curveReviews(viewed, visits, viewed.AddDate(0, 0, 5), now); due != 1 {
		t.Fatalf("тайлант хугацаанаас өмнөх үе шат тоологдох ёсгүй: due=%d", due)
	}
	if due, _ := curveReviews(viewed, nil, viewed.AddDate(0, 0, -30), viewed.AddDate(0, 0, 1).Add(time.Hour)); due != 0 {
		t.Fatalf("24 цаг өнгөрөөгүй үе шат тоологдох ёсгүй: due=%d", due)
	}
}
