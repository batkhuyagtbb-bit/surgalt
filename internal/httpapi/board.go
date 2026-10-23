package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

// «Цагаан самбар — багаар ажиллах» (board): хичээл доторх хамтын зургийн самбар.
//
// Өрөө (room):
//
//	"main"    — багшийн самбар: багш тайлбарлана, анги бүхэлдээ шууд харна;
//	"u:<uid>" — суралцагч бүрийн самбар («Хувиараа» горим);
//	"t:<n>"   — багийн самбар («Багаар» горим, n = 0..11).
//
// Объектууд нь Fabric.js-ийн JSON: сервер зөвхөн id, төрөл, хэмжээ, зургийн эх сурвалжийг шалгаад
// тунгалаг (opaque) хадгална; нэг объектод сүүлд бичсэн нь ялна. Өөрчлөлтийг HTTP-ээр авч, тухайн өрөөг
// харж буй хүмүүст WebSocket-ын хэрэглэгчийн сувгаар түгээнэ; багш бусад өрөөний «шинэчлэгдлээ» дохиог авна.
const (
	boardMaxObjects  = 3000
	boardMaxObjBytes = 400 << 10 // нэг объект (урт зураас)
	boardMaxRoomSize = 8 << 20
	boardMaxBody     = 4 << 20
	boardMaxOps      = 300
	boardMaxTeams    = 12
	boardFxBytes     = 32 << 10         // түр дохио (лазер заагч, зурж буй зураас)
	boardPeerTTL     = 40 * time.Second // «самбар дээр байна» (heartbeat 15 сек)
	boardIdle        = 3 * time.Hour    // санах ойгоос буулгах
	boardGrantTTL    = 2 * time.Minute  // эрхийн шалгалтын кэш (ops, presence)
	boardImageMax    = 15 << 20
	boardSaveEvery   = 4 * time.Second
)

