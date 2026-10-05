// Package grpcapi — surgalt.mn-ийн gRPC + Protobuf үйлчилгээ.
//
// HTTP API-тай ижил бизнес логикийг (httpapi.Server-ийн экспортлосон функцүүд) дуудна, тиймээс
// хоёр API нэг л дүрэм, нэг л ClickHouse сан дээр ажиллана. Хөтөч HTML/JSON-оор, мобайл ба
// сервис хоорондын клиент gRPC-ээр холбогдоно.
package grpcapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"surgalt/internal/httpapi"
	"surgalt/internal/pb"
	"surgalt/internal/store"
)

type Server struct {
	pb.UnimplementedSurgaltServer
	api *httpapi.Server
}

// New нь gRPC серверийг үүсгэж үйлчилгээг бүртгэнэ.
func New(api *httpapi.Server, opts ...grpc.ServerOption) *grpc.Server {
	gs := grpc.NewServer(append([]grpc.ServerOption{grpc.MaxRecvMsgSize(4 << 20)}, opts...)...)
	pb.RegisterSurgaltServer(gs, &Server{api: api})
	return gs
}

// ---- алдаа ----

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	var ae *httpapi.APIError
	if errors.As(err, &ae) {
		return status.Error(httpCode(ae.Code), ae.Msg)
	}
	if errors.Is(err, store.ErrNotFound) {
		return status.Error(codes.NotFound, "олдсонгүй")
	}
	if errors.Is(err, store.ErrConflict) {
		return status.Error(codes.AlreadyExists, "давхардсан")
	}
	return status.Error(codes.Internal, "алдаа гарлаа")
}

func httpCode(c int) codes.Code {
	switch c {
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusPaymentRequired, http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusLocked:
		return codes.FailedPrecondition
	case http.StatusConflict:
		return codes.FailedPrecondition
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	}
	return codes.Internal
}

// ---- нэвтрэлт ----

type principal struct{ uid, role, name string }

// auth нь metadata "authorization: Bearer <token>"-оос хэрэглэгчийг танина.
func (s *Server) auth(ctx context.Context) (principal, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	var tok string
	for _, v := range md.Get("authorization") {
		tok = strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
	}
	if tok == "" {
		return principal{}, status.Error(codes.Unauthenticated, "нэвтэрнэ үү")
	}
	uid, role, name, err := s.api.VerifyToken(tok)
	if err != nil || uid == "" {
		return principal{}, status.Error(codes.Unauthenticated, "токен хүчингүй")
	}
	return principal{uid, role, name}, nil
}

func clientIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-real-ip"); len(v) > 0 {
			return v[0]
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return ""
}

// ---- хөрвүүлэлт ----

func teacherPB(t httpapi.PublicTeacher) *pb.Teacher {
	links := make([]*pb.Link, 0, len(t.Links))
	for _, l := range t.Links {
		links = append(links, &pb.Link{Key: l.Key, Label: l.Label, Url: l.URL})
	}
	return &pb.Teacher{Id: t.ID, Username: t.Username, DisplayName: t.DisplayName, Headline: t.Headline, Bio: t.Bio,
		AvatarUrl: t.AvatarURL, CoverUrl: t.CoverURL, Subjects: t.Subjects, Location: t.Location, Links: links,
		JoinedYear: int32(t.JoinedYear), StudentCount: t.StudentCount, CourseCount: int32(t.CourseCount),
		LessonCount: int32(t.LessonCount), FreeLessonCount: int32(t.FreeLessonCount)}
}

func coursePB(c store.Course) *pb.Course {
	return &pb.Course{Id: c.ID, TeacherId: c.TeacherID, Title: c.Title, Description: c.Description, Price: c.Price,
		Published: c.Published, Drip: c.Drip, UnlockAllPaid: c.UnlockAllPaid, Camera: c.Camera, Certificate: c.Certificate,
		LessonCount: int32(c.LessonCount), FreeLessonCount: int32(c.FreeLessonCount), Views: c.Views, CreatedAtUnix: c.CreatedAt.Unix()}
}

// ---- нээлттэй ----

func (s *Server) GetTeacher(ctx context.Context, in *pb.GetTeacherRequest) (*pb.TeacherProfile, error) {
	p, err := s.api.PublicTeacherByUsername(ctx, in.GetUsername())
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.TeacherProfile{Teacher: teacherPB(p.Teacher), TopCourseId: p.TopCourseID}
	for _, c := range p.Courses {
		out.Courses = append(out.Courses, coursePB(c))
	}
	return out, nil
}

func (s *Server) GetCourse(ctx context.Context, in *pb.GetCourseRequest) (*pb.CourseDetail, error) {
	c, err := s.api.PublicCourseByID(ctx, in.GetId())
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.CourseDetail{Course: coursePB(c.Course), Teacher: teacherPB(c.Teacher)}
	for _, l := range c.Lessons {
		out.Lessons = append(out.Lessons, &pb.Lesson{Id: l.ID, Title: l.Title, IsFree: l.IsFree, Price: l.Price, Position: int32(l.Position),
			UnlockAfterH: int32(l.UnlockAfterH), AlwaysOpen: l.AlwaysOpen, Format: l.Format, Mode: l.Mode, Section: l.Section,
			ActiveMin: int32(l.ActiveMin), Exam: l.Exam != nil})
	}
	return out, nil
}

