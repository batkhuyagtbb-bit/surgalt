package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"surgalt/internal/store"
)

// ---- Нүүр хуудасны «Идэвхтэй суралцагчид»: 5 зэрэглэл (тус бүр 10) ----
//   courses    — олон сургалтад идэвхтэй суралцсан
//   rank       — өндөр цолтой (сургалтуудын онооны дундаж)
//   creative   — бүтээлчээр суралцдаг (дүгнэлт, даалгавар, үүлэн үг, тодруулга)
//   discipline — сахилга баттай (сануулгагүй хичээлийн идэвхтэй минут)
//   talks      — харилцаа сайтай (мессеж, хичээлийн сэтгэгдэл)
// Хувийн мэдээлэл: нэрийг товчилно (Нэр О.), профайлын холбоосгүй; хүсвэл өөрийгөө жагсаалтаас нууна.

// warnTypes — сануулгад тооцогдох үйл явдал (багшид мэдэгддэг эсвэл зөрчил).
func warnTypes() []string {
	var out []string
	for k, m := range eventInfo {
		if m.Notify || m.Violation {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

type topStudent struct {
	Rank      int    `json:"rank"`
	Key       string `json:"key"` // өөрийн картаа танихад (хэрэглэгчийн ID-г задлахгүй)
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Value     int64  `json:"value"`
	Level     string `json:"level"`
	Insignia  string `json:"insignia"`
	Courses   int64  `json:"courses"`
	Points    int64  `json:"points"`
	Creative  int64  `json:"creative"`
	ActiveMin int64  `json:"active_min"`
	Warnings  int64  `json:"warnings"`
	Talks     int64  `json:"talks"`
}

// shortName — «Батхуяг Төмөр» → «Батхуяг Т.»; «Б.Энхжаргал», нэг үгтэй нэр хэвээр; имэйл бол @-ийн өмнөх хэсэг.
func shortName(n string) string {
	n = strings.TrimSpace(n)
	if i := strings.Index(n, "@"); i > 0 {
		n = n[:i]
	}
	f := strings.Fields(n)
	switch {
	case len(f) == 0:
		return "Суралцагч"
	case len(f) == 1:
		return f[0]
	case strings.HasSuffix(f[0], ".") && len([]rune(f[0])) <= 3: // «Т. Батхуяг»
		return f[0] + f[1]
	}
	r := []rune(f[1])
	return f[0] + " " + string(r[0]) + "."
}

func (s *Server) rankKey(uid string) string { return s.tokens.MAC("top-student", uid)[:12] }

func rankLevel(points int64) (string, string) {
	name, sign := rankLadder[0].Name, rankLadder[0].Sign
	for _, st := range rankLadder {
		if points >= int64(st.Points) {
			name, sign = st.Name, st.Sign
		}
	}
	return name, sign
}

func (s *Server) topStudents(ctx context.Context) (map[string][]topStudent, error) {
	ranks, err := s.store.StudentRanks(ctx, warnTypes())
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
		r *store.StudentRank
	}
	rows := make([]row, 0, len(users))
	for i := range users {
		u := &users[i]
		if u.RankHidden || (u.Role != store.RoleStudent && u.Role != store.RoleTeacher) {
			continue
		}
		rows = append(rows, row{u, ranks[u.ID]})
	}
	board := func(val, tie func(*store.StudentRank) int64) []topStudent {
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
		out := make([]topStudent, len(rs))
		for i, x := range rs {
			lv, sign := rankLevel(x.r.Points)
			out[i] = topStudent{Rank: i + 1, Key: s.rankKey(x.u.ID), Name: shortName(x.u.DisplayName), AvatarURL: x.u.AvatarURL, Value: val(x.r),
				Level: lv, Insignia: sign, Courses: x.r.Courses, Points: x.r.Points, Creative: x.r.Creative, ActiveMin: x.r.ActiveMin,
				Warnings: x.r.Warnings, Talks: x.r.Talks}
		}
		return out
	}
	active := func(r *store.StudentRank) int64 { return r.ActiveMin }
	points := func(r *store.StudentRank) int64 { return r.Points }
	return map[string][]topStudent{
		"courses":    board(func(r *store.StudentRank) int64 { return r.Courses }, active),
		"rank":       board(points, func(r *store.StudentRank) int64 { return r.Courses }),
		"creative":   board(func(r *store.StudentRank) int64 { return r.Creative }, points),
		"discipline": board(func(r *store.StudentRank) int64 { return r.CleanMin }, active),
		"talks":      board(func(r *store.StudentRank) int64 { return r.Talks }, func(r *store.StudentRank) int64 { return r.Creative }),
	}, nil
}

// handleTopStudents: GET /api/top-students
func (s *Server) handleTopStudents(w http.ResponseWriter, r *http.Request) {
	s.topStu.mu.Lock()
	defer s.topStu.mu.Unlock()
	if s.topStu.body == nil || time.Since(s.topStu.at) > topTTL {
		boards, err := s.topStudents(r.Context())
		if err != nil {
			if s.topStu.body == nil {
				s.storeErr(w, r, err)
				return
			}
			s.log.Warn("идэвхтэй суралцагчид", "err", err)
		} else {
			b, _ := json.Marshal(map[string]any{"boards": boards, "updated_at": time.Now()})
			s.topStu.body, s.topStu.at = b, time.Now()
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache") // өөрийгөө нуусны дараа шууд алга болно
	_, _ = w.Write(s.topStu.body)
}

// handleRankHidden: PUT /api/me/rank-hidden {hidden} — «Идэвхтэй суралцагчид» жагсаалтаас өөрийгөө нуух/харуулах.
func (s *Server) handleRankHidden(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Hidden bool `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	u, err := s.store.UpdateUser(r.Context(), c.UID, func(u *store.User) error {
		u.RankHidden = in.Hidden
		return nil
	})
	if s.storeErr(w, r, err) {
		return
	}
	s.topStu.mu.Lock()
	s.topStu.body = nil // дахин тооцоолно
	s.topStu.mu.Unlock()
	writeJSON(w, http.StatusOK, s.me(u))
}