var (
	boardObjIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{4,32}$`)
	boardCidRe   = regexp.MustCompile(`^[a-z0-9]{4,16}$`)
	boardUIDRe   = regexp.MustCompile(`^[0-9a-f]{24}$`)
	boardTypes   = map[string]bool{"path": true, "rect": true, "circle": true, "ellipse": true, "triangle": true, "polygon": true,
		"polyline": true, "line": true, "text": true, "i-text": true, "textbox": true, "sticky": true, "image": true, "group": true}
	boardModes  = map[string]bool{"teach": true, "solo": true, "team": true}
	boardImgSem = make(chan struct{}, 4) // зураг боловсруулах зэрэг ажлын тоо
)

// ---- өрөө ----

type boardRoom struct {
	v     int64
	order []string
	objs  map[string]json.RawMessage
	size  int
	dirty bool
}

func newBoardRoom() *boardRoom { return &boardRoom{objs: map[string]json.RawMessage{}} }

func (r *boardRoom) snapshot() []json.RawMessage {
	out := make([]json.RawMessage, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.objs[id])
	}
	return out
}

func (r *boardRoom) put(id string, o json.RawMessage) {
	if old, ok := r.objs[id]; ok {
		r.size -= len(old)
	} else {
		r.order = append(r.order, id)
	}
	r.objs[id] = o
	r.size += len(o)
}

func (r *boardRoom) remove(id string) {
	o, ok := r.objs[id]
	if !ok {
		return
	}
	r.size -= len(o)
	delete(r.objs, id)
	if i := slices.Index(r.order, id); i >= 0 {
		r.order = slices.Delete(r.order, i, i+1)
	}
}

func (r *boardRoom) move(id string, front bool) {
	i := slices.Index(r.order, id)
	if i < 0 {
		return
	}
	r.order = slices.Delete(r.order, i, i+1)
	if front {
		r.order = append(r.order, id)
	} else {
		r.order = slices.Insert(r.order, 0, id)
	}
}

// copyFrom — багшийн самбарыг (ажлын хуудас) хуулна. RawMessage өөрчлөгддөггүй тул хуваалцаж болно.
func (r *boardRoom) copyFrom(m *boardRoom) {
	r.order, r.objs, r.size = slices.Clone(m.order), maps.Clone(m.objs), m.size
}

// boardLocked — багш «түгжсэн» объект (суралцагч хөдөлгөж, устгаж чадахгүй).
func boardLocked(o json.RawMessage) bool { return bytes.Contains(o, []byte(`"lk":true`)) }

type boardSnap struct {
	V       int64             `json:"v"`
	Objects []json.RawMessage `json:"objects"`
}

// ---- ангийн төлөв ----

type boardState struct {
	Mode   string            `json:"mode"`   // teach | solo | team
	Teams  int               `json:"teams"`  // багийн тоо
	Assign map[string]int    `json:"assign"` // суралцагч → баг (0..Teams-1)
	Names  map[string]string `json:"names"`  // самбарт орж байсан суралцагчид → нэр
	Copy   bool              `json:"copy"`   // шинэ самбар багшийн самбараас (ажлын хуудас) эхэлнэ
	Lock   bool              `json:"lock"`   // суралцагчдыг түр зогсоох («Анхаараарай!»)
	Spot   string            `json:"spot"`   // бүх ангид харуулж буй өрөө
	V      int64             `json:"v"`
}

type boardPeer struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	Room    string `json:"room"`
	Teacher bool   `json:"teacher,omitempty"`
	at      time.Time
}

type boardSess struct {
	mu         sync.Mutex
	key        string
	lessonID   string
	bid        string
	teacherID  string
	state      boardState
	stateDirty bool
	rooms      map[string]*boardRoom
	peers      map[string]*boardPeer
	used       time.Time
	saved      time.Time
	gone       bool // санах ойгоос буулгасан — дахин ачаална
}

type boardGrant struct {
	lessonID, bid, teacherID, title string
	teacher                         bool
	teams                           int
	exp                             time.Time
}

type boardHub struct {
	mu     sync.Mutex
	m      map[string]*boardSess
	grants map[string]boardGrant
}

func validBoardRoom(room string) bool {
	if room == "main" {
		return true
	}
	if uid, ok := strings.CutPrefix(room, "u:"); ok {
		return boardUIDRe.MatchString(uid)
	}
	if n, ok := strings.CutPrefix(room, "t:"); ok {
		i, err := strconv.Atoi(n)
		return err == nil && i >= 0 && i < boardMaxTeams && strconv.Itoa(i) == n
	}
	return false
}

func (bs *boardSess) normalize(defTeams int) {
	st := &bs.state
	if !boardModes[st.Mode] {
		st.Mode = "teach"
	}
	if st.Teams < 2 || st.Teams > boardMaxTeams {
		st.Teams = min(max(defTeams, 2), boardMaxTeams)
	}
	if st.Assign == nil {
		st.Assign = map[string]int{}
	}
	if st.Names == nil {
		st.Names = map[string]string{}
	}
	if st.Spot != "" && !validBoardRoom(st.Spot) {
		st.Spot = ""
	}
}

// ownRoom — суралцагчийн одоогийн горимд бичих өрөө ("" бол бичих өрөөгүй).
func (bs *boardSess) ownRoom(uid string) string {
	switch bs.state.Mode {
	case "solo":
		return "u:" + uid
	case "team":
		if n, ok := bs.state.Assign[uid]; ok && n < bs.state.Teams {
			return "t:" + strconv.Itoa(n)
		}
	}
	return ""
}

func (bs *boardSess) canRead(uid string, teacher bool, room string) bool {
	return teacher || room == "main" || room == "u:"+uid || room == bs.state.Spot || room == bs.ownRoom(uid)
}

func (bs *boardSess) canWrite(uid string, teacher bool, room string) bool {
	if teacher {
		return true
	}
	return !bs.state.Lock && room != "main" && room == bs.ownRoom(uid)
}

func (bs *boardSess) defaultRoom(uid string, teacher bool) string {
	if teacher {
		return "main"
	}
	if bs.state.Spot != "" {
		return bs.state.Spot
	}
	if r := bs.ownRoom(uid); r != "" {
		return r
	}
	return "main"
}

// room — өрөөг буцаана; байхгүй бол үүсгэнэ (Copy үед багшийн самбараас хуулна).
func (bs *boardSess) room(name string) *boardRoom {
	r := bs.rooms[name]
	if r == nil {
		r = newBoardRoom()
		if name != "main" && bs.state.Copy {
			if m := bs.rooms["main"]; m != nil {
				r.copyFrom(m)
				r.dirty = len(r.order) > 0
			}
		}
		bs.rooms[name] = r
	}
	return r
}

func (bs *boardSess) smallestTeam() int {
	cnt := make([]int, bs.state.Teams)
	for _, n := range bs.state.Assign {
		if n >= 0 && n < len(cnt) {
			cnt[n]++
		}
	}
	best := 0
	for i, c := range cnt {
		if c < cnt[best] {
			best = i
		}
	}
	return best
}

func (bs *boardSess) freshPeers(now time.Time) []boardPeer {
	out := make([]boardPeer, 0, len(bs.peers))
	for _, p := range bs.peers {
		if now.Sub(p.at) <= boardPeerTTL {
			out = append(out, *p)
		}
	}
	slices.SortFunc(out, func(a, b boardPeer) int { return strings.Compare(a.Name, b.Name) })
	return out
}

func (bs *boardSess) peerIDs(now time.Time) []string {
	out := make([]string, 0, len(bs.peers))
	for uid, p := range bs.peers {
		if now.Sub(p.at) <= boardPeerTTL {
			out = append(out, uid)
		}
	}
	return out
}

// touchPeer — «самбар дээр байна»; шинэ эсвэл өөр өрөө рүү шилжсэн бол true.
func (bs *boardSess) touchPeer(c auth.Claims, teacher bool, room string, now time.Time) bool {
	p := bs.peers[c.UID]
	if p == nil || now.Sub(p.at) > boardPeerTTL {
		bs.peers[c.UID] = &boardPeer{UID: c.UID, Name: c.Name, Room: room, Teacher: teacher, at: now}
		return true
	}
	p.at = now
	if p.Room != room {
		p.Room = room
		return true
	}
	return false
}

func (bs *boardSess) prunePeers(now time.Time) bool {
	changed := false
	for uid, p := range bs.peers {
		if now.Sub(p.at) > boardPeerTTL {
			delete(bs.peers, uid)
			changed = true
		}
	}
	return changed
}

func (bs *boardSess) roomSummary() []map[string]any {
	out := make([]map[string]any, 0, len(bs.rooms))
	for name, r := range bs.rooms {
		out = append(out, map[string]any{"room": name, "v": r.v, "n": len(r.order)})
	}
	return out
}

// ---- WebSocket-оор түгээх (сессийн түгжээтэй үед дуудна: дараалал алдагдахгүй, publish блоклохгүй) ----

type boardOp struct {
	T  string          `json:"t"` // add | upd | del | front | back | clear | ptr | ink | view
	ID string          `json:"id,omitempty"`
	O  json.RawMessage `json:"o,omitempty"`
}

type boardEvent struct {
	Type  string      `json:"type"` // "board"
	Lid   string      `json:"lid"`
	Bid   string      `json:"bid"`
	Ev    string      `json:"ev"` // ops | fx | touch | state | peers | reload
	Room  string      `json:"room,omitempty"`
	V     int64       `json:"v,omitempty"`
	Cid   string      `json:"cid,omitempty"`
	Ops   []boardOp   `json:"ops,omitempty"`
	State *boardState `json:"state,omitempty"`
	Peers []boardPeer `json:"peers,omitempty"`
	Rooms []string    `json:"rooms,omitempty"`
}

func (s *Server) boardSend(bs *boardSess, ids []string, ev boardEvent) {
	ev.Type, ev.Lid, ev.Bid = "board", bs.lessonID, bs.bid
	s.hub.SendUsers(ids, ev)
}

func (s *Server) boardSendState(bs *boardSess, now time.Time) {
	st := bs.state
	s.boardSend(bs, bs.peerIDs(now), boardEvent{Ev: "state", State: &st})
}

func (s *Server) boardSendPeers(bs *boardSess, now time.Time) {
	peers := bs.freshPeers(now)
	ids := make([]string, len(peers))
	for i, p := range peers {
		ids[i] = p.UID
	}
	s.boardSend(bs, ids, boardEvent{Ev: "peers", Peers: peers})
}

// ---- сесс ачаалах, хадгалах ----

func (s *Server) boardSession(ctx context.Context, g boardGrant) (*boardSess, error) {
	key := g.lessonID + "/" + g.bid
	h := &s.boards
	h.mu.Lock()
	defer h.mu.Unlock()
	if bs := h.m[key]; bs != nil {
		return bs, nil
	}
	raw, err := s.store.BoardRooms(ctx, g.lessonID, g.bid)
	if err != nil {
		return nil, err
	}
	bs := &boardSess{key: key, lessonID: g.lessonID, bid: g.bid, teacherID: g.teacherID,
		state: boardState{Mode: "teach", Copy: true}, rooms: map[string]*boardRoom{}, peers: map[string]*boardPeer{}, used: time.Now()}
	for room, data := range raw {
		if room == "_s" {
			var st boardState
			if json.Unmarshal(data, &st) == nil {
				bs.state = st
			}
			continue
		}
		var snap boardSnap
		if !validBoardRoom(room) || json.Unmarshal(data, &snap) != nil {
			continue
		}
		r := newBoardRoom()
		r.v = snap.V
		for _, o := range snap.Objects {
			var head struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(o, &head) == nil && head.ID != "" {
				r.put(head.ID, o)
			}
		}
		bs.rooms[room] = r
	}
	bs.normalize(g.teams)
	if h.m == nil {
		h.m = map[string]*boardSess{}
	}
	h.m[key] = bs
	return bs, nil
}

func (s *Server) boardLoop(ctx context.Context) {
	t := time.NewTicker(2 * time.Second) // ClickHouse-д секундэд 1-ээс цөөн INSERT
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			fctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			s.flushBoards(fctx, true)
			cancel()
			return
		case <-t.C:
			s.flushBoards(ctx, false)
		}
	}
}

// flushBoards — өөрчлөгдсөн өрөөнүүдийг (boardSaveEvery тутамд) нэг INSERT-ээр хадгалж,
// хуучирсан оролцогчийг хасаж, удаан ашиглаагүй сессийг санах ойгоос буулгана.
func (s *Server) flushBoards(ctx context.Context, final bool) {
	now := time.Now()
	h := &s.boards
	h.mu.Lock()
	sessions := slices.Collect(maps.Values(h.m))
	for k, g := range h.grants {
		if now.After(g.exp) {
			delete(h.grants, k)
		}
	}
	h.mu.Unlock()
	var rows []store.BoardRoomData
	type mark struct {
		bs    *boardSess
		rooms []string
		state bool
	}
	var marks []mark
	for _, bs := range sessions {
		bs.mu.Lock()
		if bs.prunePeers(now) {
			s.boardSendPeers(bs, now)
		}
		if final || now.Sub(bs.saved) >= boardSaveEvery {
			m := mark{bs: bs}
			if bs.stateDirty {
				if data, err := json.Marshal(bs.state); err == nil {
					rows = append(rows, store.BoardRoomData{LessonID: bs.lessonID, BlockID: bs.bid, Room: "_s", Data: data})
					bs.stateDirty, m.state = false, true
				}
			}
			for name, r := range bs.rooms {
				if !r.dirty {
					continue
				}
				if data, err := json.Marshal(boardSnap{V: r.v, Objects: r.snapshot()}); err == nil {
					rows = append(rows, store.BoardRoomData{LessonID: bs.lessonID, BlockID: bs.bid, Room: name, Data: data})
					r.dirty = false
					m.rooms = append(m.rooms, name)
				}
			}
			if m.state || len(m.rooms) > 0 {
				bs.saved = now
				marks = append(marks, m)
			}
		}
		bs.mu.Unlock()
	}
	if len(rows) > 0 {
		if err := s.store.SaveBoardRooms(ctx, rows); err != nil {
			s.log.Warn("цагаан самбар хадгалж чадсангүй", "err", err, "rows", len(rows))
			for _, m := range marks { // дараагийн удаа дахин оролдоно
				m.bs.mu.Lock()
				m.bs.stateDirty = m.bs.stateDirty || m.state
				for _, name := range m.rooms {
					if r := m.bs.rooms[name]; r != nil {
						r.dirty = true
					}
				}
				m.bs.mu.Unlock()
			}
			return
		}
	}
	h.mu.Lock()
	for k, bs := range h.m {
		bs.mu.Lock()
		idle := !bs.stateDirty && len(bs.peers) == 0 && now.Sub(bs.used) > boardIdle
		for _, r := range bs.rooms {
			idle = idle && !r.dirty
		}
		if idle {
			bs.gone = true
			delete(h.m, k)
		}
		bs.mu.Unlock()
	}
	h.mu.Unlock()
}

// ---- хүсэлтийн эрх ----

type boardReq struct {
	s       *boardSess
	c       auth.Claims
	g       boardGrant
	teacher bool
}

// boardOpen — хэрэглэгч хичээлд хандах эрхтэй эсэхийг шалгаж (fresh=false бол 2 минутын кэшээр),
// самбарын сессийг түгжиж буцаана. Дуудагч bs.mu.Unlock() хийнэ.
func (s *Server) boardOpen(w http.ResponseWriter, r *http.Request, fresh bool) (*boardReq, bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return nil, false
	}
	cid, lid, bid := r.PathValue("id"), r.PathValue("lid"), r.PathValue("bid")
	gk := c.UID + "|" + cid + "|" + lid + "|" + bid
	h := &s.boards
	h.mu.Lock()
	g, ok := h.grants[gk]
	h.mu.Unlock()
	if fresh || !ok || time.Now().After(g.exp) {
		course, l, ok := s.lessonForUser(w, r, c.UID, cid, lid)
		if !ok {
			return nil, false
		}
		var b *store.Block
		for i := range l.Blocks {
			if l.Blocks[i].ID == bid && l.Blocks[i].Type == "board" {
				b = &l.Blocks[i]
			}
		}
		if b == nil {
			writeErr(w, http.StatusNotFound, "цагаан самбар олдсонгүй")
			return nil, false
		}
		g = boardGrant{lessonID: l.ID, bid: b.ID, teacherID: course.TeacherID, title: b.Text,
			teacher: course.TeacherID == c.UID, teams: b.Max, exp: time.Now().Add(boardGrantTTL)}
		h.mu.Lock()
		if h.grants == nil {
			h.grants = map[string]boardGrant{}
		}
		h.grants[gk] = g
		h.mu.Unlock()
	}
	for {
		bs, err := s.boardSession(r.Context(), g)
		if s.storeErr(w, r, err) {
			return nil, false
		}
		bs.mu.Lock()
		if bs.gone {
			bs.mu.Unlock()
			continue
		}
		bs.used = time.Now()
		return &boardReq{s: bs, c: c, g: g, teacher: g.teacher}, true
	}
}

// writeLocked — хариуг түгжээн дотор JSON болгож, түгжээг тайлсны дараа илгээнэ (удаан клиент сессийг саатуулахгүй).
func writeLocked(w http.ResponseWriter, bs *boardSess, v any) {
	data, err := json.Marshal(v)
	bs.mu.Unlock()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(append(data, '\n'))
}

func errLocked(w http.ResponseWriter, bs *boardSess, code int, msg string) {
	bs.mu.Unlock()
	writeErr(w, code, msg)
}

// ---- handlers ----

// GET .../board/{bid}?room=&peek=1 — самбарт нэгдэх (эсвэл багш жижиг зураг авах): төлөв ба өрөөний объектууд.
func (s *Server) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	br, ok := s.boardOpen(w, r, r.URL.Query().Get("peek") != "1")
	if !ok {
		return
	}
	bs, uid, now := br.s, br.c.UID, time.Now()
	room, peek := r.URL.Query().Get("room"), r.URL.Query().Get("peek") == "1"
	if !br.teacher && !peek {
		changed := false
		if bs.state.Names[uid] != br.c.Name {
			bs.state.Names[uid], changed = br.c.Name, true
		}
		if bs.state.Mode == "team" {
			if _, ok := bs.state.Assign[uid]; !ok {
				bs.state.Assign[uid], changed = bs.smallestTeam(), true
			}
		}
		if changed {
			bs.state.V++
			bs.stateDirty = true
			s.boardSendState(bs, now)
		}
	}
	if room == "" || !validBoardRoom(room) || !bs.canRead(uid, br.teacher, room) {
		if peek {
			errLocked(w, bs, http.StatusForbidden, "энэ самбарыг харах эрхгүй")
			return
		}
		room = bs.defaultRoom(uid, br.teacher)
	}
	rm := bs.room(room)
	if !peek && bs.touchPeer(br.c, br.teacher, room, now) {
		s.boardSendPeers(bs, now)
	}
	out := map[string]any{"room": room, "v": rm.v, "objects": rm.snapshot()}
	if !peek {
		out["me"] = map[string]any{"uid": uid, "name": br.c.Name, "teacher": br.teacher}
		out["title"] = br.g.title
		out["state"] = bs.state
		out["peers"] = bs.freshPeers(now)
		out["write"] = bs.canWrite(uid, br.teacher, room)
	}
	if br.teacher {
		out["rooms"] = bs.roomSummary()
	}
	writeLocked(w, bs, out)
}

// POST .../board/{bid}/ops — өөрчлөлтийн багц: {room, cid, ops:[{t, id, o}]}.
func (s *Server) handleBoardOps(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.boardLim.Allow(c.UID) {
		w.Header().Set("Retry-After", "1")
		writeErr(w, http.StatusTooManyRequests, "хэт олон өөрчлөлт, түр хүлээнэ үү")
		return
	}
	var in struct {
		Room string    `json:"room"`
		Cid  string    `json:"cid"`
		Ops  []boardOp `json:"ops"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, boardMaxBody)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeErr(w, http.StatusRequestEntityTooLarge, "өөрчлөлт хэт том")
			return
		}
		writeErr(w, http.StatusBadRequest, "JSON буруу")
		return
	}
	if !validBoardRoom(in.Room) || !boardCidRe.MatchString(in.Cid) || len(in.Ops) == 0 || len(in.Ops) > boardMaxOps {
		writeErr(w, http.StatusBadRequest, "самбарын өөрчлөлт буруу")
		return
	}
	var ops, fx []boardOp
	for _, op := range in.Ops {
		switch op.T {
		case "add", "upd":
			id, err := validBoardObj(op.O)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			op.ID = id
			ops = append(ops, op)
		case "del", "front", "back":
			if !boardObjIDRe.MatchString(op.ID) {
				writeErr(w, http.StatusBadRequest, "объектын ID буруу")
				return
			}
			op.O = nil
			ops = append(ops, op)
		case "clear":
			op.ID, op.O = "", nil
			ops = append(ops, op)
		case "ptr", "ink", "view":
			if len(op.O) > boardFxBytes || !boardObjIDRe.MatchString(op.ID) && op.ID != "" {
				writeErr(w, http.StatusBadRequest, "дохио хэт том")
				return
			}
			fx = append(fx, op)
		default:
			writeErr(w, http.StatusBadRequest, "үйлдэл буруу: "+op.T)
			return
		}
	}
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return
	}
	bs, uid, now := br.s, br.c.UID, time.Now()
	if !bs.canWrite(uid, br.teacher, in.Room) {
		msg := "энэ самбарт бичих эрхгүй"
		if bs.state.Lock {
			msg = "багш самбарыг түр зогсоосон байна"
		}
		errLocked(w, bs, http.StatusForbidden, msg)
		return
	}
	rm := bs.room(in.Room)
	full := false
	applied := ops[:0]
	for _, op := range ops {
		old, exists := rm.objs[op.ID]
		if exists && !br.teacher && boardLocked(old) && op.T != "front" && op.T != "back" {
			continue // түгжсэн объект
		}
		switch op.T {
		case "add", "upd":
			if (!exists && len(rm.order) >= boardMaxObjects) || rm.size-len(old)+len(op.O) > boardMaxRoomSize {
				full = true
				continue
			}
			rm.put(op.ID, op.O)
		case "del":
			if !exists {
				continue
			}
			rm.remove(op.ID)
		case "front", "back":
			if !exists {
				continue
			}
			rm.move(op.ID, op.T == "front")
		case "clear":
			for _, id := range slices.Clone(rm.order) {
				if br.teacher || !boardLocked(rm.objs[id]) {
					rm.remove(id)
				}
			}
		}
		applied = append(applied, op)
	}
	if len(applied) > 0 {
		rm.v++
		rm.dirty = true
	}
	if bs.touchPeer(br.c, br.teacher, in.Room, now) {
		s.boardSendPeers(bs, now)
	}
	if len(applied) > 0 || len(fx) > 0 {
		var viewers []string
		teacherAway := false
		for id, p := range bs.peers {
			if now.Sub(p.at) > boardPeerTTL {
				continue
			}
			if p.Room == in.Room {
				viewers = append(viewers, id)
			} else if id == bs.teacherID {
				teacherAway = true
			}
		}
		if len(applied) > 0 {
			s.boardSend(bs, viewers, boardEvent{Ev: "ops", Room: in.Room, V: rm.v, Cid: in.Cid, Ops: applied})
			if teacherAway {
				s.boardSend(bs, []string{bs.teacherID}, boardEvent{Ev: "touch", Room: in.Room, V: rm.v})
			}
		}
		if len(fx) > 0 {
			s.boardSend(bs, viewers, boardEvent{Ev: "fx", Room: in.Room, Cid: in.Cid, Ops: fx})
		}
	}
	writeLocked(w, bs, map[string]any{"v": rm.v, "full": full})
}

