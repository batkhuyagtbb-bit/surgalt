package store

import (
	"context"
	"errors"
	"reflect"
	"time"

	"surgalt/internal/cache"
)

// ---- Уншилтын кэш (ClickHouse) ----
// Хэрэглэгч, сургалт, хичээл, элсэлтийг хүсэлт бүрт алсын ClickHouse-оос (асуулга бүр ~10-16ms, FINAL) уншихгүй:
// хичээл нээх, явц хадгалах, үүлэн үг, тодруулга зэрэг хүсэлт бүр эрхийн шалгалтад 6-12 асуулга явуулдаг байв.
// Бүх бичилт writeUser / writeCourse / writeLesson / enroll-оор дамждаг тул тэнд кэшийг шууд хүчингүй болгоно
// (TTL нь зөвхөн хамгаалалт). Олон хуулбартай (Keeper) үед өөр хуулбарын бичилтийг мэдэхгүй тул кэш унтарна.
// Буцаах утгыг үргэлж гүн хуулбарлана: дуудагч объектоо өөрчилсөн ч кэш болон бусад хүсэлтэд нөлөөлөхгүй.

type readCache struct {
	users    *cache.Cache[User]
	courses  *cache.Cache[Course]
	lessons  *cache.Cache[[]Lesson] // сургалт → хичээлүүд (lessonsWhere-ийн дарааллаар)
	enrolled *cache.Cache[bool]     // зөвхөн «элссэн» (элсэлт цуцлагдахгүй; элсээгүйг кэшлэхгүй — төлсөн даруй нээгдэнэ)
}

const readCacheTTL = 30 * time.Second

func newReadCache() *readCache {
	nf := func(err error) bool { return errors.Is(err, ErrNotFound) }
	return &readCache{
		users:    cache.New[User](readCacheTTL, 2*time.Second, nf),
		courses:  cache.New[Course](readCacheTTL, 2*time.Second, nf),
		lessons:  cache.New[[]Lesson](readCacheTTL, 0, nil),
		enrolled: cache.New[bool](5*time.Minute, 0, nil),
	}
}

// startJanitors — хугацаа нь дууссан бичлэгийг үе үе цэвэрлэнэ (Close хүртэл).
func (rc *readCache) startJanitors(ctx context.Context) {
	go rc.users.Janitor(ctx, time.Minute)
	go rc.courses.Janitor(ctx, time.Minute)
	go rc.lessons.Janitor(ctx, time.Minute)
	go rc.enrolled.Janitor(ctx, time.Minute)
}

func (c *ClickHouse) forgetUser(id string) {
	if c.rc != nil {
		c.rc.users.Delete(id)
	}
}

func (c *ClickHouse) forgetCourse(id string) {
	if c.rc != nil {
		c.rc.courses.Delete(id)
		c.rc.lessons.Delete(id)
	}
}

// forgetAll — олон мөрөнд нөлөөлөх ховор өөрчлөлтөд (бүртгэл устгах г.м.).
func (c *ClickHouse) forgetAll() {
	if c.rc != nil {
		c.rc.users.Clear()
		c.rc.courses.Clear()
		c.rc.lessons.Clear()
		c.rc.enrolled.Clear()
	}
}

// deepCopy — бүтцийн гүн хуулбар (заагч, slice, map, interface-ийг дахин үүсгэнэ; нээлттэй бус талбар хэвээр).
func deepCopy[T any](v T) T {
	var out T
	cloneValue(reflect.ValueOf(&out).Elem(), reflect.ValueOf(v))
	return out
}

func cloneValue(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return
		}
		n := reflect.New(src.Elem().Type())
		cloneValue(n.Elem(), src.Elem())
		dst.Set(n)
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		n := reflect.New(src.Elem().Type()).Elem()
		cloneValue(n, src.Elem())
		dst.Set(n)
	case reflect.Struct:
		dst.Set(src)
		for i := 0; i < src.NumField(); i++ {
			if f := dst.Field(i); f.CanSet() {
				cloneValue(f, src.Field(i))
			}
		}
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		n := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			cloneValue(n.Index(i), src.Index(i))
		}
		dst.Set(n)
	case reflect.Map:
		if src.IsNil() {
			return
		}
		n := reflect.MakeMapWithSize(src.Type(), src.Len())
		it := src.MapRange()
		for it.Next() {
			v := reflect.New(it.Value().Type()).Elem()
			cloneValue(v, it.Value())
			n.SetMapIndex(it.Key(), v)
		}
		dst.Set(n)
	default:
		dst.Set(src)
	}
}
