// loadtest — open-model ачааллын тест: хариуг хүлээхгүйгээр секундэд тогтмол
// тооны хүсэлт илгээнэ (бодит "секундэд 10000 хүн орж ирэх" нөхцөл).
//
//	go run ./cmd/loadtest -base http://localhost:8080 -rate 10000 -d 10s \
//	   -paths /t/demo,/api/teachers/demo,/c/<id>
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	base := flag.String("base", "http://localhost:8080", "серверийн хаяг")
	paths := flag.String("paths", "/t/demo", "таслалаар тусгаарласан замууд (санамсаргүй сонгоно)")
	rate := flag.Int("rate", 10000, "секунд дэх хүсэлт")
	dur := flag.Duration("d", 10*time.Second, "үргэлжлэх хугацаа")
	maxInflight := flag.Int("inflight", 20000, "зэрэг хүлээгдэж буй хүсэлтийн дээд тоо")
	timeout := flag.Duration("timeout", 10*time.Second, "хүсэлт бүрийн timeout")
	flag.Parse()

	ps := strings.Split(*paths, ",")
	tr := &http.Transport{
		MaxIdleConns: *maxInflight, MaxIdleConnsPerHost: *maxInflight, IdleConnTimeout: 90 * time.Second,
		DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: tr, Timeout: *timeout}

	total := int(float64(*rate) * dur.Seconds())
	lat := make([]int64, total)
	var (
		idx, sent, ok, dropped atomic.Int64
		mu                     sync.Mutex
		codes                  = map[int]int{}
		errs                   = map[string]int{}
		wg                     sync.WaitGroup
	)
	sem := make(chan struct{}, *maxInflight)
	fire := func() {
		select {
		case sem <- struct{}{}:
		default:
			dropped.Add(1)
			return
		}
		sent.Add(1)
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			t0 := time.Now()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, *base+ps[rand.IntN(len(ps))], nil)
			res, err := client.Do(req)
			if err != nil {
				msg := err.Error()
				if i := strings.LastIndex(msg, ": "); i > 0 {
					msg = msg[i+2:]
				}
				mu.Lock()
				errs[msg]++
				mu.Unlock()
				return
			}
			_, _ = io.Copy(io.Discard, res.Body)
			res.Body.Close()
			if i := idx.Add(1) - 1; int(i) < len(lat) {
				lat[i] = int64(time.Since(t0))
			}
			mu.Lock()
			codes[res.StatusCode]++
			mu.Unlock()
			if res.StatusCode < 400 {
				ok.Add(1)
			}
		}()
	}

	fmt.Printf("→ %s  %d req/s × %s = %d хүсэлт, замууд: %v\n", *base, *rate, *dur, total, ps)
	start := time.Now()
	const tick = time.Millisecond
	perTick := float64(*rate) * tick.Seconds()
	var acc float64
	fired := 0
	t := time.NewTicker(tick)
	for fired < total {
		<-t.C
		// Цаг хоцорсон ч нийт хурдыг барина.
		want := int(time.Since(start).Seconds()*float64(*rate)) - fired
		acc += perTick
		n := max(want, int(acc))
		acc -= float64(int(acc))
		for i := 0; i < n && fired < total; i++ {
			fire()
			fired++
		}
	}
	t.Stop()
	sendDur := time.Since(start)
	wg.Wait()
	all := time.Since(start)

	n := min(int(idx.Load()), len(lat))
	l := lat[:n]
	slices.Sort(l)
	pct := func(p float64) time.Duration {
		if n == 0 {
			return 0
		}
		return time.Duration(l[min(n-1, int(float64(n)*p))])
	}
	fmt.Printf("\nИлгээсэн:        %d (%.0f req/s илгээлтийн хурд)\n", sent.Load(), float64(sent.Load())/sendDur.Seconds())
	fmt.Printf("Амжилттай (<400): %d  (%.2f%%)\n", ok.Load(), 100*float64(ok.Load())/float64(max(1, sent.Load())))
	fmt.Printf("Бүрэн дууссан:   %s  → бодит дамжуулалт %.0f req/s\n", all.Round(time.Millisecond), float64(ok.Load())/all.Seconds())
	fmt.Printf("Latency:         p50=%s  p90=%s  p99=%s  p99.9=%s  max=%s\n",
		pct(.50).Round(time.Microsecond), pct(.90).Round(time.Microsecond), pct(.99).Round(time.Microsecond), pct(.999).Round(time.Microsecond), pct(1).Round(time.Microsecond))
	fmt.Printf("HTTP кодууд:     %v\n", codes)
	if dropped.Load() > 0 {
		fmt.Printf("Клиент талд хаягдсан (inflight дүүрсэн): %d\n", dropped.Load())
	}
	if len(errs) > 0 {
		fmt.Printf("Алдаанууд:       %v\n", errs)
	}
	if ok.Load() < sent.Load() {
		os.Exit(1)
	}
}
