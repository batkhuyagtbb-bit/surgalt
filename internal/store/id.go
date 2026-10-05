package store

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"sync/atomic"
	"time"
)

// NewID нь 24 тэмдэгттэй hex ID үүсгэнэ (ObjectID маягийн бүтэц:
// 4 байт секунд + 5 байт санамсаргүй + 3 байт тоолуур). Цаг хугацааны дарааллаар
// эрэмбэлэгддэг тул "id < beforeID" гэх мэт хуудаслалтад шууд ашиглана; files
// багцын ID шалгалт (^[0-9a-f]{24}$) хэвээр хүчинтэй.
func NewID() string {
	var b [12]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))
	copy(b[4:9], idRand[:])
	n := idCounter.Add(1)
	b[9], b[10], b[11] = byte(n>>16), byte(n>>8), byte(n)
	return hex.EncodeToString(b[:])
}

var (
	idRand    [5]byte
	idCounter atomic.Uint32
)

func init() {
	if _, err := rand.Read(idRand[:]); err != nil {
		panic("id: random source unavailable: " + err.Error())
	}
	var c [4]byte
	_, _ = rand.Read(c[:])
	idCounter.Store(binary.BigEndian.Uint32(c[:]))
}
