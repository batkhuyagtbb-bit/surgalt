package httpapi

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"surgalt/internal/store"
)

// ---- Багшийн системийн эрх (багц): үнэ, төлөв, QPay-ээр сунгах, эрх дуусахад хаах, сануулга ----

const (
	planExpiredMsg = "Таны системийн эрх дууссан. Профайлын «Данс» хэсгээс багц авч сунгана уу."
	sellerShutMsg  = "Энэ багшийн эрх түр хаагдсан тул одоогоор худалдаж авах, бүртгүүлэх боломжгүй."
)

var errPlanSame = errors.New("сануулга илгээгдсэн")

// planUntil — багшийн эрх дуусах хугацаа; багш биш бол хязгааргүй. Багшийн хүсэлт бүрт өгөгдлийн сан руу
// хандахгүйн тулд 1 минут кэшлэнэ (худалдан авалт, олголтын дараа шууд цэвэрлэнэ).
func (s *Server) planUntil(ctx context.Context, uid string) (time.Time, error) {
	return s.planEnds.GetOrLoad(ctx, uid, func(ctx context.Context) (time.Time, error) {
		u, err := s.store.UserByID(ctx, uid)
		if err != nil {
			return time.Time{}, err
		}
		if u.Role != store.RoleTeacher {
			return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), nil
		}
		return u.PlanUntil(), nil
	})
}

// planOpen — багш системийн эрхтэй эсэх; эрхгүй бол 402 буцаана. Өгөгдлийн сангийн алдаанд хаахгүй.
func (s *Server) planOpen(w http.ResponseWriter, r *http.Request, uid string) bool {
	until, err := s.planUntil(r.Context(), uid)
	if err != nil || time.Now().Before(until) {
		return true
	}
	writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": planExpiredMsg, "code": "plan_expired"})
	return false
}

// sellerOpen — эрх дууссан багшийн сургалт, хичээл, ном, уулзалтыг шинээр худалдаж авах, бүртгүүлэх боломжгүй
// (өмнө нь авсан суралцагчдын эрх хэвээр).
func (s *Server) sellerOpen(w http.ResponseWriter, r *http.Request, teacherID string) bool {
	if teacherID == "" {
		return true
	}
	until, err := s.planUntil(r.Context(), teacherID)
	if err != nil || time.Now().Before(until) {
		return true
	}
	writeErr(w, http.StatusForbidden, sellerShutMsg)
	return false
}

type planPeriodView struct {
	Plan  string    `json:"plan"`
	Name  string    `json:"name"`
	Pct   float64   `json:"commission_pct"`
	From  time.Time `json:"from"`
	Until time.Time `json:"until"`
}

func planName(id string) string {
	if id == store.PlanTrial {
		return "Үнэгүй хугацаа"
	}
	if p, ok := store.PlanByID(id); ok {
		return p.Name
	}
	return ""
}

// planCatalog — u-д худалдаалж буй багцууд ба гүйлгээний шимтгэл (fee_pct; trial_pct — хуучин нэр).
func planCatalog(u *store.User) map[string]any {
	return map[string]any{"plans": store.PlansFor(u), "trial_months": store.PlanTrialMonths, "trial_pct": u.FeePct(), "fee_pct": u.FeePct()}
}

// planStatus — багшийн эрхийн төлөв (Данс хэсэг, студийн хаалт, сануулгын мөрөнд).
func planStatus(u *store.User, now time.Time) map[string]any {
	id, pct, active := u.PlanAt(now)
	until := u.PlanUntil()
	days := 0
	if active {
		days = int(math.Ceil(until.Sub(now).Hours() / 24))
	}
	upcoming := []planPeriodView{}
	for _, p := range u.Plan.Periods {
		if p.From.After(now) {
			pl, _ := store.PlanByID(p.Plan)
			upcoming = append(upcoming, planPeriodView{Plan: p.Plan, Name: pl.Name, Pct: pl.Pct, From: p.From, Until: p.Until})
		}
	}
	st := planCatalog(u)
	st["plan"], st["name"], st["commission_pct"], st["active"] = id, planName(id), pct, active
	st["until"], st["days_left"], st["trial_until"], st["upcoming"] = until, days, u.TrialUntil(), upcoming
	if u.Org != nil { // байгууллага: байгууллагын багц, хянагчийн тогтоосон шимтгэл, гэрээний файлын сан
		st["org"], st["storage_mb"] = true, u.Org.StorageMB
	}
	return st
}

