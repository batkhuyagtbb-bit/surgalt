package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Багш нар хичээлээ хооронд хуваалцана: хичээлийн эзэмшигч «хуваалцах код» (shareTTL) авч өөр багшид өгнө; тэр багш
// өөрийн сургалтад кодыг оруулж хичээлийн хуулбарыг (блок, асуулт, шалгалт, даалгавар, тохиргоо) авна. Хичээл дэх
// файлын сангийн файлууд (видео, зураг, аудио, файл, текст доторх зураг) хүлээн авагчийн сан руу хуулагдана —
// багтаамжийг нь ашиглана, эх багш хожим файлаа устгасан ч нөлөөлөхгүй. Хуулбар «бэлтгэж байна» (hidden) төлөвтэй
// үүснэ: хүлээн авсан багш шалгаад нийтэлнэ. Код нь гарын үсэгтэй (tokens.Seal) тул сервер юу ч хадгалахгүй.

type shareClaim struct {
	K string `json:"k"` // зориулалт
	C string `json:"c"` // сургалт
	L string `json:"l"` // хичээл
	U string `json:"u"` // хуваалцсан багш
}

const (
	shareKind = "lesson-share"
	shareTTL  = 30 * 24 * time.Hour
)

// filePathRe — текстийн HTML доторх өөрийн файлын сангийн замууд (зураг г.м).
var filePathRe = regexp.MustCompile(`/files/[0-9a-f]+/(?:public|private)/[^"'\s<>)]+`)

var errSharedFile = errors.New("хуваалцсан хичээлийн файл олдсонгүй")

// handleShareLesson: POST /api/courses/{id}/lessons/{lid}/share → {code, expires, title}
func (s *Server) handleShareLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	code, err := s.tokens.Seal(shareClaim{K: shareKind, C: course.ID, L: l.ID, U: c.UID}, shareTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "код үүсгэж чадсангүй")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "expires": time.Now().Add(shareTTL), "title": l.Title})
}

// handleImportLesson: POST /api/courses/{id}/lessons/import {code, section} → шинэ (hidden) хичээл.
func (s *Server) handleImportLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Code    string `json:"code"`
		Section string `json:"section"`
	}
	if !decode(w, r, &in) {
		return
	}
	var cl shareClaim
	if err := s.tokens.Open(strings.TrimSpace(in.Code), &cl); err != nil || cl.K != shareKind {
		writeErr(w, http.StatusBadRequest, "код буруу эсвэл хугацаа нь дууссан")
		return
	}
	ctx := r.Context()
	src, err := s.store.CourseByID(ctx, cl.C)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !teaches(src, cl.U)) {
		writeErr(w, http.StatusNotFound, "хуваалцсан хичээл олдсонгүй (сургалт устсан эсвэл багш нь заахаа больсон)")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	l, err := s.store.LessonByID(ctx, src.ID, cl.L)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "хуваалцсан хичээл олдсонгүй (устсан)")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	// Гүн хуулбар: кэшээс ирсэн хичээлийн блок, асуултыг өөрчлөхгүй.
	var cp store.Lesson
	if b, err := json.Marshal(l); err != nil || json.Unmarshal(b, &cp) != nil {
		writeErr(w, http.StatusInternalServerError, "хичээлийг хуулж чадсангүй")
		return
	}
	// Файлын сангийн эзэмшигч = сургалтын эзэмшигч (байгууллагын сургалтад байгууллага) — түүний багтаамжаар.
	owner, err := s.store.UserByID(ctx, course.TeacherID)
	if s.storeErr(w, r, err) {
		return
	}
	cpFile := s.fileCopier(course.TeacherID, s.quotaOf(owner))
	in2 := lessonInput{Hidden: true, Title: cp.Title, Content: cp.Content, IsFree: cp.IsFree, Price: cp.Price, UnlockAfterH: cp.UnlockAfterH,
		AlwaysOpen: cp.AlwaysOpen, Format: cp.Format, Mode: cp.Mode, Section: in.Section, Blocks: cp.Blocks, ActiveMin: cp.ActiveMin,
		Exam: cp.Exam, Assignment: cp.Assignment, Discussion: cp.Discussion, UnlockRule: cp.UnlockRule}
	if in2.VideoURL, err = cpFile(cp.VideoURL); err != nil {
		s.importFileErr(w, err)
		return
	}
	for i := range in2.Blocks {
		b := &in2.Blocks[i]
		b.Parts = nil
		switch b.Type {
		case "image", "audio", "video", "file":
			if b.URL, err = cpFile(b.URL); err != nil {
				s.importFileErr(w, err)
				return
			}
		case "text":
			if b.HTML != "" && strings.Contains(b.HTML, "/files/") {
				var ferr error
				b.HTML = filePathRe.ReplaceAllStringFunc(b.HTML, func(p string) string {
					np, err := cpFile(p)
					if err != nil {
						ferr = err
						return p
					}
					return np
				})
				if ferr != nil {
					s.importFileErr(w, ferr)
					return
				}
			}
		}
	}
	if msg := in2.validate(course.TeacherID, course); msg != "" {
		writeErr(w, http.StatusBadRequest, "хуваалцсан хичээл: "+msg)
		return
	}
	nl := lessonFromInput(in2, course.ID)
	if err := s.store.CreateLesson(ctx, nl); s.storeErr(w, r, err) {
		return
	}
	s.courseEdited(ctx, course)
	s.log.Info("хуваалцсан хичээл орлоо", "from", cl.U, "lesson", cl.L, "to", c.UID, "course", course.ID, "new", nl.ID)
	writeJSON(w, http.StatusCreated, nl)
}

// fileCopier — өөр багшийн файлын сангийн замыг toID багшийн сан руу хуулж шинэ замыг өгнө (ижил зам нэг л удаа
// хуулагдана). Гадаад холбоос, аль хэдийн өөрийн файл бол өөрчлөхгүй.
func (s *Server) fileCopier(toID string, quota int64) func(string) (string, error) {
	done := map[string]string{}
	return func(u string) (string, error) {
		u = stripSig(strings.TrimSpace(u))
		if !strings.HasPrefix(u, "/files/") || files.OwnedBy(u, toID) {
			return u, nil
		}
		if v, ok := done[u]; ok {
			return v, nil
		}
		teacher, vis, name, ok := files.SplitPath(u)
		if !ok || s.files == nil {
			return "", errSharedFile
		}
		f, _, err := s.files.Open(teacher, vis, name)
		if err != nil {
			return "", errSharedFile
		}
		defer f.Close()
		orig := name // <санамсаргүй>__<эх нэр> → эх нэрээр
		if i := strings.Index(name, "__"); i >= 0 {
			orig = name[i+2:]
		}
		info, err := s.files.SaveOpts(toID, vis, orig, f, quota, false)
		if err != nil {
			return "", err
		}
		done[u] = info.Path
		return info.Path, nil
	}
}

func (s *Server) importFileErr(w http.ResponseWriter, err error) {
	if errors.Is(err, errSharedFile) {
		writeErr(w, http.StatusNotFound, "хуваалцсан хичээлийн файл олдсонгүй (эх багш устгасан байж магадгүй)")
		return
	}
	s.filesErr(w, err)
}
