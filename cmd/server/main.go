// Surgalt — багш нарын сургалт зарах платформын сервер.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"surgalt/internal/grpcapi"
	"syscall"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/httpapi"
	"surgalt/internal/meet"
	"surgalt/internal/oauth"
	"surgalt/internal/store"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(k), 10, 64); err == nil {
		return v
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(k), 64); err == nil {
		return v
	}
	return def
}

// parsePlans: "300:5000,500:8000" → 300MB сард 5000₮, 500MB сард 8000₮.
func parsePlans(v string) []httpapi.StoragePlan {
	var out []httpapi.StoragePlan
	for _, part := range strings.Split(v, ",") {
		mb, price, ok := strings.Cut(strings.TrimSpace(part), ":")
		m, err1 := strconv.ParseInt(mb, 10, 64)
		p, err2 := strconv.ParseInt(price, 10, 64)
		if ok && err1 == nil && err2 == nil && m > 0 && p > 0 {
			out = append(out, httpapi.StoragePlan{MB: m, Price: p})
		}
	}
	return out
}

func envBool(k string) bool { b, _ := strconv.ParseBool(os.Getenv(k)); return b }

// loadDotEnv нь ажлын хавтас дахь .env файлыг уншина (байвал). Аль хэдийн тохируулсан
// орчны хувьсагчийг дарж бичихгүй — Docker/systemd-ийн тохиргоо үргэлж давуу.
func loadDotEnv(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k, v = strings.TrimSpace(strings.TrimPrefix(k, "export ")), strings.TrimSpace(v)
		if !ok || k == "" {
			continue
		}
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		if _, set := os.LookupEnv(k); !set && v != "" {
			os.Setenv(k, v)
			n++
		}
	}
	return n, nil
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if n, err := loadDotEnv(".env"); err != nil {
		log.Warn(".env уншиж чадсангүй", "err", err)
	} else if n > 0 {
		log.Info(".env ачааллаа", "vars", n)
	}
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	secret := os.Getenv("TOKEN_SECRET")
	chDSN := os.Getenv("CLICKHOUSE_DSN")
	if secret == "" {
		if chDSN != "" {
			return errors.New("TOKEN_SECRET заавал тохируулна (32+ тэмдэгт)")
		}
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = hex.EncodeToString(b)
		log.Warn("TOKEN_SECRET тохируулаагүй — түр нууц үүсгэлээ (dev горим)")
	}

	// Өгөгдлийн сан: бүх өгөгдөл ClickHouse-д. DSN хоосон бол санах ойн store (хөгжүүлэлт).
	var st store.Store
	if chDSN != "" {
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		ch, err := store.NewClickHouse(cctx, store.ClickHouseOptions{DSN: chDSN})
		cancel()
		if err != nil {
			return err
		}
		st = ch
		log.Info("ClickHouse холбогдлоо")
	} else {
		st = store.NewMemory()
		log.Warn("CLICKHOUSE_DSN хоосон — санах ойн store (өгөгдөл хадгалагдахгүй)")
	}
	defer st.Close()

	fstore, err := files.New(env("STORAGE_DIR", "./data"), []byte(secret), envInt("MAX_FILE_MB", 2048)<<20)
	if err != nil {
		return err
	}

	fstore.StartTranscoder(ctx, int(envInt("VIDEO_WORKERS", 1)), log)

	hub := chat.NewHub()
	tokens := auth.NewSigner(secret, 7*24*time.Hour)
	srv := httpapi.New(httpapi.Config{
		PublicURL:     os.Getenv("PUBLIC_URL"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		DevPayments:   envBool("DEV_PAYMENTS"),
		TrustProxy:    envBool("TRUST_PROXY"),
		LogRequests:   envBool("LOG_REQUESTS"),
		RateLimitRPS:  envFloat("RATE_LIMIT_RPS", 30),
		RateBurst:     envFloat("RATE_BURST", 60),
		StorageFreeMB: envInt("STORAGE_FREE_MB", 300),
		StoragePlans:  parsePlans(env("STORAGE_PLANS", "300:5000,500:8000,1024:14000,2048:25000,5120:55000")),
	}, st, tokens, hub, fstore, log)

	base := env("PUBLIC_URL", "http://localhost"+env("ADDR", ":8080"))
	srv.OAuth = oauth.NewRegistry(base,
		oauth.Credentials{ClientID: os.Getenv("GOOGLE_CLIENT_ID"), ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET")},
		oauth.Credentials{ClientID: os.Getenv("MICROSOFT_CLIENT_ID"), ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET")},
		oauth.Credentials{ClientID: os.Getenv("FACEBOOK_CLIENT_ID"), ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET")},
		oauth.Credentials{ClientID: os.Getenv("INSTAGRAM_CLIENT_ID"), ClientSecret: os.Getenv("INSTAGRAM_CLIENT_SECRET")},
	)
	srv.Meet = meet.New(base, os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"))
	active := map[string]bool{}
	for _, p := range srv.OAuth.List() {
		active[p.Name] = true
		log.Info("OAuth идэвхтэй", "provider", p.Name, "callback", strings.TrimRight(base, "/")+"/auth/"+p.Name+"/callback")
	}
	// Google, Facebook-ээр нэвтрэх товч зөвхөн түлхүүр тохируулсан үед гарна — юу дутууг тодорхой хэлнэ.
	for _, n := range []string{"google", "facebook"} {
		if !active[n] {
			log.Warn("OAuth тохируулаагүй — нэвтрэх товч харагдахгүй", "provider", n,
				"env", strings.ToUpper(n)+"_CLIENT_ID, "+strings.ToUpper(n)+"_CLIENT_SECRET",
				"callback", strings.TrimRight(base, "/")+"/auth/"+n+"/callback")
		}
	}

	if envBool("SEED_DEMO") {
		if err := seedDemo(ctx, st, fstore); err != nil {
			log.Warn("seed", "err", err)
		}
	}

	srv.Background(ctx)
	hs := &http.Server{
		Addr:              env("ADDR", ":8080"),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("сервер эхэллээ", "addr", hs.Addr)
		errc <- hs.ListenAndServe()
	}()
	// gRPC + Protobuf API (мобайл, сервис хоорондын харилцаа) — ижил логик, ижил ClickHouse.
	var gs *grpc.Server
	if addr := os.Getenv("GRPC_ADDR"); addr != "" {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("grpc listen: %w", err)
		}
		gs = grpcapi.New(srv)
		go func() {
			log.Info("gRPC сервер эхэллээ", "addr", addr)
			if err := gs.Serve(ln); err != nil {
				errc <- fmt.Errorf("grpc: %w", err)
			}
		}()
	}
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("зогсож байна — идэвхтэй хүсэлтүүдийг дуусгаж байна")
	if gs != nil {
		gs.GracefulStop()
	}
	sctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return hs.Shutdown(sctx)
}
