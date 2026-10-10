// Package announcement implements the Announcement Service, exposed over gRPC (ADR-02).
package announcement

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"time"

	pb "condo/gen/announcement/v1"
	"condo/pkg/events"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:embed schema.sql
var Schema string

type Server struct {
	pb.UnimplementedAnnouncementServiceServer
	db  *pgxpool.Pool
	pub *events.Publisher
}

func New(db *pgxpool.Pool, pub *events.Publisher) *Server { return &Server{db: db, pub: pub} }

const cols = `a.id, a.title, a.body, a.category, a.status, a.target_type, a.target_building_id, a.target_floor, a.publish_at, a.created_by, a.updated_at`

// Visible reports whether a resident in (building, floor) may see an announcement.
func Visible(targetType, targetBuilding string, targetFloor int32, building string, floor int32) bool {
	switch targetType {
	case "ALL":
		return true
	case "BUILDING":
		return building != "" && building == targetBuilding
	case "FLOOR":
		return building != "" && building == targetBuilding && floor == targetFloor
	}
	return false
}

// validateTarget checks target consistency.
func validateTarget(targetType, building string, floor int32) error {
	switch targetType {
	case "ALL":
		return nil
	case "BUILDING":
		if building == "" {
			return status.Error(codes.InvalidArgument, "target_building_id is required for BUILDING target")
		}
		return nil
	case "FLOOR":
		if building == "" || floor <= 0 {
			return status.Error(codes.InvalidArgument, "target_building_id and target_floor are required for FLOOR target")
		}
		return nil
	}
	return status.Error(codes.InvalidArgument, "target_type must be ALL, BUILDING or FLOOR")
}

func scan(row pgx.Row, a *pb.Announcement) error {
	var publishAt, updatedAt time.Time
	if err := row.Scan(&a.Id, &a.Title, &a.Body, &a.Category, &a.Status, &a.TargetType, &a.TargetBuildingId, &a.TargetFloor, &publishAt, &a.CreatedBy, &updatedAt); err != nil {
		return err
	}
	a.PublishAt = publishAt.UTC().Format(time.RFC3339)
	a.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return nil
}

func internalErr(err error) error {
	slog.Error("announcement service", "err", err)
	return status.Error(codes.Internal, "internal error")
}

func (s *Server) Publish(ctx context.Context, in *pb.PublishRequest) (*pb.Announcement, error) {
	if in.Title == "" || in.Body == "" {
		return nil, status.Error(codes.InvalidArgument, "title and body are required")
	}
	if in.TargetType == "" {
		in.TargetType = "ALL"
	}
	if err := validateTarget(in.TargetType, in.TargetBuildingId, in.TargetFloor); err != nil {
		return nil, err
	}
	if in.Category == "" {
		in.Category = "GENERAL"
	}
	publishAt := time.Now()
	if in.PublishAt != "" {
		t, err := time.Parse(time.RFC3339, in.PublishAt)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "publish_at must be RFC3339")
		}
		publishAt = t
	}
	var a pb.Announcement
	err := scan(s.db.QueryRow(ctx, `
INSERT INTO announcements(id,title,body,category,status,target_type,target_building_id,target_floor,publish_at,created_by)
VALUES($1,$2,$3,$4,'PUBLISHED',$5,$6,$7,$8,$9)
RETURNING id, title, body, category, status, target_type, target_building_id, target_floor, publish_at, created_by, updated_at`,
		uuid.NewString(), in.Title, in.Body, in.Category, in.TargetType, in.TargetBuildingId, in.TargetFloor, publishAt, in.CreatedBy), &a)
	if err != nil {
		return nil, internalErr(err)
	}
	s.pub.PublishLogged(ctx, events.AnnouncementPublished, events.AnnouncementPayload{
		AnnouncementID: a.Id, Title: a.Title, TargetType: a.TargetType, BuildingID: a.TargetBuildingId, Floor: int(a.TargetFloor),
	})
	return &a, nil
}

