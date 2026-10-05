package files

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Видеог вэбд зориулсан WebM (VP9 + Opus) болгон ард хөрвүүлнэ.
// Хөрвүүлж байх үед эх файл ".processing-<эцсийн нэр>.<эх өргөтгөл>" нэртэй
// нуугдмал байна; дуусмагц "<эцсийн нэр>.webm" болж, эх нь устана.
// Эцсийн зам upload-ийн хариунд шууд очих тул хичээлд тэр даруй холбож болно.

const (
	processingPrefix = ".processing-"
	failedPrefix     = ".failed-"
)

// isOfficeExt: LibreOffice-оор PDF болгож 3D номоор харуулах баримтууд.
func isOfficeExt(ext string) bool {
	switch ext {
	case ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".odt", ".odp", ".rtf":
		return true
	}
	return false
}

// targetExt нь хөрвүүлэлтийн дараах өргөтгөл ("" бол хөрвүүлэхгүй).
func (t *Transcoder) targetExt(ext string) string {
	switch {
	case t == nil:
		return ""
	case isVideoExt(ext) && t.ffmpeg != "":
		return ".webm"
	case isOfficeExt(ext) && t.soffice != "":
		return ".pdf"
	}
	return ""
}

func isVideoExt(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv", ".avi":
		return true
	}
	return false
}

type job struct{ src, dst string }

// Transcoder нь хязгаарлагдмал тооны ffmpeg процесс ажиллуулна.
type Transcoder struct {
	ffmpeg  string
	soffice string
	jobs    chan job
	log     *slog.Logger
}

// StartTranscoder: ffmpeg (видео → WebM), soffice (Office → PDF). Аль нь олдоогүй
// төрлийг хөрвүүлэхгүй, эх форматаар хадгална.
func (s *Store) StartTranscoder(ctx context.Context, workers int, log *slog.Logger) {
	t := &Transcoder{jobs: make(chan job, 1024), log: log}
	if bin, err := exec.LookPath("ffmpeg"); err == nil {
		t.ffmpeg = bin
	} else {
		log.Warn("ffmpeg олдсонгүй — видеог WebM болгохгүй, эх форматаар хадгална")
	}
	for _, name := range []string{"soffice", "libreoffice"} {
		if bin, err := exec.LookPath(name); err == nil {
			t.soffice = bin
			break
		}
	}
	if t.soffice == "" {
		log.Warn("LibreOffice (soffice) олдсонгүй — Word/PowerPoint-ийг PDF болгохгүй")
	}
	if t.ffmpeg == "" && t.soffice == "" {
		return
	}
	s.tc = t
	for i := 0; i < max(1, workers); i++ {
		go t.worker(ctx)
	}
	// Өмнө нь тасарсан ажлуудыг үргэлжлүүлнэ.
	go func() {
		matches, _ := filepath.Glob(filepath.Join(s.root, "teachers", "*", "*", processingPrefix+"*"))
		for _, m := range matches {
			t.enqueue(m, dstFor(m))
		}
	}()
}

func dstFor(src string) string {
	dir, name := filepath.Split(src)
	name = strings.TrimPrefix(name, processingPrefix)
	ext := strings.ToLower(filepath.Ext(name))
	target := ".webm"
	if isOfficeExt(ext) {
		target = ".pdf"
	}
	return filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+target)
}

func (t *Transcoder) enqueue(src, dst string) { t.jobs <- job{src, dst} }

func (t *Transcoder) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-t.jobs:
			t.run(ctx, j)
		}
	}
}

func (t *Transcoder) run(ctx context.Context, j job) {
	if strings.HasSuffix(j.dst, ".pdf") {
		t.runOffice(ctx, j)
		return
	}
	if t.ffmpeg == "" {
		return
	}
	start := time.Now()
	tmp := j.dst + ".part"
	cctx, cancel := context.WithTimeout(ctx, 3*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(cctx, t.ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", j.src,
		// 1280px-ээс том бол багасгана; VP9 чанарын горим, олон цөм ашиглана.
		"-vf", "scale='min(1280,iw)':-2",
		"-c:v", "libvpx-vp9", "-crf", "33", "-b:v", "0", "-row-mt", "1", "-deadline", "good", "-cpu-used", "4",
		"-c:a", "libopus", "-b:a", "96k",
		"-f", "webm", tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmp)
		t.fail(ctx, j, err, out)
		return
	}
	if err := os.Rename(tmp, j.dst); err != nil {
		t.log.Error("видео rename", "err", err)
		return
	}
	os.Remove(j.src)
	t.log.Info("видео WebM боллоо", "dst", filepath.Base(j.dst), "took", time.Since(start).Round(time.Second))
}

func (t *Transcoder) fail(ctx context.Context, j job, err error, out []byte) {
	if ctx.Err() != nil {
		return // сервер зогсож байна — дараа үргэлжилнэ
	}
	dir, _ := filepath.Split(j.src)
	_ = os.Rename(j.src, filepath.Join(dir, failedPrefix+filepath.Base(j.dst)))
	t.log.Error("хөрвүүлэлт амжилтгүй", "src", j.src, "err", err, "out", string(out))
}

// runOffice нь Word/PowerPoint/Excel-ийг PDF болгоно (3D номоор үзүүлэхэд).
func (t *Transcoder) runOffice(ctx context.Context, j job) {
	if t.soffice == "" {
		return
	}
	start := time.Now()
	work, err := os.MkdirTemp(filepath.Dir(j.src), ".office-")
	if err != nil {
		t.fail(ctx, j, err, nil)
		return
	}
	defer os.RemoveAll(work)
	// soffice нь гаралтын нэрийг оролтоос авдаг тул тусгаарласан хавтсанд хуулна.
	in := filepath.Join(work, "in"+filepath.Ext(j.src))
	if err := os.Link(j.src, in); err != nil {
		t.fail(ctx, j, err, nil)
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, t.soffice, "--headless", "--norestore", "--nolockcheck",
		"-env:UserInstallation=file://"+filepath.Join(work, "profile"),
		"--convert-to", "pdf", "--outdir", work, in)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.fail(ctx, j, err, out)
		return
	}
	if err := os.Rename(filepath.Join(work, "in.pdf"), j.dst); err != nil {
		t.fail(ctx, j, err, out)
		return
	}
	os.Remove(j.src)
	t.log.Info("баримт PDF боллоо", "dst", filepath.Base(j.dst), "took", time.Since(start).Round(time.Millisecond))
}