// validBoardObj — Fabric объектын JSON-г шалгаж ID-г буцаана. Зураг зөвхөн манай серверээс (/board-img/, /static/).
func validBoardObj(o json.RawMessage) (string, error) {
	if len(o) > boardMaxObjBytes {
		return "", errors.New("объект хэт том (зураасыг богиносгоно уу)")
	}
	var head struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal(o, &head) != nil || !boardObjIDRe.MatchString(head.ID) || !boardTypes[head.Type] {
		return "", errors.New("самбарын объект буруу")
	}
	if bytes.Contains(o, []byte(`"src"`)) || bytes.Contains(o, []byte(`"source"`)) {
		var v any
		if json.Unmarshal(o, &v) != nil || !boardSrcOK(v) {
			return "", errors.New("зургийг самбарт дахин оруулна уу")
		}
	}
	return head.ID, nil
}

func boardSrcOK(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			if s, ok := e.(string); ok && (k == "src" || k == "source") {
				if !boardLocalSrc(s) {
					return false
				}
				continue
			}
			if !boardSrcOK(e) {
				return false
			}
		}
	case []any:
		for _, e := range x {
			if !boardSrcOK(e) {
				return false
			}
		}
	}
	return true
}

func boardLocalSrc(s string) bool {
	return s == "" || ((strings.HasPrefix(s, "/board-img/") || strings.HasPrefix(s, "/static/")) &&
		!strings.Contains(s, "//") && !strings.Contains(s, "..") && !strings.ContainsAny(s, "\\\"<> "))
}