// handlePlans: GET /api/plans — үнийн жагсаалт (нээлттэй): багш, байгууллагын багц.
func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	st := planCatalog(&store.User{})
	st["org_plans"], st["org_pct"] = store.OrgPlans, store.OrgFeePct
	writeJSON(w, http.StatusOK, st)
}

// handleMyPlan: GET /api/me/plan — багшийн эрх (эрх дууссан ч харна).
func (s *Server) handleMyPlan(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, planStatus(u, time.Now()))
}

// handleBuyPlan: POST /api/me/plan/purchase {plan} → захиалга + QPay. Төлөгдмөгц эрх сунгагдана (store.AddPlan).
func (s *Server) handleBuyPlan(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	var in struct {
		Plan string `json:"plan"`
	}
	if !decode(w, r, &in) {
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	p, ok := store.PlanOnSale(u, in.Plan) // байгууллагад зөвхөн байгууллагын багц
	if !ok {
		writeErr(w, http.StatusBadRequest, "ийм багц байхгүй")
		return
	}
	o, err := s.store.CreatePlanOrder(r.Context(), c.UID, p)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": o, "payment": s.paymentInfo(r, o)})
}

// planPaid — багц төлөгдсөний дараа (notifyPaid).
func (s *Server) planPaid(ctx context.Context, o *store.Order) {
	s.planEnds.Delete(o.UserID)
	body := o.Title
	if u, err := s.store.UserByID(ctx, o.UserID); err == nil {
		body += " · дуусах: " + u.PlanUntil().In(mnLoc).Format("2006.01.02")
	}
	s.notify(ctx, &store.Notification{UserID: o.UserID, Type: NotifPlan, Count: 1, Title: "✅ Системийн эрх сунгагдлаа", Body: body, Link: "/me#wallet"})
	s.hub.SendUsers([]string{o.UserID}, map[string]any{"type": "plan_changed"})
}

// handleAdminGrantPlan: POST /api/admin/teachers/{id}/plan {plan} — хянагч багц үнэгүй олгоно (байгууллага, туршилт).
func (s *Server) handleAdminGrantPlan(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		Plan string `json:"plan"`
	}
	if !decode(w, r, &in) {
		return
	}
	var p store.Plan
	u, err := s.store.UpdateUser(r.Context(), r.PathValue("id"), func(x *store.User) error {
		if x.Role != store.RoleTeacher {
			return errNotTeacher
		}
		pl, ok := store.PlanOnSale(x, in.Plan) // байгууллагад байгууллагын багц
		if !ok {
			return errNoPlan
		}
		p = pl
		x.AddPlan(p, "", time.Now())
		return nil
	})
	if errors.Is(err, errNotTeacher) {
		writeErr(w, http.StatusConflict, "энэ хэрэглэгч багш биш")
		return
	}
	if errors.Is(err, errNoPlan) {
		writeErr(w, http.StatusBadRequest, "ийм багц байхгүй")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	s.planEnds.Delete(u.ID)
	s.adminInvalidate()
	s.notify(r.Context(), &store.Notification{UserID: u.ID, Type: NotifPlan, Count: 1, ActorID: admin.ID, Title: "🎁 Танд системийн эрх олгогдлоо",
		Body: p.Title() + " · " + store.PlanTerm(p.Months) + " · дуусах: " + u.PlanUntil().In(mnLoc).Format("2006.01.02"), Link: "/me#wallet"})
	s.hub.SendUsers([]string{u.ID}, map[string]any{"type": "plan_changed"})
	s.log.Info("хянагч багц олгов", "user", u.ID, "plan", p.ID, "admin", admin.ID)
	writeJSON(w, http.StatusOK, planStatus(u, time.Now()))
}

var (
	errNotTeacher = errors.New("багш биш")
	errNoPlan     = errors.New("ийм багц байхгүй")
)

// planLoop — эрх дуусахаас 7 ба 1 хоногийн өмнө, дуусахад, дараа нь 4 долоо хоног долоо хоног бүр багшид нэг удаа мэдэгдэнэ.
func (s *Server) planLoop(ctx context.Context) {
	first := time.NewTimer(2 * time.Minute)
	defer first.Stop()
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-first.C:
			s.planNotices(ctx, time.Now())
		case <-t.C:
			s.planNotices(ctx, time.Now())
		}
	}
}

