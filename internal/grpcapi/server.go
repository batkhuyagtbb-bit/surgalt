// Package grpcapi — surgalt.mn-ийн Protobuf API. Нэг handler (Connect) гурван протоколоор үйлчилнэ:
//   - gRPC (HTTP/2): мобайл апп, сервис хоорондын клиент (grpc-go, grpc-swift, grpc-kotlin ...)
//   - gRPC-Web: хөтчийн gRPC-Web клиент
//   - Connect-JSON: хөтөч энгийн fetch-ээр — POST /surgalt.v1.Surgalt/<Method>, Content-Type: application/json
//
// HTTP API-тай ижил бизнес логикийг (httpapi.Server-ийн экспортлосон функцүүд) дуудна, тиймээс
// бүх клиент нэг л дүрэм, нэг л ClickHouse сан дээр ажиллана.
package grpcapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"surgalt/internal/httpapi"
	"surgalt/internal/pb"
	"surgalt/internal/pb/pbconnect"
	"surgalt/internal/store"
)

type Server struct {
	api *httpapi.Server
}

// Handler нь (зам, handler) буцаана; HTTP mux-д `mux.Handle(path, h)` гэж байрлуулна.
// Урсгалтай RPC (Beat) серверийн Read/WriteTimeout-оос урт байж болох тул deadline-ийг цэвэрлэнэ.
func Handler(api *httpapi.Server) (string, http.Handler) {
	path, h := pbconnect.NewSurgaltHandler(&Server{api: api}, connect.WithCompressMinBytes(1024))
	return path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		_ = rc.SetReadDeadline(time.Time{})
		_ = rc.SetWriteDeadline(time.Time{})
		h.ServeHTTP(w, r)
	})
}

// ---- алдаа ----

func toConnect(err error) error {
	if err == nil {
		return nil
	}
	var ae *httpapi.APIError
	if errors.As(err, &ae) {
		return connect.NewError(codeOf(ae.Code), errors.New(ae.Msg))
	}
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, errors.New("олдсонгүй"))
	}
	if errors.Is(err, store.ErrConflict) {
		return connect.NewError(connect.CodeAlreadyExists, errors.New("давхардсан"))
	}
	return connect.NewError(connect.CodeInternal, errors.New("алдаа гарлаа"))
}

func codeOf(httpCode int) connect.Code {
	switch httpCode {
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusPaymentRequired, http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusLocked, http.StatusConflict:
		return connect.CodeFailedPrecondition
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	}
	return connect.CodeInternal
}

// ---- нэвтрэлт ----

type principal struct{ uid, role, name string }

// auth: "Authorization: Bearer <token>" толгойгоос хэрэглэгчийг танина.
func (s *Server) auth(h http.Header) (principal, error) {
	tok := strings.TrimSpace(strings.TrimPrefix(h.Get("Authorization"), "Bearer "))
	if tok == "" {
		return principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("нэвтэрнэ үү"))
	}
	uid, role, name, err := s.api.VerifyToken(tok)
	if err != nil || uid == "" {
		return principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("токен хүчингүй"))
	}
	return principal{uid, role, name}, nil
}

