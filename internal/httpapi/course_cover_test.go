package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/http"
	"strings"
	"testing"
)

// Сургалтын толгойн зургийн загвар (t01–t20): үүсгэх/засахад хадгалагдана, буруу ID татгалзагдана, cover-гүй
// хэсэгчилсэн хадгалалт зургийг арилгахгүй, нийтийн хуудсанд SVG нь орно, SVG файлууд үйлчилнэ.
func TestCourseCover(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "cov", "teacher")
	code, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Физик","price":0,"published":true,"cover":"t04"}`)
	if code != 201 || c["cover"] != "t04" {
		t.Fatalf("зурагтай үүсгэх: %d %v", code, c)
	}
	cid := c["id"].(string)
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик","price":0,"published":true,"cover":"t999"}`); code != 400 {
		t.Fatalf("буруу загвар: %d", code)
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true}`); u["cover"] != "t04" {
		t.Fatalf("cover-гүй хадгалалт зургийг арилгахгүй: %v", u)
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true,"cover":"t11"}`); u["cover"] != "t11" {
		t.Fatalf("солих: %v", u)
	}
	res, err := http.Get(srv.URL + "/c/" + cid)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "/static/covers/t11.svg") {
		t.Fatalf("нийтийн хуудас: %d, зураг олдсонгүй", res.StatusCode)
	}
	for _, id := range []string{"t01", "t11", "t20"} {
		r, err := http.Get(srv.URL + "/static/covers/" + id + ".svg")
		if err != nil || r.StatusCode != 200 || !strings.Contains(r.Header.Get("Content-Type"), "svg") {
			t.Fatalf("%s.svg: %v %d %s", id, err, r.StatusCode, r.Header.Get("Content-Type"))
		}
		r.Body.Close()
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true,"cover":""}`); u["cover"] != "" {
		t.Fatalf("зураггүй болгох: %v", u)
	}
}

// Сургалтын товч тайлбар 50 үгэнд багтана.
func TestCourseDescriptionWords(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "desc", "teacher")
	long := strings.Repeat("үг ", 51)
	if code, r := call(t, srv, "POST", "/api/courses", tt, `{"title":"Урт","price":0,"description":"`+long+`"}`); code != 400 || !strings.Contains(r["error"].(string), "50 үг") {
		t.Fatalf("51 үг: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/courses", tt, `{"title":"Яг","price":0,"description":"`+strings.Repeat("үг ", 50)+`"}`); code != 201 {
		t.Fatalf("50 үг: %d", code)
	}
}

// Толгойн зургийн загвар профайлын нүүр зураг ч болно; буруу загвар татгалзагдана.
func TestProfileCoverTemplate(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "pcov", "teacher")
	code, me := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/covers/t13.svg"}`)
	if code != 200 || me["cover_url"] != "/static/covers/t13.svg" {
		t.Fatalf("загвар нүүр зураг: %d %v", code, me)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/covers/t777.svg"}`); code != 400 {
		t.Fatalf("буруу загвар: %d", code)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/app.css"}`); code != 400 {
		t.Fatalf("өөр static файл: %d", code)
	}
}

// Толгойн зургийн эх сурвалжууд: өөрийн public файл, https холбоос, embed зөвшөөрөгдөнө; бусдын файл, private файл,
// http холбоос, хэт урт embed татгалзагдана; нийтийн хуудсанд төрлөөрөө орно.
func TestCourseCoverSources(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "cvs", "teacher")
	_, other := register(t, srv, "cvo", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Эх сурвалж","price":0,"published":true}`)
	cid := c["id"].(string)
	put := func(cover string) (int, map[string]any) {
		return call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Эх сурвалж","price":0,"published":true,"cover":"`+cover+`"}`)
	}
	pub := uploadPublic(t, srv, tt, "cover.png")
	if code, u := put(pub); code != 200 || u["cover"] != pub {
		t.Fatalf("өөрийн public файл: %d %v", code, u)
	}
	priv := uploadFile(t, srv, tt, "secret.pdf", []byte("%PDF-1.4 x"))
	if code, _ := put(priv); code != 400 {
		t.Fatalf("private файл толгой болохгүй: %d", code)
	}
	if code, _ := put("/files/" + other + "/public/x.webp"); code != 400 {
		t.Fatalf("бусдын файл: %d", code)
	}
	if code, u := put("https://example.com/cover.jpg"); code != 200 || u["cover"] != "https://example.com/cover.jpg" {
		t.Fatalf("https холбоос: %d %v", code, u)
	}
	if code, _ := put("ftp://example.com/cover.jpg"); code != 400 {
		t.Fatalf("http(s) биш холбоос: %d", code)
	}
	if code, u := put("embed:https://www.youtube.com/watch?v=abc123"); code != 200 || u["cover"] != "embed:https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("embed: %d %v", code, u)
	}
	if code, _ := put("embed:" + strings.Repeat("x", 20001)); code != 400 {
		t.Fatalf("хэт урт embed: %d", code)
	}
	res, err := http.Get(srv.URL + "/c/" + cid)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), `data-cv-embed="https://www.youtube.com/watch?v=abc123"`) {
		t.Fatal("нийтийн хуудсанд embed толгой алга")
	}
}

// uploadPublic — нээлттэй (public) файлын санд жижиг PNG оруулаад замыг нь өгнө.
func uploadPublic(t *testing.T, srv *httptest.Server, token, name string) string {
	t.Helper()
	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", name)
	fw.Write(pngBuf.Bytes())
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/me/files?visibility=public", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode != 201 && res.StatusCode != 200 {
		t.Fatalf("public файл хуулагдсангүй: %d %v", res.StatusCode, out)
	}
	return out["path"].(string)
}
