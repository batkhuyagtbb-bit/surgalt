package httpapi

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Шалгалт, даалгаврын дуусах хугацаа ба хоцорсон тохиолдлын бодлого (төлбөргүй / төлбөртэй / хаалттай),
// даалгаврын хариу илгээх, багш дүгнэх.

// DueState — суралцагчид одоо юу харагдах вэ: эхлээгүй / нээлттэй / хоцорсон (төлбөргүй, төлбөртэй, хаалттай),
// оролцооны төлбөр (шууд төлбөртэй шалгалт). Fee = одоо төлөх нийт дүн (оролцоо + хоцролт).
type DueState struct {
	StartAt    *time.Time `json:"start_at,omitempty"`
	At         *time.Time `json:"at,omitempty"`
	Policy     string     `json:"policy,omitempty"` // free | paid | closed
	NotStarted bool       `json:"not_started"`      // эхлэх цаг болоогүй
	Late       bool       `json:"late"`             // дуусах хугацаа өнгөрсөн
	Open       bool       `json:"open"`             // одоо илгээж/өгч болно
	NeedPay    bool       `json:"need_pay"`         // төлбөр төлж нээнэ (оролцоо ба/эсвэл хоцролт)
	Closed     bool       `json:"closed"`           // хаалттай
	Fee        int64      `json:"fee,omitempty"`    // одоо төлөх нийт дүн
	EntryFee   int64      `json:"entry_fee,omitempty"`
	LateFee    int64      `json:"late_fee,omitempty"`
	Paid       bool       `json:"paid,omitempty"`      // шаардлагатай төлбөрүүд төлөгдсөн
	NeedLate   bool       `json:"need_late,omitempty"` // хоцролтын төлбөр дутуу
	NeedEntry  bool       `json:"need_entry,omitempty"`
}

func dueStateOf(d store.Due, latePass, feePass bool, now time.Time) DueState {
	ds := DueState{StartAt: d.StartAt, At: d.At, Policy: d.Late, Open: true, EntryFee: d.Fee}
	if d.StartAt != nil && now.Before(*d.StartAt) {
		ds.NotStarted, ds.Open = true, false
		return ds
	}
	if d.At != nil && now.After(*d.At) {
		ds.Late = true
		switch d.Late {
		case "closed":
			ds.Open, ds.Closed = false, true
			return ds
		case "paid":
			ds.LateFee = d.LateFee
			if !latePass {
				ds.NeedLate = true
			}
		}
	}
	if d.Fee > 0 && !feePass {
		ds.NeedEntry = true
	}
	if ds.NeedLate || ds.NeedEntry {
		ds.Open, ds.NeedPay = false, true
		if ds.NeedEntry {
			ds.Fee += d.Fee
		}
		if ds.NeedLate {
			ds.Fee += d.LateFee
		}
	} else if d.Fee > 0 || (ds.Late && d.Late == "paid") {
		ds.Paid = true
	}
	return ds
}

func validateDue(d *store.Due) string {
	switch {
	case d.Late != "" && store.LatePolicies[d.Late] == "":
		return "хоцорсон тохиолдлын бодлого: төлбөргүй, төлбөртэй, хаалттай"
	case d.Late == "paid" && (d.LateFee <= 0 || d.LateFee > maxPrice):
		return "хоцорсон тохиолдлын төлбөрийг заана уу (1-100,000,000₮)"
	case d.Late != "paid" && d.LateFee != 0:
		d.LateFee = 0
	}
	if d.Fee < 0 || d.Fee > maxPrice {
		return "оролцооны төлбөр 0-100,000,000₮"
	}
	if d.StartAt != nil && d.StartAt.IsZero() {
		d.StartAt = nil
	}
	if d.At != nil && d.At.IsZero() {
		d.At = nil
	}
	if d.StartAt != nil && d.At != nil && !d.At.After(*d.StartAt) {
		return "дуусах цаг эхлэх цагаас хойш байна"
	}
	return ""
}

func validateAssignment(a *store.Assignment) string {
	if a == nil {
		return ""
	}
	if a.MaxScore <= 0 {
		a.MaxScore = 100
	}
	if a.MaxScore > 1000 {
		return "даалгаврын дээд оноо 1-1000"
	}
	a.AllowFiles = false // хариу зөвхөн текст ба холбоосоор — файл илгээхгүй
	return validateDue(&a.Due)
}

// passes — хоцролтын ба оролцооны төлбөрийг төлсөн эсэх (явцын тэмдгүүд).
func (s *Server) passes(r *http.Request, uid, courseID, lessonID string) (late, fee bool) {
	prog, err := s.store.LessonProgress(r.Context(), uid, courseID)
	if err != nil {
		return false, false
	}
	q := prog[lessonID].Quiz
	return q[store.LatePassKey], q[store.FeePassKey]
}

