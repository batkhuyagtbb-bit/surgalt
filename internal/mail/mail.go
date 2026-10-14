// Package mail — системийн имэйл (нууц үг сэргээх г.м.) SMTP-ээр илгээнэ.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// Message — нэг захидал: энгийн текст ба HTML хувилбар (хоёулаа UTF-8).
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender — захидал илгээгч (тестэд хуурамчаар солино).
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// SMTP — SMTP серверээр илгээнэ. 465 порт бол шууд TLS, бусад (587 г.м.) STARTTLS (сервер дэмжвэл).
type SMTP struct {
	Host, User, Pass string
	Port             int
	From             mail.Address
}

// NewSMTP — from нь "Нэр <хаяг>" эсвэл зөвхөн хаяг.
func NewSMTP(host string, port int, user, pass, from string) (*SMTP, error) {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM буруу: %w", err)
	}
	return &SMTP{Host: host, Port: port, User: user, Pass: pass, From: *addr}, nil
}

const timeout = 20 * time.Second

func (s *SMTP) Send(ctx context.Context, m Message) error {
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return fmt.Errorf("хүлээн авагч буруу: %w", err)
	}
	raw, err := Build(s.From, *to, m)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	d := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	if s.Port == 465 {
		conn, err = tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: s.Host})
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if s.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
				return err
			}
		}
	}
	if s.User != "" {
		if err := c.Auth(smtp.PlainAuth("", s.User, s.Pass, s.Host)); err != nil {
			return fmt.Errorf("SMTP нэвтрэлт: %w", err)
		}
	}
	if err := c.Mail(s.From.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// Build — MIME захидал (multipart/alternative: текст + HTML, base64).
func Build(from, to mail.Address, m Message) ([]byte, error) {
	if m.Text == "" && m.HTML == "" {
		return nil, errors.New("захидал хоосон")
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range []struct{ typ, content string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		if p.content == "" {
			continue
		}
		pw, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {p.typ + "; charset=UTF-8"},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.StdEncoding.EncodeToString([]byte(p.content))
		for len(enc) > 76 {
			fmt.Fprintf(pw, "%s\r\n", enc[:76])
			enc = enc[76:]
		}
		fmt.Fprintf(pw, "%s\r\n", enc)
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	var id [12]byte
	_, _ = rand.Read(id[:])
	domain := from.Address[strings.LastIndex(from.Address, "@")+1:]
	var out bytes.Buffer
	for _, h := range [][2]string{
		{"From", from.String()},
		{"To", to.String()},
		{"Subject", mime.BEncoding.Encode("UTF-8", m.Subject)},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id[:]) + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "multipart/alternative; boundary=" + mw.Boundary()},
	} {
		fmt.Fprintf(&out, "%s: %s\r\n", h[0], h[1])
	}
	out.WriteString("\r\n")
	out.Write(body.Bytes())
	return out.Bytes(), nil
}
