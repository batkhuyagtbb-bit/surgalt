package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// Өгөгдлөө татаж авах ба бүртгэлээ устгах: экспорт нь өөрийн мэдээллийг бүрэн агуулна; устгахад суралцахуй,
// хяналт, мэдэгдэл устаж, чатын мессеж «устгагдсан» болж, удахгүй болох захиалга чөлөөлөгдөн багшид мэдэгдэнэ,
// хуучин токен хүчингүй болж, имэйлээр дахин бүртгүүлж болно. Сургалттай багш шууд устгахгүй.
func TestExportAndDeleteAccount(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, ttID := register(t, srv, "teach", "teacher")
	s1, s1ID := register(t, srv, "stud1", "student")

	course := &store.Course{TeacherID: ttID, Title: "Алгебрын үндэс", Published: true}
	if err := st.CreateCourse(ctx, course); err != nil {
		t.Fatal(err)
	}
	if err := st.Enroll(ctx, s1ID, course); err != nil {
		t.Fatal(err)
	}
	if err := st.AddActivityEvents(ctx, []store.ActivityEvent{{UserID: s1ID, UserName: "stud1", TeacherID: ttID, CourseID: course.ID,
		Type: "tab_switch", Detail: "5 сек өөр цонхонд", At: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	_, d := call(t, srv, "POST", "/api/teachers/teach/chat", s1, "")
	conv := d["conversation"].(map[string]any)["id"].(string)
	_, mine := call(t, srv, "POST", "/api/chat/"+conv+"/messages", s1, `{"body":"Миний асуулт"}`)
	call(t, srv, "POST", "/api/chat/"+conv+"/messages", tt, `{"body":"Багшийн хариу"}`)
	day := time.Now().Add(72 * time.Hour).Truncate(time.Hour).UTC().Format(time.RFC3339)
	_, sl := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+day+`"],"duration_min":60,"mode":"online"}`)
	slotID := sl[0].(map[string]any)["id"].(string)
	if code, r := call(t, srv, "POST", "/api/slots/"+slotID+"/book", s1, `{"note":"ЭЕШ"}`); code != http.StatusOK {
		t.Fatalf("захиалга: %d %v", code, r)
	}

	// Экспорт
	if code, _ := call(t, srv, "GET", "/api/me/export", "", ""); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй экспорт: %d", code)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/api/me/export", nil)
	req.Header.Set("Authorization", "Bearer "+s1)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var ex struct {
		Profile    map[string]any   `json:"profile"`
		Enrolled   []map[string]any `json:"enrolled_courses"`
		Monitoring []map[string]any `json:"monitoring_events"`
		Bookings   []map[string]any `json:"bookings"`
		Chats      []struct {
			Messages []map[string]any `json:"messages"`
		} `json:"chats"`
	}
	_ = json.NewDecoder(res.Body).Decode(&ex)
	res.Body.Close()
	if cd := res.Header.Get("Content-Disposition"); res.StatusCode != http.StatusOK || !strings.Contains(cd, "attachment") || !strings.Contains(cd, ".json") {
		t.Fatalf("экспорт файл: %d %q", res.StatusCode, cd)
	}
	if ex.Profile["email"] != "stud1@x.mn" || len(ex.Enrolled) != 1 || ex.Enrolled[0]["title"] != "Алгебрын үндэс" ||
		len(ex.Monitoring) != 1 || len(ex.Bookings) != 1 || len(ex.Chats) != 1 || len(ex.Chats[0].Messages) != 2 {
		t.Fatalf("экспортын агуулга: %+v", ex)
	}

	// Устгах
	if code, _ := call(t, srv, "POST", "/api/me/delete", s1, `{"confirm":"тийм"}`); code != http.StatusBadRequest {
		t.Fatalf("баталгаажуулалтгүй: %d", code)
	}
	if code, r := call(t, srv, "POST", "/api/me/delete", tt, `{"confirm":"УСТГАХ"}`); code != http.StatusConflict || !strings.Contains(r["error"].(string), "Сургалт") {
		t.Fatalf("сургалттай багш шууд устгахгүй: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/me/delete", s1, `{"confirm":"устгах"}`); code != http.StatusOK {
		t.Fatalf("устгах: %d %v", code, r)
	}
	if code, _ := call(t, srv, "GET", "/api/me", s1, ""); code != http.StatusUnauthorized {
		t.Fatalf("хуучин токен хүчингүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/login", "", `{"email":"stud1@x.mn","password":"password1"}`); code == http.StatusOK {
		t.Fatal("устгасан бүртгэлээр нэвтрэхгүй")
	}
	if _, err := st.UserByID(ctx, s1ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("хэрэглэгч устах ёстой: %v", err)
	}
	if evs, _ := st.ActivityEvents(ctx, store.ActivityFilter{UserID: s1ID}, 10); len(evs) != 0 {
		t.Fatalf("хяналтын бүртгэл устах ёстой: %v", evs)
	}
	if ok, _ := st.IsEnrolled(ctx, s1ID, course.ID); ok {
		t.Fatal("элсэлт устах ёстой")
	}
	_, l := call(t, srv, "GET", "/api/chat/"+conv+"/messages", tt, "")
	for _, m := range l["messages"].([]any) {
		mm := m.(map[string]any)
		if mm["id"] == mine["id"] && (mm["deleted"] != true || mm["body"] != "" || mm["sender_name"] == "stud1") {
			t.Fatalf("суралцагчийн мессеж устгагдсан, нэргүй болох ёстой: %v", mm)
		}
		if mm["body"] == "Багшийн хариу" && mm["deleted"] == true {
			t.Fatal("багшийн мессеж хэвээр үлдэнэ")
		}
	}
	if _, bs := callArrMap(t, srv, "GET", "/api/me/bookings", tt); len(bs) != 0 {
		t.Fatalf("захиалга чөлөөлөгдөх ёстой: %v", bs)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", tt, "")
	if ts := strings.Join(notifTitles(ns, NotifBooking), "|"); !strings.Contains(ts, "бүртгэлээ устгасан") {
		t.Fatalf("багшид мэдэгдэл: %v", ts)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/register", "", `{"email":"stud1@x.mn","password":"password2","accept_terms":true}`); code != http.StatusCreated {
		t.Fatalf("имэйлээр дахин бүртгүүлж болно: %d", code)
	}

	// Сургалтгүй багш: сул цагтайгаа устгагдана.
	t2, _ := register(t, srv, "teach2", "teacher")
	callArr(t, srv, "POST", "/api/me/slots", t2, `{"starts":["`+day+`"],"duration_min":60,"mode":"online"}`)
	if code, r := call(t, srv, "POST", "/api/me/delete", t2, `{"confirm":"УСТГАХ"}`); code != http.StatusOK {
		t.Fatalf("сургалтгүй багш устгах: %d %v", code, r)
	}
	if code, _ := call(t, srv, "GET", "/api/teachers/teach2", "", ""); code != http.StatusNotFound {
		t.Fatalf("устгасан багшийн профайл: %d", code)
	}
}
