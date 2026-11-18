// Package files нь багш бүрийн тусдаа файлын санг удирдана:
//
//	<root>/teachers/<teacherID>/public/   — нээлттэй (профайл зураг, нүүр зураг)
//	<root>/teachers/<teacherID>/private/  — төлбөртэй агуулга; зөвхөн гарын үсэгтэй, хугацаатай URL-аар
//
// Файлын нэр: <санамсаргүй hex>__<цэвэрлэсэн эх нэр>.<өргөтгөл>
package files

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("файл олдсонгүй")
	ErrBadName       = errors.New("файлын нэр буруу")
	ErrType          = errors.New("энэ төрлийн файлыг оруулах боломжгүй")
	ErrQuota         = errors.New("файлын сангийн багтаамж хүрэлцэхгүй байна — шаардлагагүй файлаа устгах эсвэл багтаамжаа нэмнэ үү")
	ErrTooLarge      = errors.New("файлын хэмжээ хэт их байна")
	ErrBadVisibility = errors.New("visibility нь public эсвэл private")
	ErrBadImage      = errors.New("зургийг уншиж чадсангүй — файл гэмтсэн эсвэл формат нь дэмжигдэхгүй байна")
)

const (
	Public  = "public"
	Private = "private"
)

var allowed = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".m4v": true, ".mp3": true, ".m4a": true, ".wav": true,
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".svg": false,
	".mkv": true, ".avi": true, ".doc": true, ".ppt": true, ".xls": true, ".odt": true, ".odp": true, ".rtf": true, ".zip": true, ".pptx": true, ".docx": true, ".xlsx": true, ".ods": true, ".csv": true, ".txt": true, ".epub": true,
}

var (
	idRe       = regexp.MustCompile(`^[0-9a-f]{24}$`)
	fileNameRe = regexp.MustCompile(`^[0-9a-f]{16}__[\p{L}\p{N}._-]{1,120}$`)
	unsafeRe   = regexp.MustCompile(`[^\p{L}\p{N}._-]+`)
)

