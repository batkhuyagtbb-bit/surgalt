package grpcapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/httpapi"
	"surgalt/internal/pb"
	"surgalt/internal/store"
)

// newStack: нэг Connect handler-ийг h2c-тэй HTTP сервер дээр асаана — үүнийг grpc-go клиент (жинхэнэ gRPC)
// ба энгийн JSON POST (хөтөч) хоёулаа дуудна.
func newStack(t *testing.T) (*httptest.Server, pb.SurgaltClient, *auth.Signer, store.Store) {
	t.Helper()
	st := store.NewMemory()
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	tokens := auth.NewSigner("test-secret-test-secret-test-secret", time.Hour)
	api := httpapi.New(httpapi.Config{DevPayments: true}, st, tokens, chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	path, h := Handler(api)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	ts := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	t.Cleanup(ts.Close)
	conn, err := grpc.NewClient(strings.TrimPrefix(ts.URL, "http://"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return ts, pb.NewSurgaltClient(conn), tokens, st
}

func withToken(ctx context.Context, tok string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok)
}

func seed(t *testing.T, st store.Store) (teacher, student *store.User, course *store.Course, lesson *store.Lesson) {
	t.Helper()
	ctx := context.Background()
	teacher = &store.User{Username: "bagsh", Email: "b@x.mn", Role: store.RoleTeacher, DisplayName: "Багш Бат"}
	if err := st.CreateUser(ctx, teacher); err != nil {
		t.Fatal(err)
	}
	course = &store.Course{TeacherID: teacher.ID, Title: "ЭЕШ Математик", Price: 0, Published: true}
	if err := st.CreateCourse(ctx, course); err != nil {
		t.Fatal(err)
	}
	lesson = &store.Lesson{CourseID: course.ID, Title: "Логарифм", IsFree: true, ActiveMin: 1}
	if err := st.CreateLesson(ctx, lesson); err != nil {
		t.Fatal(err)
	}
	student = &store.User{Username: "suragch", Email: "s@x.mn", Role: store.RoleStudent, DisplayName: "Сурагч"}
	if err := st.CreateUser(ctx, student); err != nil {
		t.Fatal(err)
	}
	return
}

// Жинхэнэ gRPC протокол (grpc-go клиент) Connect handler дээр.
func TestGRPCFlow(t *testing.T) {
	_, cl, tokens, st := newStack(t)
	teacher, student, course, lesson := seed(t, st)
	ctx := context.Background()

	tp, err := cl.GetTeacher(ctx, &pb.GetTeacherRequest{Username: "bagsh"})
	if err != nil || tp.GetTeacher().GetDisplayName() != "Багш Бат" || len(tp.GetCourses()) != 1 {
		t.Fatalf("GetTeacher: %v %v", err, tp)
	}
	cd, err := cl.GetCourse(ctx, &pb.GetCourseRequest{Id: course.ID})
	if err != nil || cd.GetCourse().GetTitle() != "ЭЕШ Математик" || len(cd.GetLessons()) != 1 || cd.GetLessons()[0].GetActiveMin() != 1 {
		t.Fatalf("GetCourse: %v %v", err, cd)
	}
	if _, err := cl.GetCourse(ctx, &pb.GetCourseRequest{Id: "000000000000000000000000"}); status.Code(err) != codes.NotFound {
		t.Fatalf("байхгүй сургалт NotFound байх ёстой: %v", err)
	}
	sr, err := cl.Search(ctx, &pb.SearchRequest{Query: "matematik"})
	if err != nil || sr.GetTotal() != 1 || sr.GetItems()[0].GetTeacher().GetUsername() != "bagsh" {
		t.Fatalf("Search: %v %v", err, sr)
	}
	if _, err := cl.Me(ctx, &pb.MeRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("токенгүй Me: %v", err)
	}
	tok, _ := tokens.SignUser(student.ID, string(store.RoleStudent), student.Username)
	actx := withToken(ctx, tok)
	me, err := cl.Me(actx, &pb.MeRequest{})
	if err != nil || me.GetId() != student.ID || me.GetRole() != "student" {
		t.Fatalf("Me: %v %v", err, me)
	}
	ss, err := cl.StartSession(actx, &pb.StartSessionRequest{CourseId: course.ID, LessonId: lesson.ID})
	if err != nil || ss.GetSessionId() == "" || ss.GetPolicy().GetActiveMin() != 1 || ss.GetWatermark().GetName() != "Сурагч" {
		t.Fatalf("StartSession: %v %v", err, ss)
	}
	stream, err := cl.Beat(actx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&pb.BeatRequest{SessionId: ss.GetSessionId(), Active: 3, Events: []*pb.BeatEvent{{Type: "tab_switch", Detail: "тест"}}}); err != nil {
		t.Fatal(err)
	}
	resp, err := stream.Recv()
	if err != nil || !resp.GetOk() {
		t.Fatalf("Beat: %v %v", err, resp)
	}
	_ = stream.CloseSend()
	sessions, _ := st.Sessions(ctx, store.ActivityFilter{UserID: student.ID}, 10)
	if len(sessions) != 1 || sessions[0].Counts["tab_switch"] != 1 {
		t.Fatalf("сесс бичигдээгүй: %+v", sessions)
	}
	other, _ := tokens.SignUser(teacher.ID, string(store.RoleTeacher), "bagsh")
	st2, err := cl.Beat(withToken(ctx, other))
	if err != nil {
		t.Fatal(err)
	}
	_ = st2.Send(&pb.BeatRequest{SessionId: ss.GetSessionId(), Active: 1})
	if _, err := st2.Recv(); status.Code(err) != codes.NotFound {
		t.Fatalf("бусдын сесс NotFound байх ёстой: %v", err)
	}
	if _, err := cl.GetAnalytics(actx, &pb.AnalyticsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("сурагчид статистик хориотой: %v", err)
	}
	an, err := cl.GetAnalytics(withToken(ctx, other), &pb.AnalyticsRequest{CourseId: course.ID})
	if err != nil || an.GetTotalStudents() != 1 || an.GetStudents()[0].GetViolations() != 1 || len(an.GetDaily()) != 30 {
		t.Fatalf("GetAnalytics: %v %v", err, an)
	}
}

// Хөтчийн зам: ижил Protobuf API-г энгийн JSON POST-оор (Connect протокол) — fetch-тэй адил.
func TestConnectJSONFromBrowser(t *testing.T) {
	ts, _, tokens, st := newStack(t)
	_, student, course, lesson := seed(t, st)
	post := func(path, tok, body string) (int, map[string]any) {
		req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	code, sr := post("/surgalt.v1.Surgalt/Search", "", `{"query":"matematik"}`)
	items, _ := sr["items"].([]any)
	if code != 200 || len(items) != 1 || items[0].(map[string]any)["teacher"].(map[string]any)["username"] != "bagsh" {
		t.Fatalf("JSON Search: %d %v", code, sr)
	}
	if code, e := post("/surgalt.v1.Surgalt/Me", "", `{}`); code != 401 || e["code"] != "unauthenticated" {
		t.Fatalf("токенгүй JSON Me 401 байх ёстой: %d %v", code, e)
	}
	tok, _ := tokens.SignUser(student.ID, string(store.RoleStudent), student.Username)
	code, ss := post("/surgalt.v1.Surgalt/StartSession", tok, `{"courseId":"`+course.ID+`","lessonId":"`+lesson.ID+`"}`)
	sid, _ := ss["sessionId"].(string)
	if code != 200 || sid == "" || ss["policy"].(map[string]any)["activeMin"].(float64) != 1 {
		t.Fatalf("JSON StartSession: %d %v", code, ss)
	}
	// Proto3 JSON: snake_case талбарын нэр ч хүлээн авна (хуучин JS-тэй нийцтэй).
	code, b := post("/surgalt.v1.Surgalt/BeatOnce", tok, `{"session_id":"`+sid+`","active":4,"events":[{"type":"copy","detail":"json"}]}`)
	if code != 200 || b["ok"] != true || b["activeDoneSec"].(float64) != 4 {
		t.Fatalf("JSON BeatOnce: %d %v", code, b)
	}
	sessions, _ := st.Sessions(context.Background(), store.ActivityFilter{UserID: student.ID}, 10)
	if len(sessions) != 1 || sessions[0].ActiveSec != 4 || sessions[0].Counts["copy"] != 1 {
		t.Fatalf("JSON тайлан сессэд ороогүй: %+v", sessions)
	}
	// Эрхгүй хичээл → 403 (permission_denied).
	paid := &store.Course{TeacherID: student.ID, Title: "x", Price: 1000, Published: true}
	_ = st.CreateCourse(context.Background(), paid)
	pl := &store.Lesson{CourseID: paid.ID, Title: "p", Price: 500}
	_ = st.CreateLesson(context.Background(), pl)
	other := &store.User{Username: "o", Email: "o@x.mn", Role: store.RoleStudent, DisplayName: "O"}
	_ = st.CreateUser(context.Background(), other)
	otok, _ := tokens.SignUser(other.ID, string(store.RoleStudent), "o")
	if code, e := post("/surgalt.v1.Surgalt/StartSession", otok, `{"courseId":"`+paid.ID+`","lessonId":"`+pl.ID+`"}`); code != 403 || e["code"] != "permission_denied" {
		t.Fatalf("төлбөртэй хичээл 403 байх ёстой: %d %v", code, e)
	}
}
