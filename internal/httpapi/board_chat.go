package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// «Цагаан самбар — багаар ажиллах»: багийн чат, оролцооны хэмжүүр, багшийн дуу.
//
// Чатын суваг: "t:<n>" — багийн гишүүд хоорондоо; "b:<n>" — баг ↔ багш (баг бүхэлдээ багштай).
// Багш бүх сувгийг уншиж, бичнэ; "all" — бүх багийн "b:" сувагт нэг дор. Хадгалалт: board_rooms-д
// "_c:<суваг>" мөр (сүүлийн boardChatKeep мессеж).
//
// Хэмжүүр ("_m" мөр): суралцагч × өдөр бүрт [самбар дээр байсан сек, идэвхтэй сек, засвар, чат].
// Идэвхтэй хугацаа: засвар/чатын хооронд boardActGap-аас бага зай бол тэр хугацааг бүхэлд нь тооцно.
const (
	boardChatPrefix  = "_c:"
	boardMetricsRoom = "_m"
	boardChatKeep    = 300
	boardChatMaxLen  = 1000
	boardActGap      = 60 * time.Second
	boardActMin      = 5 // ганц засварын доод тооцоо (сек)
	boardVoiceMax    = 16 << 10
	boardMetricUsers = 1000
)

type boardMsg struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
	T    bool   `json:"t,omitempty"` // багш
	Body string `json:"body"`
	At   int64  `json:"at"` // unix мс
}

type boardChat struct {
	Msgs  []boardMsg `json:"msgs"`
	dirty bool
}

// boardUserMetric — нэг суралцагчийн нэг самбар дахь оролцоо.
type boardUserMetric struct {
	Name string             `json:"n"`
	Team int                `json:"t"` // сүүлд байсан баг (-1 = багт ороогүй)
	Days map[string]*[5]int `json:"d"` // "2006-01-02" → [байсан сек, идэвхтэй сек, засвар, чат, ярьсан мс]
}

type boardMetrics struct {
	U map[string]*boardUserMetric `json:"u"`
}

// validChatCh — "t:<n>" | "b:<n>" (n = 0..11).
func validChatCh(ch string) (int, bool) {
	k, n, ok := strings.Cut(ch, ":")
	if !ok || (k != "t" && k != "b") {
		return 0, false
	}
	i, err := strconv.Atoi(n)
	if err != nil || i < 0 || i >= boardMaxTeams || strconv.Itoa(i) != n {
		return 0, false
	}
	return i, true
}

// loadExtra — "_c:" (чат), "_m" (хэмжүүр) мөрийг ачаална; эдгээрийн нэг бол true.
func (bs *boardSess) loadExtra(room string, data []byte) bool {
	if room == boardMetricsRoom {
		_ = json.Unmarshal(data, &bs.met)
		return true
	}
	ch, ok := strings.CutPrefix(room, boardChatPrefix)
	if !ok {
		return false
	}
	if _, ok := validChatCh(ch); ok {
		var c boardChat
		if json.Unmarshal(data, &c) == nil {
			bs.chats[ch] = &c
		}
	}
	return true
}

// metric — суралцагчийн өнөөдрийн мөр ([байсан, идэвхтэй, засвар, чат, ярьсан мс]); өөрчилнө гэж үзээд dirty болгоно.
func (bs *boardSess) metric(uid, name string, now time.Time) *[5]int {
	if bs.met.U == nil {
		bs.met.U = map[string]*boardUserMetric{}
	}
	m := bs.met.U[uid]
	if m == nil {
		if len(bs.met.U) >= boardMetricUsers {
			return new([5]int) // хэт олон — тоолохгүй
		}
		m = &boardUserMetric{Team: -1, Days: map[string]*[5]int{}}
		bs.met.U[uid] = m
	}
	if name != "" {
		m.Name = name
	}
	if bs.state.Mode == "team" {
		if n, ok := bs.state.Assign[uid]; ok && n < bs.state.Teams {
			m.Team = n
		}
	}
	day := now.Format("2006-01-02")
	d := m.Days[day]
	if d == nil {
		d = new([5]int)
		m.Days[day] = d
	}
	bs.metDirty = true
	return d
}

