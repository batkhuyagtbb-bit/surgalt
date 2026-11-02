package store

import "context"

// BoardRoomData — «Цагаан самбар»-ын нэг өрөөний агшин зураг (board.go).
// Өрөө: "main" (багшийнх), "u:<хэрэглэгч>", "t:<баг>", "_s" (ангийн төлөв: горим, багууд).
// Data нь сервер тайлбарлахгүй JSON; хамгийн сүүлд хадгалсан нь хүчинтэй.
type BoardRoomData struct {
	LessonID string
	BlockID  string
	Room     string
	Data     []byte
}

type BoardStore interface {
	// SaveBoardRooms — хэд хэдэн өрөөг нэг дор хадгална (ClickHouse-д нэг INSERT).
	SaveBoardRooms(ctx context.Context, rows []BoardRoomData) error
	// BoardRooms — тухайн хэсгийн бүх өрөө: өрөө → JSON.
	BoardRooms(ctx context.Context, lessonID, blockID string) (map[string][]byte, error)
	// BoardRoomsNamed — өгөгдсөн хичээлүүдийн бүх самбарын тухайн нэртэй өрөө (жишээ нь "_m" — оролцооны хэмжүүр).
	BoardRoomsNamed(ctx context.Context, lessonIDs []string, room string) ([]BoardRoomData, error)
}
