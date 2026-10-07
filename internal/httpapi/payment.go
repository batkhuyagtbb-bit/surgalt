package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash/fnv"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"surgalt/internal/qpay"
	"surgalt/internal/store"
)

// ---- Төлбөр: QR (QPay эсвэл демо), банкны апп-ын холбоос, төлөгдсөнийг шалгаж эрх нээх ----

// paymentInfo — бүх худалдан авалтын хариунд ижил хэлбэртэй төлбөрийн мэдээлэл.
// provider: "qpay" (жинхэнэ QR), "demo" (DEV_PAYMENTS — QR нь демо төлбөрийн хуудас руу), "" (тохируулаагүй).
func (s *Server) paymentInfo(r *http.Request, o *store.Order) map[string]any {
	p := map[string]any{"amount": o.Amount, "currency": "MNT", "dev_pay": s.cfg.DevPayments, "provider": ""}
	if o.Status == store.OrderPaid {
		return p
	}
	switch {
	case s.QPay != nil:
		inv, err := s.qpayInvoice(r, o)
		if err != nil {
			s.log.Warn("qpay нэхэмжлэх", "order", o.ID, "err", err)
			p["error"] = "Төлбөрийн QR үүсгэж чадсангүй. Хэсэг хүлээгээд дахин оролдоно уу."
			return p
		}
		var urls []qpay.BankURL
		_ = json.Unmarshal([]byte(inv.URLs), &urls)
		p["provider"], p["qr"], p["urls"] = "qpay", s.qrDataURI(r.Context(), inv.QRText), urls
	case s.cfg.DevPayments:
		link := s.reqOrigin(r) + "/pay/" + o.ID + "?t=" + s.payMAC(o.ID)
		p["provider"], p["qr"], p["pay_url"] = "demo", s.qrDataURI(r.Context(), link), link
	}
	return p
}

func (s *Server) payMAC(orderID string) string { return s.tokens.MAC("pay", orderID) }

// reqOrigin — хүсэлт ирсэн хаяг (демо QR-ыг утсаар уншуулахад нэг сүлжээнд ажиллана).
// localhost-оор нээсэн бол утсанд ажиллахгүй тул компьютерийн дотоод сүлжээний (Wi-Fi) IP-г тавина.
func (s *Server) reqOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || (s.cfg.TrustProxy && r.Header.Get("X-Forwarded-Proto") == "https") {
		scheme = "https"
	}
	host := r.Host
	if h, port, err := net.SplitHostPort(host); err == nil && (h == "localhost" || h == "::1" || strings.HasPrefix(h, "127.")) {
		if ip := lanIP(); ip != "" {
			host = net.JoinHostPort(ip, port)
		}
	}
	return scheme + "://" + host
}

// lanIP — гадагш гарах сүлжээний интерфейсийн дотоод IP (UDP "холболт" нь пакет илгээхгүй, зөвхөн замыг сонгоно).
// Wi-Fi солигдож болох тул 1 минут л санана.
var lanCache struct {
	sync.Mutex
	ip string
	at time.Time
}

func lanIP() string {
	lanCache.Lock()
	defer lanCache.Unlock()
	if time.Since(lanCache.at) < time.Minute {
		return lanCache.ip
	}
	lanCache.ip, lanCache.at = "", time.Now()
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer c.Close()
	if a, ok := c.LocalAddr().(*net.UDPAddr); ok && a.IP.IsPrivate() {
		lanCache.ip = a.IP.String()
	}
	return lanCache.ip
}

// qrDataURI — QR зургийг data URI болгоно (img-д шууд; токенгүй <img> хүсэлт хэрэггүй).
func (s *Server) qrDataURI(ctx context.Context, text string) string {
	png, err := s.qrs.GetOrLoad(ctx, "pay|"+text, func(context.Context) ([]byte, error) {
		return qrcode.Encode(text, qrcode.Medium, 360)
	})
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// payLocks — захиалга бүрт нэг л нэхэмжлэх үүсгэж, төлбөрийг нэг л удаа биелүүлэх түгжээ (захиалгын ID-гаар хуваасан).
var payLocks [64]sync.Mutex

func payLock(orderID string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(orderID))
	return &payLocks[h.Sum32()%uint32(len(payLocks))]
}