// active — засвар/чат: идэвхтэй хугацааг нэмээд өнөөдрийн мөрийг буцаана.
func (bs *boardSess) active(uid, name string, now time.Time) *[5]int {
	d := bs.metric(uid, name, now)
	if last, ok := bs.act[uid]; ok && now.Sub(last) <= boardActGap {
		d[1] += int(now.Sub(last).Seconds() + 0.5)
	} else {
		d[1] += boardActMin
	}
	bs.act[uid] = now
	return d
}

// chatTeam — хэрэглэгч энэ сувгийг унших эрхтэй бол багийн дугаар.
func (bs *boardSess) canChat(uid string, teacher bool, ch string) bool {
	n, ok := validChatCh(ch)
	if !ok {
		return false
	}
	if teacher {
		return true
	}
	t, ok := bs.state.Assign[uid]
	return ok && t == n && t < bs.state.Teams
}

// chatViewers — сувгийг харж буй хүмүүс: тухайн багийн гишүүд ба багш (самбар дээр байгаа).
func (bs *boardSess) chatViewers(ch string, now time.Time) []string {
	n, _ := validChatCh(ch)
	var ids []string
	for uid, p := range bs.peers {
		if now.Sub(p.at) > boardPeerTTL {
			continue
		}
		if t, ok := bs.state.Assign[uid]; p.Teacher || (ok && t == n) {
			ids = append(ids, uid)
		}
	}
	return ids
}

func (bs *boardSess) chat(ch string) *boardChat {
	c := bs.chats[ch]
	if c == nil {
		c = &boardChat{}
		bs.chats[ch] = c
	}
	return c
}

// GET .../board/{bid}/chat — {team, chats: {суваг: [мессеж]}}: суралцагчид өөрийн багийн хоёр суваг, багшид бүгд.
func (s *Server) handleBoardChat(w http.ResponseWriter, r *http.Request) {
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return
	}
	bs, uid := br.s, br.c.UID
	out := map[string][]boardMsg{}
	team := -1
	if br.teacher {
		for ch, c := range bs.chats {
			if n, _ := validChatCh(ch); n < bs.state.Teams && len(c.Msgs) > 0 {
				out[ch] = c.Msgs
			}
		}
	} else if n, ok := bs.state.Assign[uid]; ok && n < bs.state.Teams {
		team = n
		for _, k := range []string{"t:", "b:"} {
			ch := k + strconv.Itoa(n)
			out[ch] = []boardMsg{}
			if c := bs.chats[ch]; c != nil {
				out[ch] = c.Msgs
			}
		}
	}
	writeLocked(w, bs, map[string]any{"team": team, "chats": out})
}

