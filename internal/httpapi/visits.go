package httpapi

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---- Хуудасны үзэлт: хэдэн хүн, хэдэн удаа, хэр удаан, хаанаас, ямар төхөөрөмжөөр — багшид статистик ----

var botRe = regexp.MustCompile(`(?i)bot|crawl|spider|slurp|facebookexternalhit|embedly|preview|headless|lighthouse`)

func deviceOf(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "ipad") || strings.Contains(l, "tablet") || (strings.Contains(l, "android") && !strings.Contains(l, "mobile")):
		return "tablet"
	case strings.Contains(l, "mobi") || strings.Contains(l, "iphone") || strings.Contains(l, "android"):
		return "mobile"
	}
	return "desktop"
}

// refName — эх сурвалжийн домэйныг ойлгомжтой нэр болгоно.
func refName(ref string) string {
	h := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ref)), "www.")
	switch {
	case h == "":
		return "Шууд"
	case h == "self":
		return "surgalt.mn дотроос"
	case strings.Contains(h, "facebook.") || h == "fb.com" || strings.HasSuffix(h, ".fb.com") || strings.Contains(h, "messenger."):
		return "Facebook"
	case strings.Contains(h, "instagram."):
		return "Instagram"
	case strings.Contains(h, "google."):
		return "Google"
	case h == "t.co" || strings.Contains(h, "twitter.") || h == "x.com":
		return "X (Twitter)"
	case strings.Contains(h, "youtube.") || h == "youtu.be":
		return "YouTube"
	case strings.Contains(h, "tiktok."):
		return "TikTok"
	case strings.Contains(h, "linkedin."):
		return "LinkedIn"
	case strings.Contains(h, "bing."):
		return "Bing"
	}
	if len(h) > 60 {
		h = h[:60]
	}
	return h
}

var vidRe = regexp.MustCompile(`[^A-Za-z0-9-]`)

func cleanVID(v string) string {
	v = vidRe.ReplaceAllString(v, "")
	if len(v) > 40 {
		v = v[:40]
	}
	if len(v) < 6 {
		return ""
	}
	return v
}

// handleVisitEnd: POST /api/views/{id}/end {t, seconds} — хуудсанд хэр удаан (идэвхтэй) байсныг шинэчилнэ (sendBeacon).
func (s *Server) handleVisitEnd(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		T       string `json:"t"`
		Seconds int    `json:"seconds"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !constantEq(in.T, s.tokens.MAC("visit", id)) {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	v, err := s.store.VisitByID(r.Context(), id)
	if s.storeErr(w, r, err) {
		return
	}
	if sec := min(max(in.Seconds, 0), 6*3600); sec > v.Seconds { // хугацаа зөвхөн өснө
		v.Seconds = sec
		if err := s.store.SaveVisit(r.Context(), v); s.storeErr(w, r, err) {
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type visitDay struct {
	Day     string `json:"day"`
	Views   int    `json:"views"`
	Unique  int    `json:"unique"`
	Seconds int    `json:"seconds"`
}

type visitRow struct {
	Name     string    `json:"name"`
	Member   bool      `json:"member"`
	At       time.Time `json:"at"`
	Seconds  int       `json:"seconds"`
	Device   string    `json:"device"`
	Referrer string    `json:"referrer"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title"`
}

type nameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// handleMyVisits: GET /api/me/visits?days=30&kind=profile|course|all — багшийн нээлттэй хуудасны үзэлтийн статистик.
func (s *Server) handleMyVisits(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}
	kind := r.URL.Query().Get("kind")
	if kind != "course" && kind != "all" {
		kind = "profile"
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	all, err := s.store.Visits(r.Context(), c.UID, start)
	if s.storeErr(w, r, err) {
		return
	}
	titles := map[string]string{}
	if cs, err := s.store.CoursesByTeacher(r.Context(), c.UID, false); err == nil {
		for _, co := range cs {
			titles[co.ID] = co.Title
		}
	}
	type dayAgg struct {
		views, sec int
		uniq       map[string]bool
	}
	byDay := map[string]*dayAgg{}
	uniq, members := map[string]bool{}, map[string]bool{}
	devices, refs := map[string]int{}, map[string]int{}
	buckets := []nameCount{{"10 сек хүртэл", 0}, {"10–30 сек", 0}, {"30 сек – 1 мин", 0}, {"1–3 мин", 0}, {"3 мин+", 0}}
	type courseAgg struct {
		views, sec, measured int
		uniq                 map[string]bool
	}
	perCourse := map[string]*courseAgg{}
	views, totalSec, measured, bounce := 0, 0, 0, 0
	recent := []visitRow{}
	phones := 0
	for _, v := range all { // шинээс хуучин
		if v.Kind == "phone" { // утасны дугаар харсан: үзэлтэд тооцохгүй, профайлын хүрээнд тусад нь
			if kind != "course" {
				phones++
				if len(recent) < 30 {
					name := v.UserName
					if name == "" {
						name = "Зочин"
					}
					recent = append(recent, visitRow{Name: name, Member: v.UserName != "", At: v.At, Device: v.Device, Kind: "phone", Title: "Утасны дугаар харсан"})
				}
			}
			continue
		}
		if kind != "all" && v.Kind != kind {
			continue
		}
		views++
		member := !strings.HasPrefix(v.VisitorID, "a:") && !strings.HasPrefix(v.VisitorID, "ip:")
		uniq[v.VisitorID] = true
		if member {
			members[v.VisitorID] = true
		}
		devices[v.Device]++
		refs[v.Referrer]++
		day := v.At.In(now.Location()).Format("2006-01-02")
		d := byDay[day]
		if d == nil {
			d = &dayAgg{uniq: map[string]bool{}}
			byDay[day] = d
		}
		d.views++
		d.sec += v.Seconds
		d.uniq[v.VisitorID] = true
		if v.Seconds > 0 {
			measured++
			totalSec += v.Seconds
			switch {
			case v.Seconds < 10:
				buckets[0].Count++
				bounce++
			case v.Seconds < 30:
				buckets[1].Count++
			case v.Seconds < 60:
				buckets[2].Count++
			case v.Seconds < 180:
				buckets[3].Count++
			default:
				buckets[4].Count++
			}
		}
		if v.Kind == "course" {
			pc := perCourse[v.TargetID]
			if pc == nil {
				pc = &courseAgg{uniq: map[string]bool{}}
				perCourse[v.TargetID] = pc
			}
			pc.views++
			pc.uniq[v.VisitorID] = true
			if v.Seconds > 0 {
				pc.measured++
				pc.sec += v.Seconds
			}
		}
		if len(recent) < 30 {
			name, title := v.UserName, "Профайл"
			if !member || name == "" {
				name = "Зочин"
			}
			if v.Kind == "course" {
				title = titles[v.TargetID]
			}
			recent = append(recent, visitRow{Name: name, Member: member, At: v.At, Seconds: v.Seconds, Device: v.Device, Referrer: v.Referrer, Kind: v.Kind, Title: title})
		}
	}
	daily := make([]visitDay, 0, days)
	for i := 0; i < days; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		vd := visitDay{Day: key}
		if d := byDay[key]; d != nil {
			vd.Views, vd.Unique, vd.Seconds = d.views, len(d.uniq), d.sec
		}
		daily = append(daily, vd)
	}
	sorted := func(m map[string]int, limit int) []nameCount {
		out := make([]nameCount, 0, len(m))
		for k, v := range m {
			out = append(out, nameCount{k, v})
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].Count > out[j].Count || (out[i].Count == out[j].Count && out[i].Name < out[j].Name)
		})
		if len(out) > limit {
			out = out[:limit]
		}
		return out
	}
	courses := []map[string]any{}
	for id, pc := range perCourse {
		avg := 0
		if pc.measured > 0 {
			avg = pc.sec / pc.measured
		}
		courses = append(courses, map[string]any{"id": id, "title": titles[id], "views": pc.views, "unique": len(pc.uniq), "avg_sec": avg})
	}
	sort.Slice(courses, func(i, j int) bool { return courses[i]["views"].(int) > courses[j]["views"].(int) })
	if len(courses) > 8 {
		courses = courses[:8]
	}
	avg, bouncePct := 0, 0
	if measured > 0 {
		avg, bouncePct = totalSec/measured, bounce*100/measured
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"days": days, "kind": kind,
		"totals": map[string]any{"views": views, "unique": len(uniq), "members": len(members), "avg_sec": avg, "total_sec": totalSec, "measured": measured, "bounce_pct": bouncePct, "phone": phones},
		"daily":  daily, "devices": sorted(devices, 3), "referrers": sorted(refs, 6), "durations": buckets, "recent": recent, "courses": courses,
	})
}
