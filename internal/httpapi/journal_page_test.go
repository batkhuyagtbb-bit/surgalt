package httpapi

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// syntheticJournal — n суралцагчтай нэгтгэл (бүгд элссэн, хүн бүрт нэг нүд).
func syntheticJournal(n int, built time.Time) *journalAgg {
	jr := &journalAgg{cells: map[string]map[string]*JournalCell{}, built: built, blank: RankInfo{Name: "Шинэ"}}
	for i := range n {
		uid := fmt.Sprintf("u%04d", i)
		jr.students = append(jr.students, JournalStudent{UserID: uid, Name: fmt.Sprintf("Суралцагч %04d", i), Username: fmt.Sprintf("user%04d", i),
			Enrolled: true, Done: 2, Active: 600, Rank: RankInfo{Points: 40, Name: "Ахисан"}})
		jr.cells[uid] = map[string]*JournalCell{"l1": {Points: 40, Done: true}}
	}
	for i := range jr.students {
		jr.lower = append(jr.lower, [2]string{strings.ToLower(jr.students[i].Name), strings.ToLower(jr.students[i].Username)})
	}
	return jr
}

// 1 000 суралцагч: хуудас ≤300, тоо бүх суралцагчаар, шүүлтүүд AND, кэшлэсэн нэгтгэл хөндөгдөхгүй.
func TestJournalPageCountsBeyondCap(t *testing.T) {
	jr := syntheticJournal(1000, time.Now())
	before := make([]JournalStudent, len(jr.students))
	copy(before, jr.students)
	at := time.Now().UTC()
	metas := map[string]store.CourseStudentMeta{}
	for i := range 1000 {
		uid := fmt.Sprintf("u%04d", i)
		switch {
		case i%10 == 0: // 100 архив (бүлэгтэй ч бүлгийн тоонд орохгүй)
			metas[uid] = store.CourseStudentMeta{UserID: uid, Group: "5А", Archived: true, ArchivedAt: &at}
		case i%3 == 0: // 300 — 5А
			metas[uid] = store.CourseStudentMeta{UserID: uid, Group: "5А"}
		case i%3 == 1: // 300 — 5б
			metas[uid] = store.CourseStudentMeta{UserID: uid, Group: "5б"}
		}
	}
	metas["u0998"] = store.CourseStudentMeta{UserID: "u0998", Alias: "Тусгай Нэр"} // хуудасны гадна, бүлэггүй

	v := journalPage(jr, metas, journalFilter{}, journalStudents)
	if len(v.students) != journalStudents || v.total != 900 {
		t.Fatalf("хуудас %d, нийт %d (300, 900 байх ёстой)", len(v.students), v.total)
	}
	g5a, g5b := 0, 0
	for i := range 1000 {
		if i%10 != 0 && i%3 == 0 {
			g5a++
		} else if i%10 != 0 && i%3 == 1 {
			g5b++
		}
	}
	want := []JournalGroup{{Name: "5А", Count: g5a}, {Name: "5б", Count: g5b}}
	if !reflect.DeepEqual(v.groups, want) || v.archived != 100 || v.ungrouped != 900-g5a-g5b {
		t.Fatalf("тоо: бүлэг %+v (%+v), бүлэггүй %d, архив %d", v.groups, want, v.ungrouped, v.archived)
	}
	if len(v.cells) != journalStudents {
		t.Fatalf("нүд %d", len(v.cells))
	}
	for _, s := range v.students {
		if s.Archived || s.AccountName == "" || v.cells[s.UserID] == nil {
			t.Fatalf("хуудасны мөр: %+v", s)
		}
	}

	// Бүлгээр (том жижиг үсэг үл харгалзан), бүлэггүй, архив, хайлт — нийт тоо нь хуудаснаас үл хамаарна; бүлгийн тоо өөрчлөгдөхгүй.
	if v := journalPage(jr, metas, journalFilter{group: "5а"}, journalStudents); v.total != g5a || len(v.students) != min(g5a, journalStudents) || v.archived != 100 || len(v.groups) != 2 {
		t.Fatalf("бүлгээр: нийт %d, хуудас %d, архив %d", v.total, len(v.students), v.archived)
	}
	if v := journalPage(jr, metas, journalFilter{noGroup: true}, journalStudents); v.total != 900-g5a-g5b {
		t.Fatalf("бүлэггүй: %d", v.total)
	}
	va := journalPage(jr, metas, journalFilter{archived: true}, journalStudents)
	if va.total != 100 || len(va.students) != 100 || !va.students[0].Archived || va.students[0].ArchivedAt == nil || va.students[0].Group != "5А" {
		t.Fatalf("архив: нийт %d, мөр %+v", va.total, va.students[0])
	}
	if v := journalPage(jr, metas, journalFilter{archived: true, group: "5б"}, journalStudents); v.total != 0 {
		t.Fatalf("архив + бүлэг: %d", v.total)
	}
	vq := journalPage(jr, metas, journalFilter{q: "тусгай"}, journalStudents)
	if vq.total != 1 || vq.students[0].UserID != "u0998" || vq.students[0].Name != "Тусгай Нэр" || vq.students[0].AccountName != "Суралцагч 0998" || vq.students[0].Alias != "Тусгай Нэр" {
		t.Fatalf("нэрээр хайх: %+v", vq.students)
	}
	if v := journalPage(jr, metas, journalFilter{q: "user0998"}, journalStudents); v.total != 1 {
		t.Fatalf("username-аар хайх: %d", v.total)
	}
	if v := journalPage(jr, metas, journalFilter{q: "суралцагч 0998"}, journalStudents); v.total != 1 {
		t.Fatalf("бүртгэлийн нэрээр хайх: %d", v.total)
	}
	// Зөвхөн тоо (limit 0).
	if v := journalPage(jr, metas, journalFilter{}, 0); len(v.students) != 0 || v.total != 900 || v.archived != 100 {
		t.Fatalf("тоо л: %d %d %d", len(v.students), v.total, v.archived)
	}
	// Мэдээлэлгүй үед бүлэг хоосон массив (null биш).
	if v := journalPage(jr, nil, journalFilter{}, 10); v.groups == nil || len(v.groups) != 0 || v.ungrouped != 1000 {
		t.Fatalf("мэдээлэлгүй: %+v %d", v.groups, v.ungrouped)
	}

	if !reflect.DeepEqual(before, jr.students) {
		t.Fatal("кэшлэсэн нэгтгэл өөрчлөгдлөө")
	}
}