// POST .../board/{bid}/chat — {ch, body}: мессеж илгээх; багш ch="all" бол бүх багт; {ch, del} — багш мессеж устгана.
func (s *Server) handleBoardChatPost(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Ch   string `json:"ch"`
		Body string `json:"body"`
		Del  string `json:"del"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Del == "" {
		if n := utf8.RuneCountInString(in.Body); n < 1 || n > boardChatMaxLen {
			writeErr(w, http.StatusBadRequest, "мессеж 1-1000 тэмдэгт")
			return
		}
		if !s.chatLim.Allow("bd:" + c.UID) {
			writeErr(w, http.StatusTooManyRequests, "хэт хурдан бичиж байна")
			return
		}
	}
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return
	}
	bs, now := br.s, time.Now()
	if in.Del != "" {
		if !br.teacher {
			errLocked(w, bs, http.StatusForbidden, "зөвхөн багш устгана")
			return
		}
		ch := bs.chats[in.Ch]
		i := -1
		if ch != nil {
			i = slices.IndexFunc(ch.Msgs, func(m boardMsg) bool { return m.ID == in.Del })
		}
		if i < 0 {
			errLocked(w, bs, http.StatusNotFound, "мессеж олдсонгүй")
			return
		}
		ch.Msgs = slices.Delete(ch.Msgs, i, i+1)
		ch.dirty = true
		s.boardSend(bs, bs.chatViewers(in.Ch, now), boardEvent{Ev: "chat", Room: in.Ch, Del: in.Del})
		writeLocked(w, bs, map[string]any{"ok": true})
		return
	}
	targets := []string{in.Ch}
	if in.Ch == "all" && br.teacher {
		targets = targets[:0]
		for i := range bs.state.Teams {
			targets = append(targets, "b:"+strconv.Itoa(i))
		}
	} else if !bs.canChat(br.c.UID, br.teacher, in.Ch) {
		errLocked(w, bs, http.StatusForbidden, "энэ багийн чатад бичих эрхгүй")
		return
	}
	if !br.teacher {
		if bs.state.Mode != "team" {
			errLocked(w, bs, http.StatusForbidden, "багийн чат «Багаар» горимд нээлттэй")
			return
		}
		if bs.state.Lock {
			errLocked(w, bs, http.StatusForbidden, "багш самбарыг түр зогсоосон байна")
			return
		}
		bs.active(br.c.UID, br.c.Name, now)[3]++
	}
	var sent []boardMsg
	for _, ch := range targets {
		m := boardMsg{ID: store.NewID(), UID: br.c.UID, Name: br.c.Name, T: br.teacher, Body: in.Body, At: now.UnixMilli()}
		cc := bs.chat(ch)
		cc.Msgs = append(cc.Msgs, m)
		if len(cc.Msgs) > boardChatKeep {
			cc.Msgs = slices.Clone(cc.Msgs[len(cc.Msgs)-boardChatKeep:])
		}
		cc.dirty = true
		s.boardSend(bs, bs.chatViewers(ch, now), boardEvent{Ev: "chat", Room: ch, Msg: &m})
		sent = append(sent, m)
	}
	if bs.touchPeer(br.c, br.teacher, bs.peerRoom(br.c.UID, br.teacher, now), now) {
		s.boardSendPeers(bs, now)
	}
	writeLocked(w, bs, map[string]any{"msgs": sent})
}

// ---- дуут өрөө (Clubhouse маягийн): багш ба суралцагчид дуугаар ----
//
// "all" — бүх анги (тайз): багш, мөн багш «ярих эрх» өгсөн суралцагчид (state.Stage) ярина, бүгд сонсоно.
// "t:<n>" — багийн дуут өрөө («Багаар» горимд): багийн гишүүд хоорондоо; багш тэр багийн самбарыг
// харж байвал сонсоно, багт хандаж ярьж болно.
//
// Серверийн ачааллыг багасгахын тулд дуу WebRTC-ээр шууд (P2P) явна: сервер зөвхөн холболтын SDP-г
// (offer/answer, ICE бүрэн цуглуулсан тул нэг холболтод 2 мессеж) WebSocket-оор дамжуулна. Сүлжээ нь
// шууд холболт зөвшөөрөөгүй (симметр NAT) сонсогчид л ярьж буй хүн 16кГц IMA ADPCM хэсгийг (~200мс)
// серверээр дамжуулна ("only" — тэр сонсогчид). Ярьсан хугацааг клиент 10 сек тутам мэдээлнэ (talk).

type boardVoice struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	To   string `json:"to"`
	Seq  int    `json:"seq"`
	D    string `json:"d"`
}

// boardRTC — WebRTC холболтын дохио (ярьж буй хүн offer, сонсогч answer илгээнэ).
type boardRTC struct {
	From string `json:"from"`
	T    string `json:"t"` // offer | answer | bye
	ID   string `json:"id"`
	SDP  string `json:"sdp,omitempty"`
}

// canSpeak — энэ хэрэглэгч to руу ярих эрхтэй эсэх.
func (bs *boardSess) canSpeak(uid string, teacher bool, to string) bool {
	if to == "all" {
		return teacher || slices.Contains(bs.state.Stage, uid)
	}
	k, _, _ := strings.Cut(to, ":")
	n, ok := validChatCh(to)
	if k != "t" || !ok || n >= bs.state.Teams {
		return false
	}
	if teacher {
		return true
	}
	t, ok := bs.state.Assign[uid]
	return bs.state.Mode == "team" && !bs.state.Lock && ok && t == n
}

// voiceListeners — дууг сонсох хүмүүс (ярьж буй хүнээс бусад).
func (bs *boardSess) voiceListeners(sender, to string, now time.Time) []string {
	n, _ := validChatCh(to)
	var ids []string
	for uid, p := range bs.peers {
		if uid == sender || now.Sub(p.at) > boardPeerTTL {
			continue
		}
		if to == "all" {
			ids = append(ids, uid)
		} else if t, ok := bs.state.Assign[uid]; (p.Teacher && p.Room == to) || (!p.Teacher && ok && t == n) {
			ids = append(ids, uid)
		}
	}
	return ids
}

// voiceLinked — a, b хоёрын аль нэг нь нөгөөдөө ярьж болох уу (WebRTC дохиог дамжуулах эрх).
func (bs *boardSess) voiceLinked(a, b string, now time.Time) bool {
	pa, pb := bs.peers[a], bs.peers[b]
	if a == b || pa == nil || pb == nil || now.Sub(pb.at) > boardPeerTTL {
		return false
	}
	if pa.Teacher || pb.Teacher || slices.Contains(bs.state.Stage, a) || slices.Contains(bs.state.Stage, b) {
		return true
	}
	ta, oka := bs.state.Assign[a]
	tb, okb := bs.state.Assign[b]
	return bs.state.Mode == "team" && oka && okb && ta == tb && ta < bs.state.Teams
}

// POST .../board/{bid}/voice:
//
//	{sig: {to, t, id, sdp}} — WebRTC дохио;  {to, seq, d, only} — нөөц: дууны хэсэг серверээр;
//	{mic} — микрофон асаасан/унтраасан;  {talk} — сүүлийн мэдээллээс хойш ярьсан мс;  {hand} — суралцагч гар өргөх.
func (s *Server) handleBoardVoice(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.boardLim.Allow("v:" + c.UID) {
		w.Header().Set("Retry-After", "1")
		writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт")
		return
	}
	var in struct {
		To   string   `json:"to"`
		Seq  int      `json:"seq"`
		D    string   `json:"d"`
		Only []string `json:"only"`
		Mic  *bool    `json:"mic"`
		Hand *bool    `json:"hand"`
		Talk int      `json:"talk"`
		Sig  *struct {
			To  string `json:"to"`
			T   string `json:"t"`
			ID  string `json:"id"`
			SDP string `json:"sdp"`
		} `json:"sig"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, boardVoiceMax+1024)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.D) > boardVoiceMax || len(in.Only) > 200 {
		writeErr(w, http.StatusBadRequest, "дууны хэсэг буруу")
		return
	}
	if in.D != "" {
		if raw, err := base64.StdEncoding.DecodeString(in.D); err != nil || len(raw) < 4 {
			writeErr(w, http.StatusBadRequest, "дууны хэсэг буруу")
			return
		}
	}
	if g := in.Sig; g != nil && (!boardUIDRe.MatchString(g.To) || !boardCidRe.MatchString(g.ID) || len(g.SDP) > boardVoiceMax ||
		(g.T != "offer" && g.T != "answer" && g.T != "bye") || (g.T != "bye" && g.SDP == "")) {
		writeErr(w, http.StatusBadRequest, "холболтын дохио буруу")
		return
	}
	br, ok := s.boardOpen(w, r, false)
	if !ok {
		return
	}
	bs, uid, now := br.s, br.c.UID, time.Now()
	if bs.touchPeer(br.c, br.teacher, bs.peerRoom(uid, br.teacher, now), now) {
		s.boardSendPeers(bs, now)
	}
	p := bs.peers[uid]
	switch {
	case in.Sig != nil:
		if !bs.voiceLinked(uid, in.Sig.To, now) {
			errLocked(w, bs, http.StatusForbidden, "энэ хүнтэй дуугаар холбогдох эрхгүй")
			return
		}
		s.boardSend(bs, []string{in.Sig.To}, boardEvent{Ev: "rtc", Rtc: &boardRTC{From: uid, T: in.Sig.T, ID: in.Sig.ID, SDP: in.Sig.SDP}})
		writeLocked(w, bs, map[string]any{"ok": true})
	case in.Hand != nil:
		if br.teacher {
			errLocked(w, bs, http.StatusBadRequest, "багш гар өргөхгүй")
			return
		}
		i := slices.Index(bs.state.Hands, uid)
		if *in.Hand && i < 0 && !slices.Contains(bs.state.Stage, uid) {
			bs.state.Hands = append(bs.state.Hands, uid)
		} else if !*in.Hand && i >= 0 {
			bs.state.Hands = slices.Delete(bs.state.Hands, i, i+1)
		} else {
			writeLocked(w, bs, map[string]any{"ok": true})
			return
		}
		bs.state.V++
		bs.stateDirty = true
		s.boardSendState(bs, now)
		writeLocked(w, bs, map[string]any{"ok": true})
	case in.Mic != nil || in.Talk > 0:
		if in.Talk > 0 && !br.teacher { // ярьсан хугацаа: сүүлийн мэдээллээс хойш өнгөрсөн хугацаанаас их байж болохгүй
			last, ok := bs.talk[uid]
			lim := 20 * time.Second
			if ok && now.Sub(last) < lim {
				lim = now.Sub(last)
			}
			bs.talk[uid] = now
			bs.active(uid, br.c.Name, now)[4] += min(in.Talk, int(lim.Milliseconds()))
		}
		if in.Mic != nil && p.Mic != *in.Mic {
			p.Mic = *in.Mic
			s.boardSendPeers(bs, now)
		}
		writeLocked(w, bs, map[string]any{"ok": true})
	case in.D != "":
		if !bs.canSpeak(uid, br.teacher, in.To) {
			errLocked(w, bs, http.StatusForbidden, "одоо ярих эрхгүй байна")
			return
		}
		ids := bs.voiceListeners(uid, in.To, now)
		if len(in.Only) > 0 {
			ids = slices.DeleteFunc(ids, func(id string) bool { return !slices.Contains(in.Only, id) })
		}
		s.boardSend(bs, ids, boardEvent{Ev: "voice", Voice: &boardVoice{UID: uid, Name: br.c.Name, To: in.To, Seq: in.Seq, D: in.D}})
		bs.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		errLocked(w, bs, http.StatusBadRequest, "дууны хэсэг буруу")
	}
}

