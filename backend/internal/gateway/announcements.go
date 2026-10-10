package gateway

import (
	"context"
	"net/http"
	"time"

	pb "condo/gen/announcement/v1"
	"condo/pkg/auth"
	"condo/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HTTPStatus maps a gRPC status code to the HTTP status returned to REST clients.
func HTTPStatus(c codes.Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	}
	return http.StatusInternalServerError
}

func grpcError(w http.ResponseWriter, err error) {
	st, _ := status.FromError(err)
	code := HTTPStatus(st.Code())
	msg := st.Message()
	if code >= 500 && st.Code() != codes.Unavailable {
		msg = "internal error"
	}
	if st.Code() == codes.Unavailable {
		msg = "announcement service unavailable"
	}
	httpx.Error(w, code, msg)
}

func callCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}

// announcementJSON is the REST representation of pb.Announcement.
type announcementJSON struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Body             string `json:"body"`
	Category         string `json:"category"`
	Status           string `json:"status"`
	TargetType       string `json:"targetType"`
	TargetBuildingID string `json:"targetBuildingId,omitempty"`
	TargetFloor      int32  `json:"targetFloor,omitempty"`
	PublishAt        string `json:"publishAt"`
	CreatedBy        string `json:"createdBy"`
	UpdatedAt        string `json:"updatedAt"`
	Read             bool   `json:"read"`
}

func toJSON(a *pb.Announcement) announcementJSON {
	return announcementJSON{a.Id, a.Title, a.Body, a.Category, a.Status, a.TargetType, a.TargetBuildingId, a.TargetFloor, a.PublishAt, a.CreatedBy, a.UpdatedAt, a.Read}
}

type announcementIn struct {
	Title            string `json:"title"`
	Body             string `json:"body"`
	Category         string `json:"category"`
	TargetType       string `json:"targetType"`
	TargetBuildingID string `json:"targetBuildingId"`
	TargetFloor      int32  `json:"targetFloor"`
	PublishAt        string `json:"publishAt"`
}

func (g *Gateway) publishAnnouncement(w http.ResponseWriter, r *http.Request) {
	var in announcementIn
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	ctx, cancel := callCtx(r)
	defer cancel()
	a, err := g.ann.Publish(ctx, &pb.PublishRequest{
		Title: in.Title, Body: in.Body, Category: in.Category, TargetType: in.TargetType,
		TargetBuildingId: in.TargetBuildingID, TargetFloor: in.TargetFloor, PublishAt: in.PublishAt,
		CreatedBy: httpx.IdentityFrom(r).UserID,
	})
	if err != nil {
		grpcError(w, err)
		return
	}
	httpx.JSON(w, 201, toJSON(a))
}

func (g *Gateway) updateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var in announcementIn
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	ctx, cancel := callCtx(r)
	defer cancel()
	a, err := g.ann.Update(ctx, &pb.UpdateRequest{
		Id: chi.URLParam(r, "id"), Title: in.Title, Body: in.Body, Category: in.Category,
		TargetType: in.TargetType, TargetBuildingId: in.TargetBuildingID, TargetFloor: in.TargetFloor,
	})
	if err != nil {
		grpcError(w, err)
		return
	}
	httpx.JSON(w, 200, toJSON(a))
}

func (g *Gateway) archiveAnnouncement(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := callCtx(r)
	defer cancel()
	a, err := g.ann.Archive(ctx, &pb.ArchiveRequest{Id: chi.URLParam(r, "id")})
	if err != nil {
		grpcError(w, err)
		return
	}
	httpx.JSON(w, 200, toJSON(a))
}

func (g *Gateway) listAnnouncements(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	ctx, cancel := callCtx(r)
	defer cancel()
	res, err := g.ann.List(ctx, &pb.ListRequest{All: id.Role == auth.RoleStaff, UserId: id.UserID, BuildingId: id.BuildingID, Floor: int32(id.Floor)})
	if err != nil {
		grpcError(w, err)
		return
	}
	out := make([]announcementJSON, 0, len(res.Announcements))
	for _, a := range res.Announcements {
		out = append(out, toJSON(a))
	}
	httpx.JSON(w, 200, out)
}

func (g *Gateway) getAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	ctx, cancel := callCtx(r)
	defer cancel()
	a, err := g.ann.Get(ctx, &pb.GetRequest{Id: chi.URLParam(r, "id"), UserId: id.UserID, All: id.Role == auth.RoleStaff, BuildingId: id.BuildingID, Floor: int32(id.Floor)})
	if err != nil {
		grpcError(w, err)
		return
	}
	httpx.JSON(w, 200, toJSON(a))
}

func (g *Gateway) markAnnouncementRead(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	ctx, cancel := callCtx(r)
	defer cancel()
	// Ensure the caller may see the announcement before recording a read.
	if _, err := g.ann.Get(ctx, &pb.GetRequest{Id: chi.URLParam(r, "id"), UserId: id.UserID, All: id.Role == auth.RoleStaff, BuildingId: id.BuildingID, Floor: int32(id.Floor)}); err != nil {
		grpcError(w, err)
		return
	}
	if _, err := g.ann.MarkRead(ctx, &pb.MarkReadRequest{Id: chi.URLParam(r, "id"), UserId: id.UserID}); err != nil {
		grpcError(w, err)
		return
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}
