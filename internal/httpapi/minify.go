package httpapi

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// ---- Static JS/CSS-ийг ачаалахдаа нэг удаа minify хийнэ (эх код репод уншигдахуйц хэвээр) ----
// JS: хоосон зай, тайлбар, дотоод нэрсийг богиносгоно (дээд түвшний нэрс — app.js, studio.js хооронд хуваалцдаг
// глобалууд хэвээр). CSS: зөвхөн хоосон зай, тайлбар. vendor/ (аль хэдийн min) хөндөхгүй. Алдаа гарвал эх файл.

func minifiable(name string) bool {
	if strings.HasPrefix(name, "vendor/") || strings.Contains(name, ".min.") {
		return false
	}
	ext := path.Ext(name)
	return ext == ".js" || ext == ".css"
}

func minifyAsset(name string, raw []byte) ([]byte, error) {
	opts := api.TransformOptions{
		Sourcefile:       name,
		MinifyWhitespace: true,
		Charset:          api.CharsetUTF8, // кирилл үсгийг \uXXXX болгохгүй
		LegalComments:    api.LegalCommentsNone,
		LogLevel:         api.LogLevelSilent,
	}
	if path.Ext(name) == ".css" {
		opts.Loader = api.LoaderCSS
	} else {
		opts.Loader = api.LoaderJS
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	}
	res := api.Transform(string(raw), opts)
	if len(res.Errors) > 0 {
		e := res.Errors[0]
		if e.Location != nil {
			return nil, &minifyErr{name: name, msg: e.Text, line: e.Location.Line}
		}
		return nil, &minifyErr{name: name, msg: e.Text}
	}
	return res.Code, nil
}

type minifyErr struct {
	name, msg string
	line      int
}

func (e *minifyErr) Error() string { return e.name + ":" + strconv.Itoa(e.line) + ": " + e.msg }

// minFS — fs.FS, JS/CSS-ийг minify хийсэн хувилбараар өгнө (http.FileServerFS, gzStatic хоёулаа үүнийг уншина).
type minFS struct {
	base fs.FS
	data map[string][]byte
}

func newMinFS(base fs.FS, log *slog.Logger) fs.FS {
	m := &minFS{base: base, data: map[string][]byte{}}
	start, before, after := time.Now(), 0, 0
	_ = fs.WalkDir(base, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !minifiable(name) {
			return err
		}
		raw, err := fs.ReadFile(base, name)
		if err != nil {
			return nil
		}
		out, err := minifyAsset(name, raw)
		if err != nil {
			if log != nil {
				log.Warn("minify хийж чадсангүй — эх файлыг өгнө", "file", name, "err", err)
			}
			return nil
		}
		m.data[name] = out
		before, after = before+len(raw), after+len(out)
		return nil
	})
	if log != nil {
		log.Info("static minify", "files", len(m.data), "kb", before>>10, "min_kb", after>>10, "d", time.Since(start).Round(time.Millisecond))
	}
	return m
}

func (m *minFS) Open(name string) (fs.File, error) {
	f, err := m.base.Open(name)
	if err != nil {
		return nil, err
	}
	data, ok := m.data[name]
	if !ok {
		return f, nil
	}
	st, err := f.Stat()
	f.Close()
	if err != nil {
		return nil, err
	}
	return &memFile{Reader: bytes.NewReader(data), info: memInfo{FileInfo: st, size: int64(len(data))}}, nil
}

type memFile struct {
	*bytes.Reader
	info memInfo
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Close() error               { return nil }

var _ io.ReadSeeker = (*memFile)(nil)

type memInfo struct {
	fs.FileInfo
	size int64
}

func (i memInfo) Size() int64 { return i.size }