// addBoardStats — багшийн статистикт самбарын оролцооны хэмжүүрийг ("_m") нэмнэ: since-ээс хойших өдрүүд.
func (s *Server) addBoardStats(ctx context.Context, scope []scopedCourse, teacherID, userID string, since time.Time, get func(uid, name string) *StudentStat) {
	var ids []string
	for _, sc := range scope {
		for _, l := range sc.lessons {
			if slices.ContainsFunc(l.Blocks, func(b store.Block) bool { return b.Type == "board" }) {
				ids = append(ids, l.ID)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.store.BoardRoomsNamed(ctx, ids, boardMetricsRoom)
	if err != nil {
		s.log.Warn("самбарын хэмжүүр уншиж чадсангүй", "err", err)
		return
	}
	from := since.Format("2006-01-02")
	lastDay := map[string]string{}
	for _, row := range rows {
		var m boardMetrics
		if json.Unmarshal(row.Data, &m) != nil {
			continue
		}
		for uid, u := range m.U {
			if uid == teacherID || (userID != "" && uid != userID) {
				continue
			}
			var sum [5]int
			days, last := 0, ""
			for day, d := range u.Days {
				if day < from || d == nil {
					continue
				}
				for i := range sum {
					sum[i] += d[i]
				}
				days++
				last = max(last, day)
			}
			if days == 0 {
				continue
			}
			st := get(uid, u.Name)
			st.BoardDays += days
			st.BoardSec += sum[0]
			st.BoardActiveSec += sum[1]
			st.BoardOps += sum[2]
			st.BoardMsgs += sum[3]
			st.BoardVoiceSec += sum[4] / 1000
			if last > lastDay[uid] {
				lastDay[uid], st.BoardTeam = last, u.Team
			}
		}
	}
}