// qpayInvoice — захиалгын QPay нэхэмжлэх (байвал дахин ашиглана; дүн өөрчлөгдсөн бол шинээр).
func (s *Server) qpayInvoice(r *http.Request, o *store.Order) (*store.PaymentInvoice, error) {
	mu := payLock(o.ID)
	mu.Lock()
	defer mu.Unlock()
	sender := o.ID
	old, err := s.store.InvoiceByOrder(r.Context(), o.ID)
	switch {
	case err == nil && old.Provider == "qpay" && old.Amount == o.Amount:
		return old, nil
	case err == nil:
		sender = o.ID + "-" + strconv.FormatInt(time.Now().Unix()%1e6, 36)
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	res, err := s.QPay.CreateInvoice(ctx, qpay.InvoiceRequest{SenderNo: sender, Description: "surgalt.mn — " + o.Title, Amount: o.Amount,
		CallbackURL: s.baseURL(r) + "/api/payments/qpay?o=" + url.QueryEscape(o.ID) + "&t=" + s.payMAC(o.ID)})
	if err != nil {
		return nil, err
	}
	urls, _ := json.Marshal(res.URLs)
	inv := &store.PaymentInvoice{OrderID: o.ID, Provider: "qpay", InvoiceID: res.ID, Amount: o.Amount, QRText: res.QRText, URLs: string(urls)}
	if err := s.store.SaveInvoice(r.Context(), inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// payChecked — QPay-ээс хамгийн сүүлд хэзээ шалгасан (хэт олон дуудахаас сэргийлнэ).
var payChecked = struct {
	sync.Mutex
	at map[string]time.Time
}{at: map[string]time.Time{}}

func payThrottle(orderID string, every time.Duration) bool {
	payChecked.Lock()
	defer payChecked.Unlock()
	now := time.Now()
	if t, ok := payChecked.at[orderID]; ok && now.Sub(t) < every {
		return false
	}
	if len(payChecked.at) > 10000 {
		payChecked.at = map[string]time.Time{}
	}
	payChecked.at[orderID] = now
	return true
}

// syncPayment — хүлээгдэж буй захиалгыг QPay-ээс шалгаж, төлөгдсөн бол биелүүлнэ (эрх нээх, мэдэгдэл).
// Callback-д итгэхгүй: төлбөрийн төлөвийг үргэлж QPay-ээс өөрөөс нь асууна.
func (s *Server) syncPayment(ctx context.Context, o *store.Order, force bool) *store.Order {
	if o.Status == store.OrderPaid || s.QPay == nil || (!force && !payThrottle(o.ID, 3*time.Second)) {
		return o
	}
	inv, err := s.store.InvoiceByOrder(ctx, o.ID)
	if err != nil || inv.Provider != "qpay" {
		return o
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	paid, err := s.QPay.PaidAmount(cctx, inv.InvoiceID)
	if err != nil {
		s.log.Warn("qpay шалгах", "order", o.ID, "err", err)
		return o
	}
	if paid <= 0 || paid < o.Amount {
		return o
	}
	mu := payLock(o.ID)
	mu.Lock()
	defer mu.Unlock()
	if cur, err := s.store.OrderByID(ctx, o.ID); err == nil && cur.Status == store.OrderPaid {
		return cur // зэрэг ирсэн callback аль хэдийн биелүүлсэн
	}
	po, err := s.store.MarkOrderPaid(ctx, o.ID, o.Amount)
	if err != nil {
		s.log.Warn("төлбөр бүртгэх", "order", o.ID, "err", err)
		return o
	}
	s.notifyPaid(ctx, po)
	return po
}

// handleQPayCallback: GET|POST /api/payments/qpay?o=<захиалга>&t=<гарын үсэг> — QPay төлбөр орох үед дуудна.
func (s *Server) handleQPayCallback(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("o")
	if s.QPay == nil || id == "" || !constantEq(r.URL.Query().Get("t"), s.payMAC(id)) {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	o, err := s.store.OrderByID(r.Context(), id)
	if s.storeErr(w, r, err) {
		return
	}
	s.syncPayment(r.Context(), o, true)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("SUCCESS"))
}

// handlePayPage: GET /pay/{id}?t=… — демо төлбөрийн хуудас (DEV_PAYMENTS): QR-ыг утсаар уншуулахад нээгдэнэ.
func (s *Server) handlePayPage(w http.ResponseWriter, r *http.Request) {
	id, t := r.PathValue("id"), r.URL.Query().Get("t")
	if !s.cfg.DevPayments || !constantEq(t, s.payMAC(id)) {
		s.pageError(w, r, store.ErrNotFound)
		return
	}
	o, err := s.store.OrderByID(r.Context(), id)
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "pay.html", map[string]any{"Order": o, "Token": t, "Paid": o.Status == store.OrderPaid}); err != nil {
		s.log.Error("pay template", "err", err)
	}
}
