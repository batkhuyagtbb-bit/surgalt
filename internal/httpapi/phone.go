package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/store"
)

// ---- Багшийн утас: профайл дээр эхний 4 орон, дарахад бүтэн дугаар (багшид мэдэгдэнэ), дахин дарахад залгана ----

const phoneRepeat = 6 * time.Hour // нэг хүн дахин нээвэл энэ хугацаанд дахин мэдэгдэж, тоолохгүй

// cleanPhone: зай, зураас, хаалт, цэгийг хасна. Монгол дугаар 8 оронтой (+976 угтвартай ирж болно) — 8 оронгоор
// хадгална; гадаад дугаар «+» ба 7–15 орон. Хоосон бол дугааргүй (устгана).
func cleanPhone(s string) (string, bool) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '(', ')', '.', ' ':
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	if s == "" {
		return "", true
	}
	plus, d := strings.HasPrefix(s, "+"), strings.TrimPrefix(s, "+")
	if d == "" || strings.Trim(d, "0123456789") != "" {
		return "", false
	}
	if len(d) == 11 && strings.HasPrefix(d, "976") {
		d, plus = d[3:], false
	}
	switch {
	case !plus && len(d) == 8:
		return d, true
	case plus && len(d) >= 7 && len(d) <= 15:
		return "+" + d, true
	}
	return "", false
}

// phoneParts: харуулах хэлбэр («9911 2233»), залгах tel: утга («+97699112233»), профайл дээр харагдах эхний 4 орон.
func phoneParts(p string) (display, tel, hint string) {
	switch {
	case p == "":
		return "", "", ""
	case strings.HasPrefix(p, "+"):
		return p, p, p[:5] // «+» ба эхний 4 орон
	}
	return p[:4] + " " + p[4:], "+976" + p, p[:4]
}

// firstPhoneView: энэ хүн тухайн багшийн дугаарыг phoneRepeat хугацаанд анх удаа нээж байгаа эсэх.
func (n *notifier) firstPhoneView(key string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if t, ok := n.phoneSeen[key]; ok && time.Since(t) < phoneRepeat {
		return false
	}
	if len(n.phoneSeen) < maxSeen {
		n.phoneSeen[key] = time.Now()
	}
	return true
}

// handleRevealPhone: POST /api/teachers/{username}/phone {vid} — нуусан дугаарыг бүтнээр нь өгнө. Нээлттэй хуудас
// (бүх зочинд нэг кэш) зөвхөн эхний 4 оронг агуулдаг; бүтэн дугаар энд л гарна: робот, хэт олон хүсэлтийг хасч,
// хэн харсныг багшид мэдэгдэн статистикт бүртгэнэ (нэг хүн 6 цагт нэг удаа). Багш өөрийнхийгөө нээвэл тоолохгүй.
func (s *Server) handleRevealPhone(w http.ResponseWriter, r *http.Request) {
	if botRe.MatchString(r.UserAgent()) || !s.phoneLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Түр хүлээгээд дахин оролдоно уу")
		return
	}
	var in struct {
		VID string `json:"vid"` // хөтчийн санамсаргүй ID (нэвтрээгүй зочныг давхар тоолохгүй)
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<10)).Decode(&in) // бие заавал биш
	t, err := s.store.UserByUsername(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	if t.Role != store.RoleTeacher || t.Phone == "" {
		writeErr(w, http.StatusNotFound, "Утасны дугаар олдсонгүй")
		return
	}
	uid := ""
	if p, ok := s.principal(r); ok && !p.IsGuest() {
		uid = p.UID
	}
	if uid != t.ID {
		visitor := uid
		switch vid := cleanVID(in.VID); {
		case visitor != "":
		case vid != "":
			visitor = "a:" + vid
		default:
			visitor = "ip:" + s.tokens.MAC("ip", s.clientIP(r))
		}
		if s.notif.firstPhoneView(t.ID + "|" + visitor) {
			v := &store.PageVisit{TeacherID: t.ID, Kind: "phone", TargetID: t.ID, At: time.Now(), VisitorID: visitor,
				Device: deviceOf(r.UserAgent()), Referrer: refName("")}
			who := "Зочин"
			if uid != "" {
				v.UserName = s.displayName(r.Context(), uid, "")
				who = v.UserName
			}
			if err := s.store.SaveVisit(r.Context(), v); err != nil {
				s.log.Warn("утас харсныг бүртгэх", "err", err)
			}
			s.notify(r.Context(), &store.Notification{UserID: t.ID, Type: NotifPhoneView, Title: "📞 " + who + " утасны дугаарыг тань харлаа",
				Body: "Удахгүй тантай холбогдож магадгүй.", Link: "/t/" + t.Username})
		}
	}
	display, tel, _ := phoneParts(t.Phone)
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]string{"display": display, "tel": tel})
}