func (s *Server) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.Announcement, error) {
	if in.Id == "" || in.Title == "" || in.Body == "" {
		return nil, status.Error(codes.InvalidArgument, "id, title and body are required")
	}
	if in.TargetType == "" {
		in.TargetType = "ALL"
	}
	if err := validateTarget(in.TargetType, in.TargetBuildingId, in.TargetFloor); err != nil {
		return nil, err
	}
	var a pb.Announcement
	err := scan(s.db.QueryRow(ctx, `
UPDATE announcements SET title=$2, body=$3, category=COALESCE(NULLIF($4,''),category), target_type=$5, target_building_id=$6, target_floor=$7, updated_at=now()
WHERE id=$1 AND status<>'ARCHIVED'
RETURNING id, title, body, category, status, target_type, target_building_id, target_floor, publish_at, created_by, updated_at`,
		in.Id, in.Title, in.Body, in.Category, in.TargetType, in.TargetBuildingId, in.TargetFloor), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "announcement not found or archived")
	}
	if err != nil {
		return nil, internalErr(err)
	}
	return &a, nil
}

func (s *Server) Archive(ctx context.Context, in *pb.ArchiveRequest) (*pb.Announcement, error) {
	var a pb.Announcement
	err := scan(s.db.QueryRow(ctx, `
UPDATE announcements SET status='ARCHIVED', updated_at=now() WHERE id=$1
RETURNING id, title, body, category, status, target_type, target_building_id, target_floor, publish_at, created_by, updated_at`, in.Id), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "announcement not found")
	}
	if err != nil {
		return nil, internalErr(err)
	}
	return &a, nil
}

func (s *Server) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	q := `SELECT ` + cols + `, EXISTS(SELECT 1 FROM announcement_reads r WHERE r.announcement_id=a.id AND r.user_id=$1) FROM announcements a `
	args := []any{in.UserId}
	if in.All {
		q += `ORDER BY a.updated_at DESC LIMIT 200`
	} else {
		q += `WHERE a.status='PUBLISHED' AND a.publish_at <= now() AND (
  a.target_type='ALL'
  OR (a.target_type='BUILDING' AND a.target_building_id=$2)
  OR (a.target_type='FLOOR' AND a.target_building_id=$2 AND a.target_floor=$3)
) ORDER BY a.publish_at DESC LIMIT 200`
		args = append(args, in.BuildingId, in.Floor)
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, internalErr(err)
	}
	defer rows.Close()
	out := &pb.ListResponse{}
	for rows.Next() {
		var a pb.Announcement
		var publishAt, updatedAt time.Time
		if err := rows.Scan(&a.Id, &a.Title, &a.Body, &a.Category, &a.Status, &a.TargetType, &a.TargetBuildingId, &a.TargetFloor, &publishAt, &a.CreatedBy, &updatedAt, &a.Read); err != nil {
			return nil, internalErr(err)
		}
		a.PublishAt = publishAt.UTC().Format(time.RFC3339)
		a.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out.Announcements = append(out.Announcements, &a)
	}
	return out, nil
}

func (s *Server) Get(ctx context.Context, in *pb.GetRequest) (*pb.Announcement, error) {
	if _, err := uuid.Parse(in.Id); err != nil {
		return nil, status.Error(codes.NotFound, "announcement not found")
	}
	var a pb.Announcement
	err := scan(s.db.QueryRow(ctx, `SELECT `+cols+` FROM announcements a WHERE a.id=$1`, in.Id), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "announcement not found")
	}
	if err != nil {
		return nil, internalErr(err)
	}
	if !in.All {
		pub, perr := time.Parse(time.RFC3339, a.PublishAt)
		if a.Status != "PUBLISHED" || (perr == nil && pub.After(time.Now())) || !Visible(a.TargetType, a.TargetBuildingId, a.TargetFloor, in.BuildingId, in.Floor) {
			return nil, status.Error(codes.NotFound, "announcement not found")
		}
	}
	_ = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM announcement_reads WHERE announcement_id=$1 AND user_id=$2)`, in.Id, in.UserId).Scan(&a.Read)
	return &a, nil
}

func (s *Server) MarkRead(ctx context.Context, in *pb.MarkReadRequest) (*pb.MarkReadResponse, error) {
	if _, err := uuid.Parse(in.Id); err != nil {
		return nil, status.Error(codes.NotFound, "announcement not found")
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO announcement_reads(announcement_id,user_id) SELECT id,$2 FROM announcements WHERE id=$1 ON CONFLICT DO NOTHING`, in.Id, in.UserId)
	if err != nil {
		return nil, internalErr(err)
	}
	return &pb.MarkReadResponse{}, nil
}
