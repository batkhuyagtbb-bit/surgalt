package main

import (
	"context"
	"errors"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

// seedDemo нь "demo" нэртэй багш, сургалтуудыг үүсгэнэ (байвал алгасна).
// Нэвтрэх: багш demo@surgalt.mn / demo12345, суралцагч suragch@surgalt.mn / demo12345
func seedDemo(ctx context.Context, st store.Store, fs *files.Store) error {
	if _, err := st.UserByUsername(ctx, "demo"); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(ctx, "demo12345")
	if err != nil {
		return err
	}
	u := &store.User{Username: "demo", Email: "demo@surgalt.mn", PasswordHash: hash, Role: store.RoleTeacher,
		DisplayName: "Сарангэрэл Батболд", Headline: "Математикийн багш · 12 жилийн туршлага · ЭЕШ бэлтгэл",
		Bio:      "Сайн байна уу! Би МУИС-ийн Математикийн ангийг төгсөж, 12 жил ахлах ангийн сурагчдад хичээл заасан.\n\n800+ сурагчийг ЭЕШ-д бэлдсэн бөгөөд тэдний дундаж оноо 680+ байдаг. Хичээл бүр богино, ойлгомжтой, дасгалтай.",
		Subjects: []string{"Математик", "ЭЕШ бэлтгэл", "Геометр", "Логик"}, Location: "Улаанбаатар",
		Links: map[string]string{"youtube": "https://www.youtube.com/@surgalt", "facebook": "https://www.facebook.com/surgalt"}}
	if err := st.CreateUser(ctx, u); err != nil {
		return err
	}
	_ = fs.EnsureTeacherDir(u.ID)
	courses := []struct {
		c       store.Course
		lessons []store.Lesson
	}{
		// Дарааллаар нээгдэх сургалт: өмнөхөө үзсэний дараа дараагийнх нь хугацаатай нээгдэнэ; багцаар авсан бол бүгд шууд.
		{store.Course{Title: "ЭЕШ Математик — Бүрэн бэлтгэл", Description: "Алгебр, геометр, магадлал — ЭЕШ-ийн бүх сэдвийг 40 хичээлээр. Тест бодох аргачлал, цагаа зөв хуваарилах арга.", Price: 89000, Published: true, Drip: true, UnlockAllPaid: true},
			[]store.Lesson{
				{Title: "Танилцуулга: ЭЕШ-ийн бүтэц ба стратеги", Content: "ЭЕШ-ийн шалгалт хэрхэн бүтэцлэгддэг, аль хэсэгт хэдэн минут зарцуулах вэ?", VideoURL: "https://www.youtube.com/watch?v=aqz-KE-bpKQ", IsFree: true, Format: "lecture", Mode: "online", Section: "1-р бүлэг. Эхлэл"},
				{Title: "Квадрат тэгшитгэл — 3 арга", Content: "Дискриминант, Виетийн теорем, бүтэн квадрат ялгах.", IsFree: true, Format: "seminar", Mode: "online", Section: "1-р бүлэг. Эхлэл"},
				{Title: "Логарифм ба илтгэгч функц", Content: "Бүрэн хичээл.", Price: 15000, UnlockAfterH: 24, Format: "lecture", Mode: "blended", Section: "2-р бүлэг. Алгебр"},
				{Title: "Тригонометр: томьёонуудыг цээжлэх нууц", Content: "Бүрэн хичээл.", Price: 15000, UnlockAfterH: 168, Format: "practice", Mode: "classroom", Section: "3-р бүлэг. Геометр, тригонометр"},
				{Title: "Магадлал ба комбинаторик", Content: "Бүрэн хичээл.", Price: 25000, AlwaysOpen: true, Format: "lab", Mode: "online", Section: "4-р бүлэг. Магадлал"},
			}},
		// Багц үнэгүй — хичээл бүрийг дангаар нь зарна.
		{store.Course{Title: "Геометрийн үндэс", Description: "Планиметр, стереометрийн үндсэн теоремууд зурагтай, алхам алхмаар. Хичээл бүрийг тусад нь авах боломжтой.", Price: 0, Published: true},
			[]store.Lesson{
				{Title: "Гурвалжны гайхамшигт шугамууд", Content: "Медиан, биссектрис, өндөр — хэзээ юуг ашиглах вэ?", IsFree: true, Section: "Планиметр"},
				{Title: "Тойрог ба түүний өнцгүүд", Content: "Бүрэн хичээл.", Price: 9900, Section: "Планиметр"},
				{Title: "Огторгуйн биетүүдийн эзлэхүүн", Content: "Бүрэн хичээл.", Price: 12900, Section: "Стереометр"},
			}},
		{store.Course{Title: "Оюуны сорил — Үнэгүй мини курс", Description: "Логик сэтгэлгээ хөгжүүлэх 3 богино хичээл. Бүгд үнэгүй!", Price: 0, Published: true},
			[]store.Lesson{
				{Title: "Логик даалгавар #1", Content: "Эхний даалгавар…", IsFree: true},
				{Title: "Логик даалгавар #2", Content: "Хоёр дахь даалгавар…", IsFree: true},
				{Title: "Логик даалгавар #3", Content: "Гурав дахь даалгавар…", IsFree: true, ActiveMin: 2,
					Blocks: []store.Block{
						{ID: "lg3text", Type: "text", Text: "**Анхаар:** энэ хичээлийг дор хаяж *2 минут* идэвхтэй үзэж байж дуусгана.\n\n- Дарааллыг ажигла\n- Хариугаа шалга"},
						{ID: "lg3q1", Type: "quiz", Quiz: &store.Quiz{Kind: "text", Question: "2, 4, 8, 16, … дараагийн тоо?", Answers: []string{"32"}, Explain: "Тоо бүр өмнөхөөсөө 2 дахин их."}},
						{ID: "lg3q2", Type: "quiz", Quiz: &store.Quiz{Kind: "match", Question: "Дүрсийг өнцгийн тоотой нь харгалзуул", Left: []string{"Гурвалжин", "Дөрвөлжин", "Таван өнцөгт"}, Right: []string{"3", "4", "5"}}},
						{ID: "lg3q3", Type: "quiz", Quiz: &store.Quiz{Kind: "multi", Question: "Аль нь тэгш тоо вэ?", Options: []string{"2", "3", "8", "11"}, Correct: []int{0, 2}}},
					}},
				{Title: "Эцсийн шалгалт", Content: "Логикийн мини шалгалт.", IsFree: true,
					Exam: &store.Exam{TimeMin: 10, Attempts: 3, PassPct: 60, ShowAnswers: true},
					Blocks: []store.Block{
						{ID: "exintro", Type: "text", Text: "Амжилт хүсье! Шалгалтын үеэр өөр цонх руу шилжихгүй байгаарай."},
						{ID: "ex1q", Type: "quiz", Quiz: &store.Quiz{Kind: "single", Question: "1, 1, 2, 3, 5, 8, … дараагийнх?", Options: []string{"11", "12", "13"}, Correct: []int{2}, Points: 2}},
						{ID: "ex2q", Type: "quiz", Quiz: &store.Quiz{Kind: "text", Question: "Долоо хоногт хэдэн өдөр байдаг вэ? (тоогоор)", Answers: []string{"7", "долоо"}}},
						{ID: "ex3q", Type: "quiz", Quiz: &store.Quiz{Kind: "match", Question: "Улирлыг сартай нь харгалзуул", Left: []string{"Өвөл", "Зун"}, Right: []string{"1-р сар", "7-р сар"}}},
					}},
			}},
	}
	var created []store.Course
	var firstPaid *store.Lesson
	for _, cc := range courses {
		c := cc.c
		c.TeacherID = u.ID
		if err := st.CreateCourse(ctx, &c); err != nil {
			return err
		}
		for _, l := range cc.lessons {
			l.CourseID = c.ID
			if err := st.CreateLesson(ctx, &l); err != nil {
				return err
			}
			if firstPaid == nil && len(created) == 1 && l.Price > 0 {
				lc := l
				firstPaid = &lc
			}
		}
		created = append(created, c)
	}
	if err := seedBooks(ctx, st, fs, u.ID); err != nil {
		return err
	}
	return seedStudent(ctx, st, u, created, firstPaid, hash)
}

// seedStudent нь нүүр хуудасны самбарыг үзүүлэх демо суралцагч: нэг сургалтад элссэн,
// нэг хичээл дангаар авсан, багштай чатласан, маргааш шууд хичээлтэй.
func seedStudent(ctx context.Context, st store.Store, teacher *store.User, courses []store.Course, paid *store.Lesson, hash string) error {
	s := &store.User{Username: "suragch", Email: "suragch@surgalt.mn", PasswordHash: hash, Role: store.RoleStudent, DisplayName: "Тэмүүлэн Ганбат"}
	if err := st.CreateUser(ctx, s); err != nil {
		return err
	}
	free, lessonsOnly := courses[2], courses[1]
	if err := st.Enroll(ctx, s.ID, &free); err != nil {
		return err
	}
	if paid != nil {
		o, err := st.CreateOrGetPendingLessonOrder(ctx, s.ID, &lessonsOnly, paid)
		if err != nil {
			return err
		}
		if _, err := st.MarkOrderPaid(ctx, o.ID, o.Amount); err != nil {
			return err
		}
	}
	if _, err := st.CreateOrGetPendingOrder(ctx, s.ID, &courses[0]); err != nil {
		return err
	}
	conv, err := st.GetOrCreateConversation(ctx, teacher.ID, "u:"+s.ID, s.DisplayName, s.ID)
	if err != nil {
		return err
	}
	for _, m := range []store.Message{
		{ConversationID: conv.ID, Sender: store.SenderVisitor, Body: "Сайн байна уу багшаа, ЭЕШ-ийн багцад давтлага багтсан уу?"},
		{ConversationID: conv.ID, Sender: store.SenderTeacher, Body: "Сайн уу! Тийм ээ, долоо хоног бүр шууд давтлага хийдэг."},
	} {
		if err := st.AddMessage(ctx, &m); err != nil {
			return err
		}
	}
	// Демо холбоос: жинхэнэ орчинд Google Calendar-аас үүснэ.
	tomorrow := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
	for _, mt := range []store.Meeting{
		{TeacherID: teacher.ID, CourseID: free.ID, Title: "Логик даалгаврын шууд задаргаа", StartsAt: time.Now().Add(3 * time.Hour).Truncate(time.Hour), DurationMin: 60, MeetURL: "https://meet.google.com/new"},
		{TeacherID: teacher.ID, CourseID: courses[0].ID, Title: "ЭЕШ давтлага — Логарифм", StartsAt: tomorrow, DurationMin: 90, MeetURL: "https://meet.google.com/new"},
	} {
		if err := st.CreateMeeting(ctx, &mt); err != nil {
			return err
		}
	}
	return nil
}
