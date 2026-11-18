package httpapi

import "testing"

// Анхаарлын индекс: төвлөрөл 60% + видео үзэлт 40%; видео үзээгүй бол зөвхөн төвлөрөл, сессгүй бол зөвхөн видео.
// Суралцахуйн оноонд видео үзэлт орохгүй.
func TestAttentionScore(t *testing.T) {
	cases := []struct {
		st   StudentStat
		want int
	}{
		{StudentStat{Focus: 80, TotalSec: 600, VideoClips: 2, VideoCoverage: 50}, 68},
		{StudentStat{Focus: 72, TotalSec: 600}, 72},
		{StudentStat{VideoClips: 1, VideoCoverage: 40}, 40},
		{StudentStat{Focus: 100, TotalSec: 600, VideoClips: 1, VideoCoverage: 100}, 100},
	}
	for i, c := range cases {
		if got := c.st.attentionScore(); got != c.want {
			t.Fatalf("%d: %d, хүлээсэн %d", i, got, c.want)
		}
	}
	a := StudentStat{QuizTotal: 4, QuizAccuracy: 60, Lessons: 2, Reflections: 1, ActiveDays: 1, StreakDays: 1}
	b := a
	b.VideoClips, b.VideoCoverage = 3, 0
	if a.learnScore() != b.learnScore() || a.learnScore() != 43 { // (60 + 50 + 20) / 3
		t.Fatalf("суралцахуйн оноо видеоноос хамаарахгүй: %d %d", a.learnScore(), b.learnScore())
	}
}
