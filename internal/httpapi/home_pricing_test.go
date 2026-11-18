package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Нүүрний «Үнэ» хэсэг: багш, байгууллагын багц store-оос; хэмнэлт, «Хамгийн ашигтай» (байгууллагын 6, 12 сар тэнцүү тул
// тэмдэглэхгүй).
func TestHomePricing(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(b)
	for _, want := range []string{`id="pricing"`, "Борлуулалтаас хувь авахгүй", "29,900₮", "99,000₮", "159,000₮", "56% хэмнэлт",
		"129,000₮", "500,000₮", "1,000,000₮", "Гүйлгээний шимтгэл <b>2%</b>", "гүйлгээний шимтгэл <b>1%</b>", "100 ба түүнээс доош суралцагчтай (хүүхэдтэй) байгууллагад хамаарна", "100-аас олон суралцагчтай", `href="tel:+97699858233"`, `viber://chat?number=%2B97699858233`, `href="/#pricing"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("нүүрэнд %q алга", want)
		}
	}
	if n := strings.Count(html, "Хамгийн ашигтай"); n != 1 {
		t.Fatalf("«Хамгийн ашигтай» зөвхөн багшийн жилийн багцад: %d", n)
	}
}
