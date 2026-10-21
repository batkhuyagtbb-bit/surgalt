package store

import "context"

// CloudStore — «үүлэн үг» (word cloud) хэсгийн хариултууд: суралцагч бүрийн илгээсэн үгс
// (дахин илгээвэл өмнөхийг солино).
type CloudStore interface {
	SetCloudWords(ctx context.Context, lessonID, blockID, userID string, words []string) error
	// CloudWords — тухайн хэсгийн бүх хариулт: хэрэглэгч → үгс.
	CloudWords(ctx context.Context, lessonID, blockID string) (map[string][]string, error)
}
