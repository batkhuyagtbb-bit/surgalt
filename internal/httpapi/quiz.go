package httpapi

import (
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// QuizAnswer — суралцагчийн хариулт (асуултын төрлөөс хамаарч нэг талбар).
type QuizAnswer struct {
	Answer []int     `json:"answer,omitempty"` // single / multi: сонгосон хувилбарууд
	Text   string    `json:"text,omitempty"`   // text: бичсэн хариулт
	Match  []int     `json:"match,omitempty"`  // match: зүүн мөр бүрт сонгосон баруун (харагдаж буй дарааллын) индекс
	Point  []float64 `json:"point,omitempty"`  // image: дарсан цэг [x, y] хувиар
}

var quizKinds = map[string]string{"single": "нэг сонголттой", "multi": "олон сонголттой", "text": "бичгээр хариулах", "match": "харгалзуулах", "image": "зурган дээр заах"}

// validateQuiz нь асуултыг төрлөөр нь шалгаж цэвэрлэнэ.
func validateQuiz(q *store.Quiz, teacherID string) string {
	q.Kind = q.QuizKind()
	if quizKinds[q.Kind] == "" {
		return "асуултын төрөл буруу"
	}
	q.Question, q.Explain, q.Image = strings.TrimSpace(q.Question), strings.TrimSpace(q.Explain), stripSig(strings.TrimSpace(q.Image))
	name := "«" + q.Question + "» асуулт"
	switch {
	case q.Question == "" || utf8.RuneCountInString(q.Question) > 1000:
		return "асуулт 1-1000 тэмдэгт"
	case utf8.RuneCountInString(q.Explain) > 1000:
		return "тайлбар 1000 тэмдэгтээс хэтрэхгүй"
	case q.Points < 0 || q.Points > 100:
		return "оноо 0-100"
	case q.Image != "" && (len(q.Image) > maxBlockURL || (!validURL(q.Image) && !files.OwnedBy(q.Image, teacherID))):
		return name + ": зураг өөрийн файлын сангаас эсвэл http(s) холбоос"
	}
	trimAll := func(xs []string, max int) bool {
		for i := range xs {
			xs[i] = strings.TrimSpace(xs[i])
			if xs[i] == "" || utf8.RuneCountInString(xs[i]) > max {
				return false
			}
		}
		return true
	}
	switch q.Kind {
	case "single", "multi":
		q.Multi = q.Kind == "multi"
		q.Answers, q.Left, q.Right, q.Spot = nil, nil, nil, nil
		if len(q.Options) < 2 || len(q.Options) > maxQuizOptions {
			return name + " 2-8 хариулттай байна"
		}
		if !trimAll(q.Options, 300) {
			return name + "-ын хариулт хоосон эсвэл хэт урт байна"
		}
		slices.Sort(q.Correct)
		q.Correct = slices.Compact(q.Correct)
		if len(q.Correct) == 0 {
			return name + "-ын зөв хариултыг сонгоно уу"
		}
		for _, c := range q.Correct {
			if c < 0 || c >= len(q.Options) {
				return "зөв хариултын дугаар буруу"
			}
		}
		if !q.Multi && len(q.Correct) != 1 {
			return name + ": нэг сонголттой асуултад зөвхөн нэг зөв хариулт"
		}
	case "text":
		q.Options, q.Correct, q.Left, q.Right, q.Spot, q.Multi = nil, nil, nil, nil, nil, false
		if len(q.Answers) < 1 || len(q.Answers) > 10 || !trimAll(q.Answers, 200) {
			return name + ": зөвд тооцох 1-10 хариулт бичнэ үү"
		}
	case "match":
		q.Options, q.Correct, q.Answers, q.Spot, q.Multi = nil, nil, nil, nil, false
		if len(q.Left) < 2 || len(q.Left) > 10 || len(q.Left) != len(q.Right) {
			return name + ": 2-10 хос бөглөнө үү"
		}
		if !trimAll(q.Left, 200) || !trimAll(q.Right, 200) {
			return name + ": хос бүрийн хоёр талыг бөглөнө үү"
		}
	case "image":
		q.Options, q.Correct, q.Answers, q.Left, q.Right, q.Multi = nil, nil, nil, nil, nil, false
		if q.Image == "" {
			return name + ": зураг оруулна уу"
		}
		if q.Spot == nil || q.Spot.X < 0 || q.Spot.X > 100 || q.Spot.Y < 0 || q.Spot.Y > 100 || q.Spot.R < 1 || q.Spot.R > 50 {
			return name + ": зураг дээр зөв хэсгийг дарж тэмдэглэнэ үү"
		}
	}
	return ""
}

// matchPerm — харгалзуулах асуултын баруун баганын холих дараалал. Блокийн ID-гаас тогтмол
// гардаг тул сервер хадгалахгүйгээр дахин гаргана. view[j] = Right[perm[j]].
func matchPerm(blockID string, n int) []int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(blockID))
	r := rand.New(rand.NewPCG(h.Sum64(), 0x5eed))
	p := r.Perm(n)
	if n > 1 && slices.Equal(p, identity(n)) { // хэзээ ч шууд таарахгүй
		p[0], p[1] = p[1], p[0]
	}
	return p
}

