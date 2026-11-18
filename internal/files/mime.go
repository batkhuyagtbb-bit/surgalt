package files

import "mime"

// Видео, аудионы MIME төрлийг хост бүрт ижил болгоно. Go-ийн суурь хүснэгтэд эдгээр алга тул Windows дээр
// бүртгэлээс (registry) уншдаг — тэнд ".webm" нь "audio/webm" байдаг тул ffmpeg-ээр WebM болгосон видео бүр
// «аудио» гэж шошгологдож, Content-Type буруу очдог байв. Зөвхөн хичээлийн тоглуулагч (app.js mediaHTML)
// таньдаг өргөтгөлүүд — бусдыг (ogg, opus …) энд нэмбэл гадаад холбоос тасалбартай болж, тоглуулагч танихгүй
// тул «Файл татах» холбоос 403 өгнө.
func init() {
	for ext, typ := range map[string]string{
		".webm": "video/webm", ".mp4": "video/mp4", ".m4v": "video/mp4", ".mov": "video/quicktime",
		".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".wav": "audio/wav",
	} {
		_ = mime.AddExtensionType(ext, typ)
	}
}