type Info struct {
	Name         string    `json:"name"`          // дискэн дээрх нэр
	OriginalName string    `json:"original_name"` // багшийн оруулсан нэр
	Visibility   string    `json:"visibility"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	Path         string    `json:"path"`            // /files/<teacher>/<visibility>/<name> — хичээлд холбоход
	Parts        int       `json:"parts,omitempty"` // видео 6 минутын хэдэн хэсэгт хуваагдсан
	URL          string    `json:"url"`             // шууд нээх URL (private бол гарын үсэгтэй)
	ModTime      time.Time `json:"mod_time"`
	Status       string    `json:"status"` // ready | processing | failed
}

type Store struct {
	root        string
	key         []byte
	maxFileSize int64
	tc          *Transcoder
	locks       sync.Map // teacherID -> *sync.Mutex (квот шалгалтыг цуваа болгоно)
	refs        sync.Map // тасалбарын богино лавлагаа -> refHit (видео гүйлгэх бүрт хавтас уншихгүй)
}

type refHit struct{ path, disk string }

// urlWindow — гарын үсэгтэй URL, медиа тасалбарын хугацааг энэ цонхоор бүхэлтгэнэ: нэг файлд цонхны турш ижил URL
// өгч, хөтөч дахин үзэх, гүйлгэхэд файлыг кэшээсээ уншина (хугацаа секундээр байхад URL бүр шинэ байв).
const urlWindow = 6 * time.Hour

// stableExp — ttl-ээс доошгүй, цонхны хил хүртэл (ttl … ttl+urlWindow) хүчинтэй хугацаа.
func stableExp(ttl time.Duration) int64 {
	if ttl <= 0 { // аль хэдийн дууссан (тест, хүчингүй болгох) — бүхэлтгэхгүй
		return time.Now().Add(ttl).Unix()
	}
	w := int64(urlWindow / time.Second)
	return (time.Now().Add(ttl).Unix()/w + 1) * w
}

func New(root string, signKey []byte, maxFileSize int64) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "teachers"), 0o750); err != nil {
		return nil, err
	}
	return &Store{root: root, key: signKey, maxFileSize: maxFileSize}, nil
}

func (s *Store) teacherDir(teacherID string) (string, error) {
	if !idRe.MatchString(teacherID) {
		return "", ErrBadName
	}
	return filepath.Join(s.root, "teachers", teacherID), nil
}

// DeleteOwner нь бүртгэлээ устгасан хэрэглэгчийн бүх файлыг (хавтсыг нь бүхэлд) устгана.
func (s *Store) DeleteOwner(ownerID string) error {
	dir, err := s.teacherDir(ownerID)
	if err != nil {
		return err
	}
	if nd, err := s.notesDir(ownerID); err == nil { // тэмдэглэлийн зургууд (баримтаас хүрээлж авсан)
		if err := os.RemoveAll(nd); err != nil {
			return err
		}
	}
	return os.RemoveAll(dir)
}

// EnsureTeacherDir нь багшийн public/private хавтсыг үүсгэнэ (давтан дуудахад аюулгүй).
func (s *Store) EnsureTeacherDir(teacherID string) error {
	dir, err := s.teacherDir(teacherID)
	if err != nil {
		return err
	}
	for _, v := range []string{Public, Private} {
		if err := os.MkdirAll(filepath.Join(dir, v), 0o750); err != nil {
			return err
		}
	}
	return nil
}

// Usage нь багшийн хавтасны нийт хэмжээ.
func (s *Store) Usage(teacherID string) (int64, error) {
	dir, err := s.teacherDir(teacherID)
	if err != nil {
		return 0, err
	}
	var total int64
	err = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	return total, err
}

func sanitize(original string) (base, ext string, err error) {
	original = filepath.Base(strings.ReplaceAll(original, "\\", "/"))
	ext = strings.ToLower(filepath.Ext(original))
	if !allowed[ext] {
		return "", "", ErrType
	}
	base = strings.Trim(unsafeRe.ReplaceAllString(strings.TrimSuffix(original, filepath.Ext(original)), "_"), "._-")
	if r := []rune(base); len(r) > 100 {
		base = string(r[:100])
	}
	if base == "" {
		base = "file"
	}
	return base, ext, nil
}

// Save нь r-ийг багшийн хавтас руу урсгалаар бичнэ (санах ойд бүтнээр ачаалахгүй).
// quota нь тухайн багшийн (төлбөрөөс хамаарсан) нийт багтаамж.
// Зураг бол урт талыг MaxImageSide хүртэл багасгана.
func (s *Store) Save(teacherID, visibility, originalName string, r io.Reader, quota int64) (*Info, error) {
	return s.SaveOpts(teacherID, visibility, originalName, r, quota, false)
}

// SaveOpts — officePDF=true бол Word/PowerPoint-ийг заавал PDF болгоно (ном). Үгүй бол .pptx/.docx эх
// хэвээр (хичээл дотор шууд харагдана), зөвхөн хуучин формат (.ppt/.doc …) PDF болно.
func (s *Store) SaveOpts(teacherID, visibility, originalName string, r io.Reader, quota int64, officePDF bool) (*Info, error) {
	if visibility != Public && visibility != Private {
		return nil, ErrBadVisibility
	}
	base, ext, err := sanitize(originalName)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureTeacherDir(teacherID); err != nil {
		return nil, err
	}
	mu, _ := s.locks.LoadOrStore(teacherID, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	used, err := s.Usage(teacherID)
	if err != nil {
		return nil, err
	}
	remaining := quota - used
	if remaining <= 0 {
		return nil, ErrQuota
	}
	limit := min(remaining, s.maxFileSize)

	dir, _ := s.teacherDir(teacherID)
	vdir := filepath.Join(dir, visibility)
	tmp, err := os.CreateTemp(vdir, ".upload-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name()) // амжилттай бол rename хийгдсэн тул алга
	n, err := io.Copy(tmp, io.LimitReader(r, limit+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	if n > limit {
		if limit == s.maxFileSize {
			return nil, ErrTooLarge
		}
		return nil, ErrQuota
	}
	if isImageExt(ext) {
		if _, _, err = processImage(tmp.Name()); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadImage, err)
		}
		ext = ".webp"
	}
	var rnd [8]byte
	_, _ = rand.Read(rnd[:])
	name := hex.EncodeToString(rnd[:]) + "__" + base + ext

	// Видео → WebM, Office → PDF: ард хөрвүүлнэ. Хариунд эцсийн замыг шууд өгнө.
	if target := s.tc.targetExt(ext, officePDF); target != "" {
		final := hex.EncodeToString(rnd[:]) + "__" + base + target
		src := filepath.Join(vdir, processingPrefix+name)
		if err := os.Rename(tmp.Name(), src); err != nil {
			return nil, err
		}
		fi, err := os.Stat(src)
		if err != nil {
			return nil, err
		}
		select {
		case s.tc.jobs <- job{src: src, dst: filepath.Join(vdir, final)}:
		default:
			// Дараалал дүүрсэн — сервер дахин эхлэхэд glob-оор автоматаар орно.
		}
		info := s.info(teacherID, visibility, fi)
		info.Name, info.OriginalName, info.Status = final, base+target, StatusProcessing
		info.Path = "/files/" + teacherID + "/" + visibility + "/" + url.PathEscape(final)
		info.URL, info.ContentType = s.Resolve(info.Path, 6*time.Hour), mime.TypeByExtension(target)
		return info, nil
	}

	final := filepath.Join(vdir, name)
	if err := os.Rename(tmp.Name(), final); err != nil {
		return nil, err
	}
	fi, err := os.Stat(final)
	if err != nil {
		return nil, err
	}
	return s.info(teacherID, visibility, fi), nil
}

const (
	StatusReady      = "ready"
	StatusProcessing = "processing"
	StatusFailed     = "failed"
)

func (s *Store) info(teacherID, visibility string, fi fs.FileInfo) *Info {
	name, status := fi.Name(), StatusReady
	switch {
	case strings.HasPrefix(name, processingPrefix):
		name, status = filepath.Base(dstFor(name)), StatusProcessing
	case strings.HasPrefix(name, failedPrefix):
		name, status = strings.TrimPrefix(name, failedPrefix), StatusFailed
	}
	orig := name
	if _, after, ok := strings.Cut(name, "__"); ok {
		orig = after
	}
	path := "/files/" + teacherID + "/" + visibility + "/" + url.PathEscape(name)
	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if ct == "" {
		ct = "application/octet-stream"
	}
	info := &Info{Name: name, OriginalName: orig, Visibility: visibility, Size: fi.Size(), ContentType: ct,
		Path: path, URL: s.Resolve(path, 6*time.Hour), ModTime: fi.ModTime(), Status: status}
	if parts := s.Parts(path); len(parts) > 1 { // хуваагдсан видео: нийт хэмжээ, хэсгийн тоо
		info.Parts = len(parts)
		for _, p := range parts[1:] {
			if _, vis, n, ok := SplitPath(p); ok {
				if st, err := os.Stat(filepath.Join(s.root, "teachers", teacherID, vis, n)); err == nil {
					info.Size += st.Size()
				}
			}
		}
	}
	return info
}

func (s *Store) List(teacherID string) ([]Info, error) {
	dir, err := s.teacherDir(teacherID)
	if err != nil {
		return nil, err
	}
	out := []Info{}
	for _, v := range []string{Public, Private} {
		entries, err := os.ReadDir(filepath.Join(dir, v))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || (strings.HasPrefix(n, ".") && !strings.HasPrefix(n, processingPrefix) && !strings.HasPrefix(n, failedPrefix)) || partRe.MatchString(n) {
				continue // ".p02.webm" зэрэг видеоны дараагийн хэсгүүд эхний хэсгийнхээ доор
			}
			if fi, err := e.Info(); err == nil {
				out = append(out, *s.info(teacherID, v, fi))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, nil
}

// Open нь зам шалгаад файлыг нээнэ (path traversal-аас хамгаална).
func (s *Store) Open(teacherID, visibility, name string) (*os.File, fs.FileInfo, error) {
	if visibility != Public && visibility != Private {
		return nil, nil, ErrNotFound
	}
	if !fileNameRe.MatchString(name) {
		return nil, nil, ErrNotFound
	}
	dir, err := s.teacherDir(teacherID)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	f, err := os.Open(filepath.Join(dir, visibility, name))
	if err != nil {
		return nil, nil, ErrNotFound
	}
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		f.Close()
		return nil, nil, ErrNotFound
	}
	return f, fi, nil
}

func (s *Store) Delete(teacherID, visibility, name string) error {
	if (visibility != Public && visibility != Private) || !fileNameRe.MatchString(name) {
		return ErrNotFound
	}
	dir, err := s.teacherDir(teacherID)
	if err != nil {
		return ErrNotFound
	}
	vdir := filepath.Join(dir, visibility)
	removed := false
	if b, err := os.ReadFile(partsFile(filepath.Join(vdir, name))); err == nil { // хуваагдсан видео: бүх хэсэг
		var names []string
		if json.Unmarshal(b, &names) == nil {
			for _, n := range names {
				if n != name && fileNameRe.MatchString(n) {
					_ = os.Remove(filepath.Join(vdir, n))
				}
			}
		}
		_ = os.Remove(partsFile(filepath.Join(vdir, name)))
	}
	if os.Remove(filepath.Join(vdir, name)) == nil {
		removed = true
	}
	if os.Remove(filepath.Join(vdir, failedPrefix+name)) == nil {
		removed = true
	}
	// Хөрвүүлж буй эх файлууд (өргөтгөл нь өөр байж болно).
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if ms, _ := filepath.Glob(filepath.Join(vdir, processingPrefix+globEscape(stem)+".*")); len(ms) > 0 {
		for _, m := range ms {
			if os.Remove(m) == nil {
				removed = true
			}
		}
	}
	if !removed {
		return ErrNotFound
	}
	return nil
}

func globEscape(s string) string {
	r := strings.NewReplacer(`*`, `\*`, `?`, `\?`, `[`, `\[`, `\`, `\\`)
	return r.Replace(s)
}

// ---- гарын үсэгтэй URL ----

func (s *Store) sig(path string, exp int64) string {
	mac := hmac.New(sha256.New, append([]byte("files:"), s.key...))
	fmt.Fprintf(mac, "%s|%d", path, exp)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:18])
}

// Resolve нь дотоод /files/.../private/... замыг хугацаатай гарын үсэгтэй URL болгоно.
// Бусад (public, гадаад https://...) URL-ыг хэвээр нь буцаана.
func (s *Store) Resolve(path string, ttl time.Duration) string {
	if !strings.HasPrefix(path, "/files/") || !strings.Contains(path, "/"+Private+"/") {
		return path
	}
	exp := stableExp(ttl)
	return path + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.sig(path, exp)
}

// Verify нь private файлын URL-ын гарын үсгийг шалгана. escapedPath нь URL дээрх зам.
func (s *Store) Verify(escapedPath, expStr, sig string) bool {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.sig(escapedPath, exp)))
}

// OwnedBy нь /files/<teacherID>/... зам тухайн багшийнх мөн эсэхийг шалгана.
func OwnedBy(path, teacherID string) bool {
	return strings.HasPrefix(path, "/files/"+teacherID+"/")
}

// ---- медиа тасалбар (видео, аудионы бодит замыг нуух) ----

// Ticket нь private файлын замыг нууж, хугацаатай, хэрэглэгчид холбосон тасалбар үүсгэнэ:
// base64url("path|uid|exp") + "." + HMAC. Суралцагчийн хөтөч файлын нэр, багшийн ID-г харахгүй.
func (s *Store) Ticket(path, uid string, ttl time.Duration) string {
	exp := stableExp(ttl)
	payload := fmt.Sprintf("%s|%s|%d", ticketRef(path), uid, exp)
	mac := hmac.New(sha256.New, append([]byte("ticket:"), s.key...))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:18])
}

// ResolveTicket нь тасалбарыг шалгаад (гарын үсэг, хугацаа) файлын зам ба хэрэглэгчийн ID-г буцаана.
func (s *Store) ResolveTicket(t string) (path, uid string, ok bool) {
	i := strings.LastIndexByte(t, '.')
	if i <= 0 {
		return "", "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(t[:i])
	if err != nil {
		return "", "", false
	}
	mac := hmac.New(sha256.New, append([]byte("ticket:"), s.key...))
	mac.Write(raw)
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:18])
	if !hmac.Equal([]byte(want), []byte(t[i+1:])) {
		return "", "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", "", false
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", "", false
	}
	if strings.HasPrefix(parts[0], "@") {
		p, ok := s.resolveRef(parts[0])
		return p, parts[1], ok
	}
	return parts[0], parts[1], true
}

var hex16Re = regexp.MustCompile(`^[0-9a-f]{16}$`)

// ticketRef — дотоод файлыг тасалбарт богино лавлагаагаар бичнэ: "@<багш>/<visibility>/<16 hex>/<нэрийн хэш>".
// Кирилл нэр кодлогдохоороо 6 дахин уртасч тасалбар 400+ тэмдэгт болдог бол IIS (http.sys) 260-аас урт
// URL хэсгийг «400 Bad Request»-ээр татгалздаг.
func ticketRef(path string) string {
	t, v, n, ok := SplitPath(path)
	if !ok || len(n) < 18 || n[16:18] != "__" || !hex16Re.MatchString(n[:16]) {
		return path
	}
	sum := sha256.Sum256([]byte(n))
	return "@" + t + "/" + v + "/" + n[:16] + "/" + hex.EncodeToString(sum[:4])
}

// resolveRef — богино лавлагаагаас файлын замыг олно (16 hex угтвартай файлуудаас нэрийн хэшээр).
func (s *Store) resolveRef(ref string) (string, bool) {
	if v, ok := s.refs.Load(ref); ok {
		h := v.(refHit)
		if _, err := os.Stat(h.disk); err == nil {
			return h.path, true
		}
		s.refs.Delete(ref) // файл устсан/солигдсон
	}
	parts := strings.Split(strings.TrimPrefix(ref, "@"), "/")
	if len(parts) != 4 || (parts[1] != Public && parts[1] != Private) || !hex16Re.MatchString(parts[2]) {
		return "", false
	}
	dir, err := s.teacherDir(parts[0])
	if err != nil {
		return "", false
	}
	ms, _ := filepath.Glob(filepath.Join(dir, parts[1], parts[2]+"__*"))
	for _, m := range ms {
		n := filepath.Base(m)
		if sum := sha256.Sum256([]byte(n)); hex.EncodeToString(sum[:4]) == parts[3] {
			p := "/files/" + parts[0] + "/" + parts[1] + "/" + url.PathEscape(n)
			s.refs.Store(ref, refHit{path: p, disk: m})
			return p, true
		}
	}
	return "", false
}

// SplitPath нь /files/<teacher>/<visibility>/<name> замыг задална (name нь URL-escape-гүй).
func SplitPath(p string) (teacher, visibility, name string, ok bool) {
	parts := strings.Split(strings.TrimPrefix(p, "/files/"), "/")
	if !strings.HasPrefix(p, "/files/") || len(parts) != 3 {
		return "", "", "", false
	}
	n, err := url.PathUnescape(parts[2])
	if err != nil {
		return "", "", "", false
	}
	return parts[0], parts[1], n, true
}