// POST .../board/{bid}/presence — {room, leave}: «самбар дээр байна» (15 сек тутам).
func (s *Server) handleBoardPresence(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Room  string `json:"room"`
		Leave bool   `json:"leave"`
	}
	if !decode(w, r, &in) {
		return
	}
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return
	}
	bs, uid, now := br.s, br.c.UID, time.Now()
	if in.Leave {
		if _, ok := bs.peers[uid]; ok {
			delete(bs.peers, uid)
			s.boardSendPeers(bs, now)
		}
		writeLocked(w, bs, map[string]any{"ok": true})
		return
	}
	room := in.Room
	if !validBoardRoom(room) || !bs.canRead(uid, br.teacher, room) {
		room = bs.defaultRoom(uid, br.teacher)
	}
	if bs.touchPeer(br.c, br.teacher, room, now) {
		s.boardSendPeers(bs, now)
	}
	writeLocked(w, bs, map[string]any{"room": room, "sv": bs.state.V, "v": bs.room(room).v})
}

// POST .../board/{bid}/state — багш: горим, багууд, түр зогсоох, бүх ангид харуулах, ажлын хуудас тараах.
func (s *Server) handleBoardState(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Mode       *string        `json:"mode"`
		Teams      *int           `json:"teams"`
		Assign     map[string]int `json:"assign"` // хэсэгчилсэн: -1 бол багаас гаргана
		Copy       *bool          `json:"copy"`
		Lock       *bool          `json:"lock"`
		Spot       *string        `json:"spot"`
		Shuffle    bool           `json:"shuffle"`    // одоо байгаа суралцагчдыг санамсаргүй хуваах
		Distribute bool           `json:"distribute"` // багшийн самбарыг бүх самбарт хуулах (дарж бичнэ)
	}
	if !decode(w, r, &in) {
		return
	}
	br, ok := s.boardOpen(w, r, true)
	if !ok {
		return
	}
	bs, now := br.s, time.Now()
	if !br.teacher {
		errLocked(w, bs, http.StatusForbidden, "зөвхөн багш удирдана")
		return
	}
	st := &bs.state
	if in.Mode != nil {
		if !boardModes[*in.Mode] {
			errLocked(w, bs, http.StatusBadRequest, "горим буруу")
			return
		}
		st.Mode = *in.Mode
	}
	if in.Teams != nil {
		if *in.Teams < 2 || *in.Teams > boardMaxTeams {
			errLocked(w, bs, http.StatusBadRequest, "багийн тоо 2-12")
			return
		}
		st.Teams = *in.Teams
		for id, n := range st.Assign {
			if n >= st.Teams {
				st.Assign[id] = n % st.Teams
			}
		}
	}
	for id, n := range in.Assign {
		if !boardUIDRe.MatchString(id) {
			errLocked(w, bs, http.StatusBadRequest, "суралцагч буруу")
			return
		}
		if n < 0 || n >= st.Teams {
			delete(st.Assign, id)
		} else {
			st.Assign[id] = n
		}
	}
	if in.Copy != nil {
		st.Copy = *in.Copy
	}
	if in.Lock != nil {
		st.Lock = *in.Lock
	}
	if in.Spot != nil {
		if *in.Spot != "" && !validBoardRoom(*in.Spot) {
			errLocked(w, bs, http.StatusBadRequest, "самбар буруу")
			return
		}
		st.Spot = *in.Spot
	}
	if in.Shuffle {
		var ids []string
		for _, p := range bs.freshPeers(now) {
			if !p.Teacher {
				ids = append(ids, p.UID)
			}
		}
		rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		start := rand.IntN(st.Teams)
		for i, id := range ids {
			st.Assign[id] = (start + i) % st.Teams
		}
	}
	if st.Mode == "team" { // самбар дээр байгаа хуваарилагдаагүй суралцагч
		for _, p := range bs.freshPeers(now) {
			if _, ok := st.Assign[p.UID]; !ok && !p.Teacher {
				st.Assign[p.UID] = bs.smallestTeam()
			}
		}
	}
	var reload []string
	if in.Distribute && st.Mode != "teach" {
		main := bs.room("main")
		var targets []string
		if st.Mode == "solo" {
			for id := range st.Names {
				targets = append(targets, "u:"+id)
			}
		} else {
			for i := range st.Teams {
				targets = append(targets, "t:"+strconv.Itoa(i))
			}
		}
		for _, name := range targets {
			rm := bs.rooms[name]
			if rm == nil {
				rm = newBoardRoom()
				bs.rooms[name] = rm
			}
			rm.copyFrom(main)
			rm.v++
			rm.dirty = true
			reload = append(reload, name)
		}
	}
	st.V++
	bs.stateDirty = true
	s.boardSendState(bs, now)
	if len(reload) > 0 {
		s.boardSend(bs, bs.peerIDs(now), boardEvent{Ev: "reload", Rooms: reload})
	}
	writeLocked(w, bs, map[string]any{"state": bs.state, "rooms": bs.roomSummary()})
}