// dueFor — тухайн хичээлийн (шалгалт эсвэл даалгавар) хугацааны төлөв; багшид үргэлж нээлттэй.
func (s *Server) dueFor(r *http.Request, uid string, course *store.Course, l *store.Lesson) DueState {
	var d store.Due
	switch {
	case l.Exam != nil:
		d = l.Exam.Due
	case l.Assignment != nil:
		d = l.Assignment.Due
	}
	if course.TeacherID == uid {
		return DueState{StartAt: d.StartAt, At: d.At, Policy: d.Late, Open: true, EntryFee: d.Fee}
	}
	lp, fp := s.passes(r, uid, course.ID, l.ID)
	return dueStateOf(d, lp, fp, time.Now())
}

// dueBlocked — эхлээгүй/хаалттай (423) эсвэл төлбөр шаардлагатай (402) бол хариулаад true.
func dueBlocked(w http.ResponseWriter, ds DueState) bool {
	switch {
	case ds.NotStarted:
		writeJSON(w, http.StatusLocked, map[string]any{"error": "эхлэх цаг болоогүй: " + ds.StartAt.Local().Format("01/02 15:04"), "due": ds})
		return true
	case ds.Closed:
		writeJSON(w, http.StatusLocked, map[string]any{"error": "хугацаа дууссан — хаалттай", "due": ds})
		return true
	case ds.NeedPay:
		msg := fmt.Sprintf("оролцооны төлбөр %d₮ төлж нээнэ", ds.Fee)
		if ds.NeedLate {
			msg = fmt.Sprintf("хугацаа хоцорсон — %d₮ төлж нээнэ", ds.Fee)
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"error": msg, "due": ds})
		return true
	}
	return false
}

// handleLatePay: POST .../late-pay — хоцорсон шалгалт/даалгаврыг нээх захиалга.
func (s *Server) handleLatePay(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	ds := s.dueFor(r, c.UID, course, l)
	if !ds.NeedPay {
		writeJSON(w, http.StatusOK, map[string]any{"unlocked": true, "due": ds})
		return
	}
	kind := store.OrderKindFee
	if ds.NeedLate {
		kind = store.OrderKindLate // хоцролт + (дутуу бол) оролцоо нэг төлбөрөөр
	}
	o, err := s.store.CreateOrGetPendingPassOrder(r.Context(), c.UID, course, l, kind, ds.Fee)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"unlocked": false, "order": o,
		"payment": s.paymentInfo(r, o)})
}

func (s *Server) subView(sub *store.Submission) map[string]any {
	urls := make([]map[string]string, 0, len(sub.Files))
	for _, p := range sub.Files {
		name := p[strings.LastIndexByte(p, '/')+1:]
		if i := strings.Index(name, "__"); i >= 0 {
			name = name[i+2:]
		}
		urls = append(urls, map[string]string{"name": name, "url": s.media(p)})
	}
	links := sub.Links
	if links == nil {
		links = []string{}
	}
	out := map[string]any{"user_id": sub.UserID, "user_name": sub.UserName, "text": sub.Text, "files": urls, "links": links,
		"submitted_at": sub.SubmittedAt, "late": sub.Late, "feedback": sub.Feedback, "graded_at": sub.GradedAt}
	if sub.Score != nil {
		out["score"] = *sub.Score
	}
	return out
}

// handleAssignmentInfo: GET .../assignment — даалгаврын нөхцөл, хугацаа, өөрийн хариу.
func (s *Server) handleAssignmentInfo(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if l.Assignment == nil {
		writeErr(w, http.StatusNotFound, "энэ хичээл даалгавар биш")
		return
	}
	out := map[string]any{"assignment": l.Assignment, "due": s.dueFor(r, c.UID, course, l)}
	if sub, err := s.store.SubmissionFor(r.Context(), c.UID, l.ID); err == nil {
		out["submission"] = s.subView(sub)
	}
	writeJSON(w, http.StatusOK, out)
}

const maxSubmissionFiles = 3