// Нэгтгэлээс хойш өгөгдөл нь устсан суралцагч: элссэн бол хоосон мөр (нүдгүй), элсээгүй бол бүр хасагдана.
func TestJournalPageBlanksStaleReset(t *testing.T) {
	built := time.Now().Add(-10 * time.Second)
	jr := syntheticJournal(3, built)
	jr.students[1].Enrolled = false // зочин: зөвхөн үзсэн
	after, before := built.Add(5*time.Second), built.Add(-time.Minute)
	metas := map[string]store.CourseStudentMeta{
		"u0000": {UserID: "u0000", ResetAt: &after},  // элссэн, нэгтгэлээс хойш устгасан
		"u0001": {UserID: "u0001", ResetAt: &after},  // элсээгүй — алга болно
		"u0002": {UserID: "u0002", ResetAt: &before}, // нэгтгэлээс өмнө — нэгтгэл аль хэдийн шинэ
	}
	v := journalPage(jr, metas, journalFilter{}, journalStudents)
	if v.total != 2 || len(v.students) != 2 || v.ungrouped != 2 {
		t.Fatalf("нийт %d, мөр %d, бүлэггүй %d", v.total, len(v.students), v.ungrouped)
	}
	blank, fresh := v.students[0], v.students[1]
	if blank.UserID != "u0000" || blank.Done != 0 || blank.Active != 0 || blank.Rank.Name != "Шинэ" || blank.Rank.Points != 0 ||
		!blank.Enrolled || blank.Name != "Суралцагч 0000" || blank.AccountName != "Суралцагч 0000" || blank.Username != "user0000" {
		t.Fatalf("хоосон мөр: %+v", blank)
	}
	if _, ok := v.cells["u0000"]; ok {
		t.Fatal("устгасан суралцагчийн нүд харагдлаа")
	}
	if fresh.UserID != "u0002" || fresh.Done != 2 || v.cells["u0002"] == nil {
		t.Fatalf("шинэ нэгтгэлийн мөр: %+v", fresh)
	}
	if v := journalPage(jr, metas, journalFilter{q: "user0001"}, journalStudents); v.total != 0 {
		t.Fatal("элсээгүй, устгасан суралцагч хайлтаар олдлоо")
	}
	if jr.students[0].Done != 2 || jr.cells["u0000"] == nil {
		t.Fatal("кэшлэсэн нэгтгэл өөрчлөгдлөө")
	}
}