func (s *Server) planNotices(ctx context.Context, now time.Time) {
	users, err := s.store.AdminUsers(ctx)
	if err != nil {
		s.log.Warn("багцын сануулга", "err", err)
		return
	}
	for i := range users {
		u := &users[i]
		if u.Role != store.RoleTeacher {
			continue
		}
		n, key := planNotice(u, now)
		if n == nil {
			continue
		}
		if _, err := s.store.UpdateUser(ctx, u.ID, func(x *store.User) error {
			if x.Plan.Notice == key || x.PlanUntil() != u.PlanUntil() {
				return errPlanSame // илгээсэн, эсвэл энэ хооронд сунгасан
			}
			x.Plan.Notice = key
			return nil
		}); err != nil {
			continue
		}
		s.notify(ctx, n)
		s.planEnds.Delete(u.ID)
		if strings.HasSuffix(key, ":0") {
			s.hub.SendUsers([]string{u.ID}, map[string]any{"type": "plan_changed"})
		}
	}
}

// planNotice — илгээх сануулга ба давтахгүй байх түлхүүр (дуусах өдөр:үе шат), эсвэл nil.
func planNotice(u *store.User, now time.Time) (*store.Notification, string) {
	until := u.PlanUntil()
	left := until.Sub(now)
	stage := ""
	switch {
	case left <= 0:
		if left < -30*24*time.Hour {
			return nil, "" // удаан хугацаанд эрхгүй — дахин сануулахгүй
		}
		stage = "0"
		if w := int(-left / (7 * 24 * time.Hour)); w >= 1 {
			stage = "w" + strconv.Itoa(w) // дууссанаас хойш долоо хоног бүр төлбөр сануулна
		}
	case left <= 24*time.Hour:
		stage = "1"
	case left <= 7*24*time.Hour:
		stage = "7"
	default:
		return nil, ""
	}
	key := until.UTC().Format("20060102") + ":" + stage
	if u.Plan.Notice == key {
		return nil, ""
	}
	what := "Системийн эрх"
	if id, _, _ := u.PlanAt(until.Add(-time.Second)); id == store.PlanTrial {
		what = "Үнэгүй хугацаа"
	}
	date := until.In(mnLoc).Format("2006.01.02")
	n := &store.Notification{UserID: u.ID, Type: NotifPlan, Count: 1, Link: "/me#wallet"}
	switch stage {
	case "7":
		n.Title, n.Body = "⏳ "+what+" 7 хоногийн дараа дуусна", date+"-нд дуусна. «Данс» хэсгээс багц авч сунгаарай — сургалтаа тасралтгүй удирдана."
	case "1":
		n.Title, n.Body = "⚠️ "+what+" маргааш дуусна", date+"-нд дуусна. Сунгаагүй бол сургалт удирдах, шинээр борлуулах боломж хаагдана."
	case "0":
		n.Title, n.Body = "⛔ "+what+" дууслаа", "Сургалт удирдах, шинээр борлуулах боломж хаагдлаа. «Данс» хэсгээс багц авч сунгана уу. Худалдаж авсан суралцагчдын эрх хэвээр."
	default:
		n.Title = "💳 Системийн эрхийн төлбөрөө төлнө үү"
		n.Body = date + "-нд дууссан тул сургалт удирдах, шинээр борлуулах боломж хаалттай байна. «Данс» хэсгээс багц сонгоод QPay-ээр төлмөгц шууд нээгдэнэ."
	}
	return n, key
}