// handleSubmit: POST .../submit — multipart (text, file...) эсвэл JSON {text}. Дахин илгээвэл шинэчлэгдэнэ.
func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if l.Assignment == nil {
		writeErr(w, http.StatusNotFound, "энэ хичээл даалгавар биш")
		return
	}
	if course.TeacherID == c.UID {
		writeErr(w, http.StatusBadRequest, "багш өөрийн даалгаварт хариу илгээхгүй")
		return
	}
	ds := s.dueFor(r, c.UID, course, l)
	if dueBlocked(w, ds) {
		return
	}
	sub := &store.Submission{UserID: c.UID, UserName: s.displayName(r.Context(), c.UID, c.Name), CourseID: course.ID, LessonID: l.ID,
		TeacherID: course.TeacherID, SubmittedAt: time.Now(), Late: ds.Late, Files: []string{}, Links: []string{}}
	var rawLinks []string
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct == "multipart/form-data" {
		mr, err := r.MultipartReader()
		if err != nil {
			writeErr(w, http.StatusBadRequest, "multipart/form-data хэлбэрээр илгээнэ үү")
			return
		}
		var quota int64 = -1
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeErr(w, http.StatusBadRequest, "илгээлт тасарлаа")
				return
			}
			if part.FormName() == "text" {
				b, _ := io.ReadAll(io.LimitReader(part, 60_000))
				sub.Text = string(b)
				continue
			}
			if part.FormName() == "links" { // мөр эсвэл таслалаар тусгаарласан холбоосууд
				b, _ := io.ReadAll(io.LimitReader(part, 10_000))
				rawLinks = append(rawLinks, strings.FieldsFunc(string(b), func(r rune) bool { return r == '\n' || r == ',' || r == ' ' })...)
				continue
			}
			if part.FileName() == "" {
				continue
			}
			if !l.Assignment.AllowFiles { // одоогоор үргэлж: хариуг текст, холбоосоор илгээнэ
				writeErr(w, http.StatusBadRequest, "даалгаврын хариуд файл илгээхгүй — текст болон холбоос (Google Docs, видео г.м) оруулна уу")
				return
			}
			if len(sub.Files) >= maxSubmissionFiles {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("дээд тал нь %d файл", maxSubmissionFiles))
				return
			}
			if quota < 0 { // багшийн сангийн багтаамжид хадгална
				u, err := s.store.UserByID(r.Context(), course.TeacherID)
				if s.storeErr(w, r, err) {
					return
				}
				quota = s.quotaOf(u)
			}
			info, err := s.files.Save(course.TeacherID, files.Private, "daalgavar_"+tail(c.UID, 6)+"_"+part.FileName(), part, quota)
			if err != nil {
				s.filesErr(w, err)
				return
			}
			sub.Files = append(sub.Files, info.Path)
		}
	} else {
		var in struct {
			Text  string   `json:"text"`
			Links []string `json:"links"`
		}
		if !decode(w, r, &in) {
			return
		}
		sub.Text, rawLinks = in.Text, in.Links
	}
	for _, u := range rawLinks {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !validURL(u) || len(u) > 500 {
			writeErr(w, http.StatusBadRequest, "холбоос http(s)://-ээр эхэлсэн байна: "+short(u))
			return
		}
		if len(sub.Links) >= 5 {
			writeErr(w, http.StatusBadRequest, "дээд тал нь 5 холбоос")
			return
		}
		sub.Links = append(sub.Links, u)
	}
	sub.Text = strings.TrimSpace(sub.Text)
	if sub.Text == "" && len(sub.Files) == 0 && len(sub.Links) == 0 {
		writeErr(w, http.StatusBadRequest, "хариугаа бичих, холбоос эсвэл файл хавсаргана уу")
		return
	}
	if utf8.RuneCountInString(sub.Text) > 20_000 {
		writeErr(w, http.StatusBadRequest, "хариу хэт урт (20,000 тэмдэгт)")
		return
	}
	if err := s.store.SaveSubmission(r.Context(), sub); s.storeErr(w, r, err) {
		return
	}
	_ = s.store.MarkLessonCompleted(r.Context(), c.UID, course.ID, l.ID)
	s.notify(r.Context(), &store.Notification{UserID: course.TeacherID, Type: "assignment", Title: "📎 Даалгаврын хариу ирлээ",
		Body: sub.UserName + " — «" + l.Title + "»" + map[bool]string{true: " (хоцорсон)", false: ""}[sub.Late], Link: "/t/" + c.Name + "#course=" + course.ID})
	s.courses.Delete(course.ID)
	writeJSON(w, http.StatusOK, map[string]any{"submission": s.subView(sub), "due": ds})
}

// handleSubmissions: GET .../submissions — багшид бүх хариу.
func (s *Server) handleSubmissions(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	subs, err := s.store.Submissions(r.Context(), l.ID)
	if s.storeErr(w, r, err) {
		return
	}
	out := make([]map[string]any, len(subs))
	for i := range subs {
		out[i] = s.subView(&subs[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"lesson": l.Title, "assignment": l.Assignment, "submissions": out})
}

// handleGrade: PUT .../submissions/{uid} — оноо, тайлбар; суралцагчид мэдэгдэнэ.
func (s *Server) handleGrade(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	var in struct {
		Score    int    `json:"score"`
		Feedback string `json:"feedback"`
	}
	if !decode(w, r, &in) {
		return
	}
	maxScore := 100
	if l.Assignment != nil && l.Assignment.MaxScore > 0 {
		maxScore = l.Assignment.MaxScore
	}
	if in.Score < 0 || in.Score > maxScore {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("оноо 0-%d", maxScore))
		return
	}
	if utf8.RuneCountInString(in.Feedback) > 5000 {
		writeErr(w, http.StatusBadRequest, "тайлбар хэт урт")
		return
	}
	uid := r.PathValue("uid")
	if err := s.store.GradeSubmission(r.Context(), l.ID, uid, in.Score, strings.TrimSpace(in.Feedback)); s.storeErr(w, r, err) {
		return
	}
	s.notify(r.Context(), &store.Notification{UserID: uid, Type: "grade", Title: fmt.Sprintf("✅ Даалгавар дүгнэгдлээ: %d/%d", in.Score, maxScore),
		Body: "«" + l.Title + "»" + map[bool]string{true: " — " + in.Feedback, false: ""}[in.Feedback != ""], Link: "/c/" + course.ID + "#l=" + l.ID})
	sub, err := s.store.SubmissionFor(r.Context(), uid, l.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.subView(sub))
}