func TestJournalTextAndGroupName(t *testing.T) {
	m := journalMsgs{"bad", "long", "letter"}
	// u — тэмдэгтийн кодуудаас мөр (үл харагдах тэмдэгтийг эх кодонд шууд бичихгүйн тулд).
	u := func(parts ...any) string {
		var b strings.Builder
		for _, p := range parts {
			switch v := p.(type) {
			case string:
				b.WriteString(v)
			case int:
				b.WriteRune(rune(v))
			}
		}
		return b.String()
	}
	mn := u(0x182E, 0x1823, 0x1829, 0x182D, 0x1823, 0x182F) // монгол бичгийн үсгүүд
	for in, want := range map[string]string{
		"  Бат   Дорж ": "Бат Дорж", "": "", "   ": "", "5": "5",
		u("е", 0x0308):            "ё",                       // NFC: задарсан «ё» нэг тэмдэгт болно
		u(mn, 0x180E, 0x1820):     u(mn, 0x180E, 0x1820),     // монгол бичиг: эгшиг тусгаарлагч
		u(0x182A, 0x200D, 0x1832): u(0x182A, 0x200D, 0x1832), // ZWJ
		u(0x182A, 0x200C, 0x1832): u(0x182A, 0x200C, 0x1832), // ZWNJ
		u(0x1820, 0x180B):         u(0x1820, 0x180B),         // хувилбар сонгогч
	} {
		if got, msg := journalText(in, 80, m); got != want || msg != "" {
			t.Fatalf("%q → %q %q", in, got, msg)
		}
	}
	for _, in := range []string{"Бат\x00", "Бат\nДорж", "Бат\t", u(0x202E, "абв"), u("а", 0x200B, "б"), u("Ба", 0x200D, "т"),
		u("Ба", 0x200C, "т"), u("Бат", 0x180E), u(0x3164, "Бат"), u("Бат", 0x115F), u(0x1160), u("Бат", 0xFFA0), u(0x2800, "Бат"),
		u("Ба", 0x034F, "т"), u("Бат", 0xFE0F), u("Бат", 0xE0100), u(0xFEFF, "Бат"), u(0x200D)} {
		if _, msg := journalText(in, 80, m); msg != m.bad {
			t.Fatalf("%q зөвшөөрөгдлөө: %q", in, msg)
		}
	}
	for _, in := range []string{"...", "😀", "- -", "★"} {
		if _, msg := journalText(in, 80, m); msg != m.letter {
			t.Fatalf("%q үсэг/тоогүй ч зөвшөөрөгдлөө: %q", in, msg)
		}
	}
	if _, msg := journalText(strings.Repeat("ё", 81), 80, m); msg != m.long {
		t.Fatal("урт нэр")
	}
	if _, msg := journalText(" "+strings.Repeat(u("е", 0x0308), 80)+"  ", 80, m); msg != "" {
		t.Fatal("80 тэмдэгт (NFC, зайгүй) зөвшөөрөгдөх ёстой")
	}

	all := map[string]store.CourseStudentMeta{"a": {Group: "5А"}, "me": {Group: "Миний"}}
	if g, err := journalGroupName("5а", "me", all); err != nil || g != "5А" {
		t.Fatalf("байгаа бүлгийн бичлэг: %q %v", g, err)
	}
	if g, err := journalGroupName("мИНИЙ", "me", all); err != nil || g != "мИНИЙ" {
		t.Fatalf("өөрийн бүлгийн үсэг солих: %q %v", g, err)
	}
	for i := range journalGroupsMax {
		all[fmt.Sprintf("u%d", i)] = store.CourseStudentMeta{Group: fmt.Sprintf("Г%d", i)}
	}
	if _, err := journalGroupName("Шинэ", "me", all); err == nil || err.(*APIError).Code != 400 {
		t.Fatalf("101 дэх бүлэг: %v", err)
	}
	if g, err := journalGroupName("г5", "me", all); err != nil || g != "Г5" {
		t.Fatalf("байгаа бүлэгт нэгдэх: %q %v", g, err)
	}
}
