package mail

import (
	"bufio"
	"context"
	"io"
	"mime"
	"net"
	nmail "net/mail"
	"strings"
	"testing"
	"time"
)

// fakeSMTP — TLS-гүй, нэвтрэлтгүй энгийн SMTP сервер: хүлээн авсан захидлыг буцаана.
func fakeSMTP(t *testing.T) (port int, got <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		r, w := bufio.NewReader(conn), conn
		say := func(s string) { io.WriteString(w, s+"\r\n") }
		say("220 fake ESMTP")
		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
				say("250 OK")
			case cmd == "DATA":
				say("354 go")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					data.WriteString(l)
				}
				say("250 queued")
			case cmd == "QUIT":
				say("221 bye")
				ch <- data.String()
				return
			default:
				say("250 OK")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, ch
}

func TestSMTPSend(t *testing.T) {
	port, got := fakeSMTP(t)
	s, err := NewSMTP("127.0.0.1", port, "", "", "surgalt.mn <no-reply@surgalt.mn>")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send(context.Background(), Message{To: "bat@x.mn", Subject: "Нууц үг сэргээх", Text: "Сайн байна уу, холбоос: https://surgalt.mn/reset#t=abc", HTML: "<p>Сайн байна уу</p>"}); err != nil {
		t.Fatal(err)
	}
	raw := <-got
	msg, err := nmail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subj, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subj != "Нууц үг сэргээх" || !strings.Contains(msg.Header.Get("To"), "bat@x.mn") || !strings.HasPrefix(msg.Header.Get("Content-Type"), "multipart/alternative") {
		t.Fatalf("толгой: %v (%q)", msg.Header, subj)
	}
	if !strings.Contains(raw, "Content-Transfer-Encoding: base64") {
		t.Fatal("base64 хэсэг алга")
	}
	if _, err := NewSMTP("h", 587, "", "", "буруу хаяг"); err == nil {
		t.Fatal("буруу илгээгчийн хаяг")
	}
}
