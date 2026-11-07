package httpapi

import (
	"strings"
	"testing"
)

// Нэвтрэх хуудас Google/Facebook товчийг серверт урьдчилан зурна (JS хожуу нэмж маягтыг доош түлхэхгүй).
func TestLoginSocialPrerender(t *testing.T) {
	var b strings.Builder
	ps := []map[string]string{{"name": "google", "url": "/auth/google/start", "short": "Google"}, {"name": "facebook", "url": "/auth/facebook/start", "short": "Facebook"}}
	if err := parseTemplates().ExecuteTemplate(&b, "login.html", map[string]any{"Providers": ps}); err != nil {
		t.Fatal(err)
	}
	h := b.String()
	if strings.Count(h, `class="btn btn-social"`) != 2 || !strings.Contains(h, `href="/auth/google/start"`) || !strings.Contains(h, "Google-ээр үргэлжлүүлэх") || strings.Contains(h, `class="divider" hidden`) {
		t.Fatalf("товчнууд урьдчилан зурагдаагүй:\n%s", h[strings.Index(h, `id="social"`)-20:][:600])
	}
	b.Reset()
	if err := parseTemplates().ExecuteTemplate(&b, "login.html", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `class="divider" hidden`) || strings.Contains(b.String(), "btn-social\"") {
		t.Fatal("үйлчилгээгүй үед «эсвэл имэйлээр» нуугдах ёстой")
	}
}
