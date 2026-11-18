package httpapi

import (
	"context"
	"testing"
)

// «👁 Суралцагчаар үзэх»: багш сургалтаа шинэ суралцагчийн нүдээр — хаалттай хичээл харагдахгүй, дарааллын түгжээ,
// төлбөр, нууц үг суралцагчийнх шиг; юу ч бичихгүй. Суралцагч ?view=student дамжуулсан ч өөрийн жинхэнэ төлвөө авна.
func TestStudentPreviewAccess(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "svteach", "teacher")
	s1, _ := register(t, srv, "svstud", "student")
	cid := jmCourse(t, srv, tt, `{"title":"Дараалал","price":0,"published":true,"drip":true}`)
	l1 := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	l2 := jmLesson(t, srv, tt, cid, `{"title":"Хоёр","is_free":true}`)
	l3 := jmLesson(t, srv, tt, cid, `{"title":"Гурав","is_free":true,"unlock_rule":"manual"}`)
	lh := jmLesson(t, srv, tt, cid, `{"title":"Нуусан","is_free":true}`)
	if code, r := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+lh+"/visibility", tt, `{"hidden":true}`); code != 200 {
		t.Fatalf("нуух: %d %v", code, r)
	}

	// Багшийн ердийн төлөв: бүгд нээлттэй, нуусан хичээл ч харагдана.
	_, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", tt, "")
	if a["preview"] == true || a["all"] != true || a["owner"] != true {
		t.Fatalf("багшийн ердийн төлөв: %v", a)
	}
	if st := a["states"].(map[string]any); st[lh] == nil || st[l2].(map[string]any)["open"] != true {
		t.Fatalf("багш бүгдийг нээлттэй, нуусныг ч харах ёстой: %v", st)
	}

	// Суралцагчаар үзэх: эхний хичээл нээлттэй, 2 нь «prev», 3 нь «manual», нуусан нь алга, эрх ч алга.
	code, p := call(t, srv, "GET", "/api/courses/"+cid+"/access?view=student", tt, "")
	if code != 200 || p["preview"] != true || p["owner"] != true || p["all"] != false || p["enrolled"] != false {
		t.Fatalf("урьдчилан харах: %d %v", code, p)
	}
	st := p["states"].(map[string]any)
	if st[lh] != nil {
		t.Errorf("нуусан хичээл суралцагчид харагдах ёсгүй: %v", st[lh])
	}
	if s := st[l1].(map[string]any); s["open"] != true {
		t.Errorf("эхний хичээл нээлттэй байх ёстой: %v", s)
	}
	if s := st[l2].(map[string]any); s["open"] == true || s["reason"] != "prev" || s["prev_title"] != "Нэг" {
		t.Errorf("2-р хичээл «prev» түгжээтэй байх ёстой: %v", s)
	}
	if s := st[l3].(map[string]any); s["open"] == true || s["reason"] != "manual" {
		t.Errorf("3-р хичээл «manual» байх ёстой: %v", s)
	}
	if p["total"].(float64) != 3 || p["done"].(float64) != 0 || len(p["lessons"].([]any)) != 0 {
		t.Errorf("тоо: total=%v done=%v lessons=%v", p["total"], p["done"], p["lessons"])
	}

	// Суралцагч ?view=student дамжуулсан ч өөрийн төлөв (урьдчилан харах биш).
	_, sp := call(t, srv, "GET", "/api/courses/"+cid+"/access?view=student", s1, "")
	if sp["preview"] == true || sp["owner"] == true {
		t.Fatalf("суралцагчид урьдчилан харах горим үйлчлэх ёсгүй: %v", sp)
	}

	// Нууц үгтэй сургалт: урьдчилан харахад need_code үнэн (багшид ердийн үед худал).
	lc := jmCourse(t, srv, tt, `{"title":"Нууц","price":0,"published":true,"access_code":"abcd1234"}`)
	jmLesson(t, srv, tt, lc, `{"title":"А","is_free":true}`)
	_, n := call(t, srv, "GET", "/api/courses/"+lc+"/access", tt, "")
	_, np := call(t, srv, "GET", "/api/courses/"+lc+"/access?view=student", tt, "")
	if n["need_code"] != false || np["need_code"] != true {
		t.Errorf("нууц үг: ердийн=%v урьдчилан=%v", n["need_code"], np["need_code"])
	}
}

// Багш, хамтран заагч багш өөрийн сургалтын асуултад хариулбал (жишээ нь «Суралцагчаар үзэх» горимд) явц, асуултын
// бүртгэл бичигдэхгүй; суралцагчийнх бичигдэнэ.
func TestQuizNoWriteForCoTeacher(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "qzowner", "teacher")
	co, coID := register(t, srv, "qzco", "teacher")
	s1, s1ID := register(t, srv, "qzstud", "student")
	cid := jmCourse(t, srv, tt, `{"title":"Асуулт","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":[{"id":"q001","type":"quiz","quiz":{"question":"2+2?","options":["3","4"],"correct":[1]}}]}`)
	ctx := context.Background()
	c, err := st.CourseByID(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCourseOrg(ctx, c.TeacherID, cid, "", []string{coID}); err != nil {
		t.Fatal(err)
	}
	for who, tok := range map[string]string{"эзэмшигч": tt, "хамтран заагч": co, "суралцагч": s1} {
		if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/q001", tok, `{"answer":[1]}`); code != 200 || r["correct"] != true {
			t.Fatalf("%s хариулт: %d %v", who, code, r)
		}
	}
	if p, _ := st.LessonProgress(ctx, coID, cid); len(p[lid].Quiz) != 0 {
		t.Errorf("хамтран заагчийн хариулт хадгалагдах ёсгүй: %v", p[lid])
	}
	if p, _ := st.LessonProgress(ctx, s1ID, cid); !p[lid].Quiz["q001"] {
		t.Errorf("суралцагчийн хариулт хадгалагдах ёстой: %v", p[lid])
	}
}