func (s *Server) Search(ctx context.Context, in *pb.SearchRequest) (*pb.SearchResponse, error) {
	res, err := s.api.SearchCourses(ctx, in.GetQuery(), in.GetTag(), int(in.GetPage()))
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.SearchResponse{Total: int32(res.Total), Page: int32(res.Page), Pages: int32(res.Pages), PerPage: int32(res.PerPage)}
	for _, h := range res.Items {
		out.Items = append(out.Items, &pb.SearchHit{CourseId: h.CourseID, Title: h.Title, Teacher: teacherPB(h.Teacher), Price: h.Price,
			Lessons: int32(h.Lessons), FreeLessons: int32(h.FreeLessons), Views: h.Views, Tags: h.Tags, Match: h.Match})
	}
	return out, nil
}

// ---- нэвтэрсэн ----

func (s *Server) Me(ctx context.Context, _ *pb.MeRequest) (*pb.User, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.api.Store().UserByID(ctx, p.uid)
	if err != nil {
		return nil, toStatus(err)
	}
	return &pb.User{Id: u.ID, Username: u.Username, Role: string(u.Role), DisplayName: u.DisplayName, AvatarUrl: u.AvatarURL}, nil
}

func (s *Server) StartSession(ctx context.Context, in *pb.StartSessionRequest) (*pb.StartSessionResponse, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.api.StartActivity(ctx, p.uid, p.name, clientIP(ctx), in.GetCourseId(), in.GetLessonId(), in.GetKind())
	if err != nil {
		return nil, toStatus(err)
	}
	return &pb.StartSessionResponse{
		SessionId: st.SessionID,
		Watermark: &pb.Watermark{Name: st.Watermark["name"], Id: st.Watermark["id"], Ip: st.Watermark["ip"]},
		Policy: &pb.Policy{Camera: st.Policy.Camera, PingSec: int32(st.Policy.PingSec), PingAnswerSec: int32(st.Policy.PingAnswerSec),
			IdleSec: int32(st.Policy.IdleSec), BeatSec: int32(st.Policy.BeatSec), ActiveMin: int32(st.Policy.ActiveMin),
			ActiveDoneSec: int32(st.Policy.ActiveDoneSec), Owner: st.Policy.Owner},
	}, nil
}

// Beat — хоёр чиглэлт урсгал: клиент тайлан бүрт сервер нийт идэвхтэй секундийг буцаана.
func (s *Server) Beat(stream pb.Surgalt_BeatServer) error {
	ctx := stream.Context()
	p, err := s.auth(ctx)
	if err != nil {
		return err
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		b := httpapi.BeatInput{SessionID: in.GetSessionId(), Active: int(in.GetActive()), Idle: int(in.GetIdle()), Away: int(in.GetAway()),
			FocusSum: in.GetFocusSum(), FocusN: int(in.GetFocusN()), Camera: in.GetCamera(), AttemptID: in.GetAttemptId(), End: in.GetEnd()}
		for _, e := range in.GetEvents() {
			b.Events = append(b.Events, httpapi.BeatEvent{Type: e.GetType(), Detail: e.GetDetail()})
		}
		done, err := s.api.RecordBeat(ctx, p.uid, b)
		if err != nil {
			return toStatus(err)
		}
		if err := stream.Send(&pb.BeatResponse{Ok: true, ActiveDoneSec: int32(done)}); err != nil {
			return err
		}
	}
}

// ---- багш ----

func (s *Server) GetAnalytics(ctx context.Context, in *pb.AnalyticsRequest) (*pb.AnalyticsResponse, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	if p.role != string(store.RoleTeacher) {
		return nil, status.Error(codes.PermissionDenied, "зөвхөн багш")
	}
	data, err := s.api.Analytics(ctx, p.uid, in.GetCourseId(), int(in.GetDays()))
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.AnalyticsResponse{}
	for _, st := range data.Students {
		counts := make(map[string]int32, len(st.Counts))
		for k, v := range st.Counts {
			counts[k] = int32(v)
		}
		out.Students = append(out.Students, &pb.StudentStat{UserId: st.UserID, Name: st.Name, Sessions: int32(st.Sessions), TotalSec: int32(st.TotalSec),
			ActiveSec: int32(st.ActiveSec), ActivePct: int32(st.ActivePct), Attention: int32(st.Attention), Violations: int32(st.Violations),
			Lessons: int32(st.Lessons), ExamBest: int32(st.ExamBest), QuizAccuracy: int32(st.QuizAccuracy), QuizTotal: int32(st.QuizTotal),
			Reflections: int32(st.Reflections), VideoCoverage: int32(st.VideoCoverage), StreakDays: int32(st.StreakDays),
			LearnScore: int32(st.LearnScore), Risk: st.Risk, LastAtUnix: st.LastAt.Unix(), Counts: counts})
	}
	for _, d := range data.Daily {
		out.Daily = append(out.Daily, &pb.DailyActivity{Day: str(d["day"]), ActiveSec: i32(d["active_sec"]), InactiveSec: i32(d["inactive_sec"])})
	}
	t := data.Totals
	out.TotalStudents, out.Live, out.ActivePct = i32(t["students"]), i32(t["live"]), i32(t["active_pct"])
	out.Attention, out.Violations, out.LearnScore = i32(t["attention"]), i32(t["violations"]), i32(t["learn_score"])
	return out, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func i32(v any) int32 {
	switch n := v.(type) {
	case int:
		return int32(n)
	case int32:
		return n
	case int64:
		return int32(n)
	case float64:
		return int32(n)
	}
	return 0
}