// boardImageAllowed — зураг оруулах эрх: багш, эсвэл одоогийн горимд өөрийн самбартай суралцагч.
func (s *Server) boardImageAllowed(w http.ResponseWriter, r *http.Request) (*boardReq, bool) {
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return nil, false
	}
	bs := br.s
	if !br.teacher && (bs.state.Lock || bs.ownRoom(br.c.UID) == "") {
		errLocked(w, bs, http.StatusForbidden, "одоо самбарт зураг оруулах боломжгүй")
		return nil, false
	}
	bs.mu.Unlock()
	if !s.boardLim.Allow("img:" + br.c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон зураг, түр хүлээнэ үү")
		return nil, false
	}
	return br, true
}

// POST .../board/{bid}/image — зураг (түүхий бие: PNG/JPEG/WebP/GIF, 15MB хүртэл) → {url, w, h}.
func (s *Server) handleBoardImage(w http.ResponseWriter, r *http.Request) {
	br, ok := s.boardImageAllowed(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, boardImageMax)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusRequestEntityTooLarge, "зураг 15MB-аас ихгүй байна")
		return
	}
	s.boardSaveImage(w, br.g.lessonID, raw)
}

// POST .../board/{bid}/fetch — {url}: интернэтээс (Google зураг г.м) зургийг татаж самбарт хадгална.
func (s *Server) handleBoardFetch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if !decode(w, r, &in) {
		return
	}
	br, ok := s.boardImageAllowed(w, r)
	if !ok {
		return
	}
	u, err := url.Parse(strings.TrimSpace(in.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || len(in.URL) > 4000 {
		writeErr(w, http.StatusBadRequest, "зургийн холбоос буруу")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "зургийн холбоос буруу")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; surgalt-board/1; +https://surgalt.mn)")
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/*;q=0.8")
	res, err := mediaProxy.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "зургийг татаж чадсангүй — зургаа хуулж (Copy image) самбарт буулгана уу")
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		writeErr(w, http.StatusBadGateway, "зургийг татаж чадсангүй ("+strconv.Itoa(res.StatusCode)+") — зургаа хуулж самбарт буулгана уу")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, boardImageMax+1))
	if err != nil || len(raw) > boardImageMax {
		writeErr(w, http.StatusRequestEntityTooLarge, "зураг 15MB-аас ихгүй байна")
		return
	}
	if !strings.HasPrefix(http.DetectContentType(raw), "image/") {
		writeErr(w, http.StatusUnsupportedMediaType, "энэ холбоос зураг биш байна — зураг дээр баруун товч дарж «Copy image address» сонгоно уу")
		return
	}
	s.boardSaveImage(w, br.g.lessonID, raw)
}

func (s *Server) boardSaveImage(w http.ResponseWriter, lessonID string, raw []byte) {
	boardImgSem <- struct{}{}
	name, iw, ih, err := s.files.SaveBoardImage(lessonID, raw)
	<-boardImgSem
	switch {
	case errors.Is(err, files.ErrImage):
		writeErr(w, http.StatusBadRequest, "зургийг уншиж чадсангүй (PNG, JPEG, WebP, GIF)")
	case errors.Is(err, files.ErrBoardFull):
		writeErr(w, http.StatusRequestEntityTooLarge, err.Error())
	case err != nil:
		s.log.Error("самбарын зураг", "err", err)
		writeErr(w, http.StatusInternalServerError, "зургийг хадгалж чадсангүй")
	default:
		writeJSON(w, http.StatusOK, map[string]any{"url": "/board-img/" + lessonID + "/" + name, "w": iw, "h": ih})
	}
}

// GET /board-img/{lid}/{name} — самбарын зураг (таамаглах боломжгүй нэртэй, өөрчлөгддөггүй).
func (s *Server) handleBoardImg(w http.ResponseWriter, r *http.Request) {
	f, fi, err := s.files.OpenBoardImage(r.PathValue("lid"), r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "зураг олдсонгүй")
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "image/webp")
	http.ServeContent(w, r, "", fi.ModTime(), f)
}