func identity(n int) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return p
}

// normText: том жижиг үсэг, илүүдэл зай, төгсгөлийн цэг таслалыг үл тооно.
func normText(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	return strings.TrimRightFunc(s, func(r rune) bool { return unicode.IsPunct(r) })
}

// gradeQuiz — хариулт зөв эсэх.
func gradeQuiz(blockID string, q *store.Quiz, a QuizAnswer) bool {
	switch q.QuizKind() {
	case "single", "multi":
		ans := slices.Clone(a.Answer)
		slices.Sort(ans)
		return slices.Equal(slices.Compact(ans), q.Correct)
	case "text":
		t := normText(a.Text)
		return t != "" && slices.ContainsFunc(q.Answers, func(x string) bool { return normText(x) == t })
	case "match":
		perm := matchPerm(blockID, len(q.Left))
		if len(a.Match) != len(q.Left) {
			return false
		}
		for i, j := range a.Match {
			if j < 0 || j >= len(perm) || perm[j] != i {
				return false
			}
		}
		return true
	case "image":
		if q.Spot == nil || len(a.Point) != 2 {
			return false
		}
		dx, dy := a.Point[0]-q.Spot.X, a.Point[1]-q.Spot.Y
		return dx*dx+dy*dy <= q.Spot.R*q.Spot.R
	}
	return false
}

// answered — хоосон хариулт эсэх.
func (a QuizAnswer) empty() bool {
	return len(a.Answer) == 0 && strings.TrimSpace(a.Text) == "" && len(a.Match) == 0 && len(a.Point) == 0
}

// revealQuiz — хариулсны дараа харуулах зөв хариулт.
func revealQuiz(blockID string, q *store.Quiz) map[string]any {
	out := map[string]any{"explain": q.Explain}
	switch q.QuizKind() {
	case "single", "multi":
		out["answer"] = q.Correct
	case "text":
		out["answers"] = q.Answers
	case "match":
		perm := matchPerm(blockID, len(q.Left))
		m := make([]int, len(q.Left))
		for j, i := range perm {
			m[i] = j
		}
		out["match"] = m
	case "image":
		out["spot"] = q.Spot
	}
	return out
}

// viewerQuiz — суралцагчид илгээх хуулбар: зөв хариултгүй, баруун багана холилдсон.
func (s *Server) viewerQuiz(blockID string, q *store.Quiz) *store.Quiz {
	v := store.Quiz{Kind: q.QuizKind(), Question: q.Question, Options: slices.Clone(q.Options), Multi: q.QuizKind() == "multi", Left: slices.Clone(q.Left), Points: q.Points}
	if q.Image != "" {
		v.Image = s.media(q.Image)
	}
	if len(q.Right) > 0 {
		perm := matchPerm(blockID, len(q.Right))
		v.Right = make([]string, len(perm))
		for j, i := range perm {
			v.Right[j] = q.Right[i]
		}
	}
	return &v
}

// quizPoints — шалгалтын оноо (0 бол 1).
func quizPoints(q *store.Quiz) float64 {
	if q.Points <= 0 {
		return 1
	}
	return float64(q.Points)
}
