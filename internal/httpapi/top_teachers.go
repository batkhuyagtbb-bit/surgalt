package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"surgalt/internal/store"
)

// ---- Нүүр хуудасны «Шилдэг багш нар»: 4 зэрэглэл (тус бүр 10 багш) ----
//   views    — профайл хамгийн их үзэгдсэн
//   students — сургалт нь хамгийн их авагдсан (элсэлт)
//   courses  — хамгийн олон нийтлэгдсэн сургалттай
//   talks    — хамгийн олон хүнтэй харилцсан (чат, хичээлийн сэтгэгдэл)
// Бүх нийтэд нэг ижил тул 10 минут кэшлэнэ.

const topTTL = 10 * time.Minute

type topCache struct {
	mu   sync.Mutex
	at   time.Time
	body []byte
}

type topTeacher struct {
	Rank        int      `json:"rank"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Headline    string   `json:"headline"`
	AvatarURL   string   `json:"avatar_url"`
	Subjects    []string `json:"subjects"`
	Value       int64    `json:"value"`
	Views       int64    `json:"views"`
	Courses     int64    `json:"courses"`
	Students    int64    `json:"students"`
	Enrolls     int64    `json:"enrolls"`
	Talks       int64    `json:"talks"`
}

func (s *Server) topTeachers(ctx context.Context) (map[string][]topTeacher, error) {
	ranks, err := s.store.TeacherRanks(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(ranks))
	for id := range ranks {
		ids = append(ids, id)
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	type row struct {
		u *store.User
		r *store.TeacherRank
	}
	rows := make([]row, 0, len(users))
	for i := range users {
		u := &users[i]
		if u.Role != store.RoleTeacher || u.Org != nil { // байгууллага багшийн жагсаалтад орохгүй
			continue
		}
		rows = append(rows, row{u, ranks[u.ID]})
	}
	board := func(val func(*store.TeacherRank) int64, tie func(*store.TeacherRank) int64) []topTeacher {
		rs := make([]row, 0, len(rows))
		for _, x := range rows {
			if val(x.r) > 0 {
				rs = append(rs, x)
			}
		}
		sort.SliceStable(rs, func(i, j int) bool {
			a, b := rs[i], rs[j]
			if val(a.r) != val(b.r) {
				return val(a.r) > val(b.r)
			}
			if tie(a.r) != tie(b.r) {
				return tie(a.r) > tie(b.r)
			}
			return a.u.CreatedAt.Before(b.u.CreatedAt)
		})
		if len(rs) > 10 {
			rs = rs[:10]
		}
		out := make([]topTeacher, len(rs))
		for i, x := range rs {
			subj := x.u.Subjects
			if len(subj) > 3 {
				subj = subj[:3]
			}
			if subj == nil {
				subj = []string{}
			}
			out[i] = topTeacher{Rank: i + 1, Username: x.u.Username, DisplayName: x.u.DisplayName, Headline: x.u.Headline, AvatarURL: x.u.AvatarURL,
				Subjects: subj, Value: val(x.r), Views: x.r.Views, Courses: x.r.Courses, Students: x.r.Students, Enrolls: x.r.Enrolls, Talks: x.r.Talks}
		}
		return out
	}
	views := func(r *store.TeacherRank) int64 { return r.Views }
	enrolls := func(r *store.TeacherRank) int64 { return r.Enrolls }
	return map[string][]topTeacher{
		"views":    board(views, enrolls),
		"students": board(enrolls, func(r *store.TeacherRank) int64 { return r.Students }),
		"courses":  board(func(r *store.TeacherRank) int64 { return r.Courses }, enrolls),
		"talks":    board(func(r *store.TeacherRank) int64 { return r.Talks }, views),
	}, nil
}

// handleTopTeachers: GET /api/top-teachers
func (s *Server) handleTopTeachers(w http.ResponseWriter, r *http.Request) {
	s.top.mu.Lock()
	defer s.top.mu.Unlock()
	if s.top.body == nil || time.Since(s.top.at) > topTTL {
		boards, err := s.topTeachers(r.Context())
		if err != nil {
			if s.top.body == nil {
				s.storeErr(w, r, err)
				return
			}
			s.log.Warn("шилдэг багш нар", "err", err) // хуучин кэшээ үзүүлсээр байна
		} else {
			b, _ := json.Marshal(map[string]any{"boards": boards, "updated_at": time.Now()})
			s.top.body, s.top.at = b, time.Now()
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(s.top.body)
}
