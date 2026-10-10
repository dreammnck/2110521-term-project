// Package parcel implements Parcel Management (use case 1).
package parcel

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"condo/pkg/auth"
	"condo/pkg/events"
	"condo/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var Schema string

const maxPhotoBytes = 5 << 20

type Service struct {
	db       *pgxpool.Pool
	pub      *events.Publisher
	photoDir string
}

func New(db *pgxpool.Pool, pub *events.Publisher, photoDir string) *Service {
	return &Service{db: db, pub: pub, photoDir: photoDir}
}

type Parcel struct {
	ID           string     `json:"id"`
	RoomID       string     `json:"roomId"`
	Carrier      string     `json:"carrier"`
	TrackingNo   string     `json:"trackingNo"`
	Note         string     `json:"note"`
	HasPhoto     bool       `json:"hasPhoto"`
	Status       string     `json:"status"`
	ReceivedAt   time.Time  `json:"receivedAt"`
	PickedUpAt   *time.Time `json:"pickedUpAt,omitempty"`
	PickedUpBy   *string    `json:"pickedUpBy,omitempty"`
	RegisteredBy string     `json:"registeredBy"`
}

const cols = `id, room_id, carrier, tracking_no, note, photo_path <> '', status, received_at, picked_up_at, picked_up_by::text, registered_by::text`

func scan(row pgx.Row, p *Parcel) error {
	return row.Scan(&p.ID, &p.RoomID, &p.Carrier, &p.TrackingNo, &p.Note, &p.HasPhoto, &p.Status, &p.ReceivedAt, &p.PickedUpAt, &p.PickedUpBy, &p.RegisteredBy)
}

func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Method(http.MethodPost, "/parcels", httpx.RequireRoleFunc(auth.RoleStaff, s.register))
	r.Method(http.MethodGet, "/parcels", httpx.RequireRoleFunc(auth.RoleStaff, s.listPending))
	r.Method(http.MethodGet, "/parcels/me", httpx.RequireRoleFunc(auth.RoleResident, s.listMine))
	r.Method(http.MethodPost, "/parcels/{id}/pickup", httpx.RequireRoleFunc(auth.RoleResident, s.confirmPickup))
	r.Get("/parcels/{id}/photo", s.photo)
	return r
}

func (s *Service) register(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+(1<<20))
	if err := r.ParseMultipartForm(maxPhotoBytes); err != nil {
		httpx.Error(w, 400, "invalid multipart form")
		return
	}
	roomID := strings.TrimSpace(r.FormValue("roomId"))
	if _, err := uuid.Parse(roomID); err != nil {
		httpx.Error(w, 400, "roomId is required")
		return
	}
	pid := uuid.NewString()
	photoPath := ""
	if f, hdr, err := r.FormFile("photo"); err == nil {
		defer f.Close()
		ext := strings.ToLower(filepath.Ext(hdr.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			httpx.Error(w, 400, "photo must be jpg, png or webp")
			return
		}
		if err := os.MkdirAll(s.photoDir, 0o755); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		photoPath = pid + ext
		out, err := os.Create(filepath.Join(s.photoDir, photoPath))
		if err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		_, err = io.Copy(out, io.LimitReader(f, maxPhotoBytes))
		out.Close()
		if err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
	}
	var p Parcel
	err := scan(s.db.QueryRow(r.Context(), `
INSERT INTO parcels(id, room_id, carrier, tracking_no, note, photo_path, registered_by)
VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+cols,
		pid, roomID, r.FormValue("carrier"), r.FormValue("trackingNo"), r.FormValue("note"), photoPath, id.UserID), &p)
	if err != nil {
		slog.Error("register parcel", "err", err)
		httpx.Error(w, 500, "internal error")
		return
	}
	s.pub.PublishLogged(r.Context(), events.ParcelReceived, events.ParcelPayload{ParcelID: p.ID, RoomID: p.RoomID, Carrier: p.Carrier})
	httpx.JSON(w, 201, p)
}

func (s *Service) query(ctx context.Context, w http.ResponseWriter, sql string, args ...any) {
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer rows.Close()
	out := []Parcel{}
	for rows.Next() {
		var p Parcel
		if err := scan(rows, &p); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, 200, out)
}

// listPending lists parcels for staff; status=WAITING (default) or PICKED_UP or all.
func (s *Service) listPending(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(r.URL.Query().Get("status"))
	if status == "" {
		status = "WAITING"
	}
	if status == "ALL" {
		s.query(r.Context(), w, `SELECT `+cols+` FROM parcels ORDER BY received_at DESC LIMIT 200`)
		return
	}
	if status != "WAITING" && status != "PICKED_UP" {
		httpx.Error(w, 400, "invalid status")
		return
	}
	s.query(r.Context(), w, `SELECT `+cols+` FROM parcels WHERE status=$1 ORDER BY received_at DESC LIMIT 200`, status)
}

func (s *Service) listMine(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	if id.RoomID == "" {
		httpx.JSON(w, 200, []Parcel{})
		return
	}
	s.query(r.Context(), w, `SELECT `+cols+` FROM parcels WHERE room_id=$1 ORDER BY (status='WAITING') DESC, received_at DESC`, id.RoomID)
}

func (s *Service) confirmPickup(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	pid := chi.URLParam(r, "id")
	if _, err := uuid.Parse(pid); err != nil {
		httpx.Error(w, 400, "invalid id")
		return
	}
	var p Parcel
	err := scan(s.db.QueryRow(r.Context(), `
UPDATE parcels SET status='PICKED_UP', picked_up_at=now(), picked_up_by=$3
WHERE id=$1 AND room_id=$2 AND status='WAITING' RETURNING `+cols, pid, nullIfEmpty(id.RoomID), id.UserID), &p)
	if errors.Is(err, pgx.ErrNoRows) {
		// Distinguish not-yours/not-found from already picked up.
		var st string
		var room string
		if e := s.db.QueryRow(r.Context(), `SELECT status, room_id::text FROM parcels WHERE id=$1`, pid).Scan(&st, &room); e != nil || room != id.RoomID {
			httpx.Error(w, 404, "parcel not found")
			return
		}
		httpx.Error(w, 409, "parcel already picked up")
		return
	}
	if err != nil {
		slog.Error("confirm pickup", "err", err)
		httpx.Error(w, 500, "internal error")
		return
	}
	s.pub.PublishLogged(r.Context(), events.ParcelAccepted, events.ParcelPayload{ParcelID: p.ID, RoomID: p.RoomID, PickedUpBy: id.UserID})
	httpx.JSON(w, 200, p)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Service) photo(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	var photo, room string
	err := s.db.QueryRow(r.Context(), `SELECT photo_path, room_id::text FROM parcels WHERE id=$1`, chi.URLParam(r, "id")).Scan(&photo, &room)
	if err != nil || photo == "" {
		httpx.Error(w, 404, "photo not found")
		return
	}
	if id.Role != auth.RoleStaff && id.RoomID != room {
		httpx.Error(w, 404, "photo not found")
		return
	}
	http.ServeFile(w, r, filepath.Join(s.photoDir, filepath.Base(photo)))
}
