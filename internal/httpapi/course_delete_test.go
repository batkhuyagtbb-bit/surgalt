package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Сургалт устгах: зөвхөн эзэн; суралцагчтай бол татгалзана; устгасны дараа олдохгүй, жагсаалтаас алга.
func TestDeleteCourse(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "delteach", "teacher")
	ot, _ := register(t, srv, "otherteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Хоосон","price":0,"published":true}`)
	cid := c["id"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хичээл","is_free":true}`)
	_, c2 := call(t, srv, "POST", "/api/courses", tt, `{"title":"Суралцагчтай","price":0,"published":true}`)
	cid2 := c2["id"].(string)
	st, _ := register(t, srv, "delstud", "student")
	call(t, srv, "POST", "/api/courses/"+cid2+"/enroll", st, "")

	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cid, ot, ""); code != http.StatusNotFound && code != http.StatusForbidden {
		t.Fatalf("өөр багш устгаж болохгүй: %d", code)
	}
	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cid, st, ""); code != http.StatusForbidden {
		t.Fatalf("суралцагч устгаж болохгүй: %d", code)
	}
	code, out := call(t, srv, "DELETE", "/api/courses/"+cid2, tt, "")
	if code != http.StatusConflict || out["students"] != float64(1) {
		t.Fatalf("суралцагчтай сургалт устгагдах ёсгүй: %d %v", code, out)
	}
	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cid, tt, ""); code != 200 {
		t.Fatalf("устгах: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid, "", ""); code != http.StatusNotFound {
		t.Fatalf("устгасан сургалт олдох ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/c/"+cid, "", ""); code != http.StatusNotFound {
		t.Fatalf("устгасан сургалтын хуудас: %d", code)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/api/me/courses", nil)
	req.Header.Set("Authorization", "Bearer "+tt)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var courses []map[string]any
	_ = json.NewDecoder(res.Body).Decode(&courses)
	if len(courses) != 1 || courses[0]["id"] != cid2 {
		t.Fatalf("миний сургалтууд: %v", courses)
	}
}