func clientIP(h http.Header, peer connect.Peer) string {
	if v := h.Get("X-Real-IP"); v != "" {
		return v
	}
	if host, _, err := net.SplitHostPort(peer.Addr); err == nil {
		return host
	}
	return peer.Addr
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

func beatInput(in *pb.BeatRequest) httpapi.BeatInput {
	b := httpapi.BeatInput{SessionID: in.GetSessionId(), Active: int(in.GetActive()), Idle: int(in.GetIdle()), Away: int(in.GetAway()),
		FocusSum: in.GetFocusSum(), FocusN: int(in.GetFocusN()), Camera: in.GetCamera(), AttemptID: in.GetAttemptId(), End: in.GetEnd()}
	for _, e := range in.GetEvents() {
		b.Events = append(b.Events, httpapi.BeatEvent{Type: e.GetType(), Detail: e.GetDetail()})
	}
	return b
}

// ---- нээлттэй ----

func (s *Server) GetTeacher(ctx context.Context, req *connect.Request[pb.GetTeacherRequest]) (*connect.Response[pb.TeacherProfile], error) {
	p, err := s.api.PublicTeacherByUsername(ctx, req.Msg.GetUsername())
	if err != nil {
		return nil, toConnect(err)
	}
	out := &pb.TeacherProfile{Teacher: teacherPB(p.Teacher), TopCourseId: p.TopCourseID}
	for _, c := range p.Courses {
		out.Courses = append(out.Courses, coursePB(c))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) GetCourse(ctx context.Context, req *connect.Request[pb.GetCourseRequest]) (*connect.Response[pb.CourseDetail], error) {
	c, err := s.api.PublicCourseByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, toConnect(err)
	}
	out := &pb.CourseDetail{Course: coursePB(c.Course), Teacher: teacherPB(c.Teacher)}
	for _, l := range c.Lessons {
		out.Lessons = append(out.Lessons, &pb.Lesson{Id: l.ID, Title: l.Title, IsFree: l.IsFree, Price: l.Price, Position: int32(l.Position),
			UnlockAfterH: int32(l.UnlockAfterH), AlwaysOpen: l.AlwaysOpen, Format: l.Format, Mode: l.Mode, Section: l.Section,
			ActiveMin: int32(l.ActiveMin), Exam: l.Exam != nil})
	}
	return connect.NewResponse(out), nil
}

func (s *Server) Search(ctx context.Context, req *connect.Request[pb.SearchRequest]) (*connect.Response[pb.SearchResponse], error) {
	res, err := s.api.SearchCourses(ctx, req.Msg.GetQuery(), req.Msg.GetTag(), int(req.Msg.GetPage()))
	if err != nil {
		return nil, toConnect(err)
	}
	out := &pb.SearchResponse{Total: int32(res.Total), Page: int32(res.Page), Pages: int32(res.Pages), PerPage: int32(res.PerPage)}
	for _, h := range res.Items {
		out.Items = append(out.Items, &pb.SearchHit{CourseId: h.CourseID, Title: h.Title, Teacher: teacherPB(h.Teacher), Price: h.Price,
			Lessons: int32(h.Lessons), FreeLessons: int32(h.FreeLessons), Views: h.Views, Tags: h.Tags, Match: h.Match})
	}
	resp := connect.NewResponse(out)
	resp.Header().Set("Cache-Control", "public, max-age=15")
	return resp, nil
}

// ---- нэвтэрсэн ----

func (s *Server) Me(ctx context.Context, req *connect.Request[pb.MeRequest]) (*connect.Response[pb.User], error) {
	p, err := s.auth(req.Header())
	if err != nil {
		return nil, err
	}
	u, err := s.api.Store().UserByID(ctx, p.uid)
	if err != nil {
		return nil, toConnect(err)
	}
	return connect.NewResponse(&pb.User{Id: u.ID, Username: u.Username, Role: string(u.Role), DisplayName: u.DisplayName, AvatarUrl: u.AvatarURL}), nil
}

func (s *Server) StartSession(ctx context.Context, req *connect.Request[pb.StartSessionRequest]) (*connect.Response[pb.StartSessionResponse], error) {
	p, err := s.auth(req.Header())
	if err != nil {
		return nil, err
	}
	st, err := s.api.StartActivity(ctx, p.uid, p.name, clientIP(req.Header(), req.Peer()), req.Msg.GetCourseId(), req.Msg.GetLessonId(), req.Msg.GetKind())
	if err != nil {
		return nil, toConnect(err)
	}
	return connect.NewResponse(&pb.StartSessionResponse{
		SessionId: st.SessionID,
		Watermark: &pb.Watermark{Name: st.Watermark["name"], Id: st.Watermark["id"], Ip: st.Watermark["ip"]},
		Policy: &pb.Policy{Camera: st.Policy.Camera, PingSec: int32(st.Policy.PingSec), PingAnswerSec: int32(st.Policy.PingAnswerSec),
			IdleSec: int32(st.Policy.IdleSec), BeatSec: int32(st.Policy.BeatSec), ActiveMin: int32(st.Policy.ActiveMin),
			ActiveDoneSec: int32(st.Policy.ActiveDoneSec), Owner: st.Policy.Owner},
	}), nil
}

// Beat — хоёр чиглэлт урсгал (gRPC клиентэд): тайлан бүрт нийт идэвхтэй секундийг буцаана.
func (s *Server) Beat(ctx context.Context, stream *connect.BidiStream[pb.BeatRequest, pb.BeatResponse]) error {
	p, err := s.auth(stream.RequestHeader())
	if err != nil {
		return err
	}
	for {
		in, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		done, err := s.api.RecordBeat(ctx, p.uid, beatInput(in))
		if err != nil {
			return toConnect(err)
		}
		if err := stream.Send(&pb.BeatResponse{Ok: true, ActiveDoneSec: int32(done)}); err != nil {
			return err
		}
	}
}

// BeatOnce — нэг удаагийн тайлан (хөтөч fetch-ээр дуудна).
func (s *Server) BeatOnce(ctx context.Context, req *connect.Request[pb.BeatRequest]) (*connect.Response[pb.BeatResponse], error) {
	p, err := s.auth(req.Header())
	if err != nil {
		return nil, err
	}
	done, err := s.api.RecordBeat(ctx, p.uid, beatInput(req.Msg))
	if err != nil {
		return nil, toConnect(err)
	}
	return connect.NewResponse(&pb.BeatResponse{Ok: true, ActiveDoneSec: int32(done)}), nil
}

// ---- багш ----

func (s *Server) GetAnalytics(ctx context.Context, req *connect.Request[pb.AnalyticsRequest]) (*connect.Response[pb.AnalyticsResponse], error) {
	p, err := s.auth(req.Header())
	if err != nil {
		return nil, err
	}
	if p.role != string(store.RoleTeacher) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("зөвхөн багш"))
	}
	data, err := s.api.Analytics(ctx, p.uid, req.Msg.GetCourseId(), int(req.Msg.GetDays()))
	if err != nil {
		return nil, toConnect(err)
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
	return connect.NewResponse(out), nil
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
