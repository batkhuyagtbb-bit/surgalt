package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// /intro2 — танилцуулгын кино интро бүтэн HTML хуудсаар (template-гүй) үйлчлэгдэнэ.
func TestIntroPage(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/intro2")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	body := string(b)
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("intro: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	for _, want := range []string{"<!doctype html>", "Surgalt.mn Intro 2", "gsap.min.js", "Өнөөдрийн ойлголт", "${"} {
		if !strings.Contains(body, want) {
			t.Fatalf("intro хуудсанд %q алга", want)
		}
	}
}
