package grpcapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/httpapi"
	"surgalt/internal/pb"
	"surgalt/internal/store"
)

func newClient(t *testing.T) (pb.SurgaltClient, *auth.Signer, store.Store) {
	t.Helper()
	st := store.NewMemory()
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	tokens := auth.NewSigner("test-secret-test-secret-test-secret", time.Hour)
	api := httpapi.New(httpapi.Config{DevPayments: true}, st, tokens, chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	gs := New(api)
	ln := bufconn.Listen(1 << 20)
	go func() { _ = gs.Serve(ln) }()
	t.Cleanup(gs.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return ln.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewSurgaltClient(conn), tokens, st
}

func withToken(ctx context.Context, tok string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok)
}

func TestGRPCFlow(t *testing.T) {
	cl, tokens, st := newClient(t)
	ctx := context.Background()
	teacher := &store.User{Username: "bagsh", Email: "b@x.mn", Role: store.RoleTeacher, DisplayName: "Багш Бат"}
	if err := st.CreateUser(ctx, teacher); err != nil {
		t.Fatal(err)
	}
	course := &store.Course{TeacherID: teacher.ID, Title: "ЭЕШ Математик", Price: 0, Published: true}
	if err := st.CreateCourse(ctx, course); err != nil {
		t.Fatal(err)
	}
	lesson := &store.Lesson{CourseID: course.ID, Title: "Логарифм", IsFree: true, ActiveMin: 1}
	if err := st.CreateLesson(ctx, lesson); err != nil {
		t.Fatal(err)
	}
	student := &store.User{Username: "suragch", Email: "s@x.mn", Role: store.RoleStudent, DisplayName: "Сурагч"}
	if err := st.CreateUser(ctx, student); err != nil {
		t.Fatal(err)
	}

	// Нээлттэй: багш, сургалт, хайлт.
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

	// Нэвтрэлт: токенгүй → Unauthenticated.
	if _, err := cl.Me(ctx, &pb.MeRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("токенгүй Me: %v", err)
	}
	tok, err := tokens.SignUser(student.ID, string(store.RoleStudent), student.Username)
	if err != nil {
		t.Fatal(err)
	}
	actx := withToken(ctx, tok)
	me, err := cl.Me(actx, &pb.MeRequest{})
	if err != nil || me.GetId() != student.ID || me.GetRole() != "student" {
		t.Fatalf("Me: %v %v", err, me)
	}

	// Идэвхийн сесс + урсгалаар цохилт → ClickHouse/store-д бичигдэнэ.
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
	// Бусдын сесс → NotFound.
	other, _ := tokens.SignUser(teacher.ID, string(store.RoleTeacher), "bagsh")
	st2, err := cl.Beat(withToken(ctx, other))
	if err != nil {
		t.Fatal(err)
	}
	_ = st2.Send(&pb.BeatRequest{SessionId: ss.GetSessionId(), Active: 1})
	if _, err := st2.Recv(); status.Code(err) != codes.NotFound {
		t.Fatalf("бусдын сесс NotFound байх ёстой: %v", err)
	}

	// Багшийн статистик: сурагч хориотой, багш харна.
	if _, err := cl.GetAnalytics(actx, &pb.AnalyticsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("сурагчид статистик хориотой: %v", err)
	}
	an, err := cl.GetAnalytics(withToken(ctx, other), &pb.AnalyticsRequest{CourseId: course.ID})
	if err != nil || an.GetTotalStudents() != 1 || an.GetStudents()[0].GetViolations() != 1 || len(an.GetDaily()) != 30 {
		t.Fatalf("GetAnalytics: %v %v", err, an)
	}
}
