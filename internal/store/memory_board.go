package store

import (
	"context"
	"slices"
)

// boardMem — (хичээл, хэсэг, өрөө) → JSON (board.go).
type boardMem map[[3]string][]byte

func (m *Memory) SaveBoardRooms(_ context.Context, rows []BoardRoomData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.boards == nil {
		m.boards = boardMem{}
	}
	for _, r := range rows {
		m.boards[[3]string{r.LessonID, r.BlockID, r.Room}] = slices.Clone(r.Data)
	}
	return nil
}

func (m *Memory) BoardRooms(_ context.Context, lessonID, blockID string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string][]byte{}
	for k, d := range m.boards {
		if k[0] == lessonID && k[1] == blockID {
			out[k[2]] = slices.Clone(d)
		}
	}
	return out, nil
}

func (m *Memory) BoardRoomsNamed(_ context.Context, lessonIDs []string, room string) ([]BoardRoomData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []BoardRoomData
	for k, d := range m.boards {
		if k[2] == room && slices.Contains(lessonIDs, k[0]) {
			out = append(out, BoardRoomData{LessonID: k[0], BlockID: k[1], Room: room, Data: slices.Clone(d)})
		}
	}
	return out, nil
}
