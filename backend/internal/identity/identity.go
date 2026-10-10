// Package identity owns users, buildings and rooms, and issues JWTs.
package identity

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"condo/pkg/auth"
	"condo/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var Schema string

type Service struct {
	db        *pgxpool.Pool
	jwtSecret string
}

func New(db *pgxpool.Pool, jwtSecret string) *Service {
	return &Service{db: db, jwtSecret: jwtSecret}
}

type userRow struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	RoomID     string `json:"roomId,omitempty"`
	RoomNumber string `json:"roomNumber,omitempty"`
	BuildingID string `json:"buildingId,omitempty"`
	Building   string `json:"building,omitempty"`
	Floor      int    `json:"floor,omitempty"`
}

const userSelect = `
SELECT u.id, u.email, u.name, u.role,
       COALESCE(r.id::text,''), COALESCE(r.number,''), COALESCE(b.id::text,''), COALESCE(b.name,''), COALESCE(r.floor,0)
FROM users u
LEFT JOIN rooms r ON r.id = u.room_id
LEFT JOIN buildings b ON b.id = r.building_id`

func scanUser(row pgx.Row, u *userRow) error {
	return row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.RoomID, &u.RoomNumber, &u.BuildingID, &u.Building, &u.Floor)
}

func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Post("/auth/login", s.login)
	r.Get("/users/me", s.me)
	r.Method(http.MethodGet, "/rooms", httpx.RequireRoleFunc(auth.RoleStaff, s.listRooms))
	r.Get("/internal/recipients", s.recipients)
	return r
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	var u userRow
	var hash string
	row := s.db.QueryRow(r.Context(), `SELECT u.password_hash FROM users u WHERE u.email=$1`, strings.ToLower(strings.TrimSpace(in.Email)))
	if err := row.Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		httpx.Error(w, 401, "invalid email or password")
		return
	}
	if err := scanUser(s.db.QueryRow(r.Context(), userSelect+` WHERE u.email=$1`, strings.ToLower(strings.TrimSpace(in.Email))), &u); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	token, err := auth.Sign(s.jwtSecret, auth.Claims{UserID: u.ID, Role: u.Role, RoomID: u.RoomID, BuildingID: u.BuildingID, Floor: u.Floor}, 12*time.Hour)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	httpx.JSON(w, 200, map[string]any{"token": token, "user": u})
}

func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	var u userRow
	if err := scanUser(s.db.QueryRow(r.Context(), userSelect+` WHERE u.id=$1`, id.UserID), &u); err != nil {
		httpx.Error(w, 404, "user not found")
		return
	}
	httpx.JSON(w, 200, u)
}

func (s *Service) listRooms(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
SELECT r.id, r.number, r.floor, b.id, b.name FROM rooms r JOIN buildings b ON b.id=r.building_id ORDER BY b.name, r.floor, r.number`)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer rows.Close()
	type room struct {
		ID         string `json:"id"`
		Number     string `json:"number"`
		Floor      int    `json:"floor"`
		BuildingID string `json:"buildingId"`
		Building   string `json:"building"`
	}
	out := []room{}
	for rows.Next() {
		var x room
		if err := rows.Scan(&x.ID, &x.Number, &x.Floor, &x.BuildingID, &x.Building); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, 200, out)
}

// recipients resolves user IDs for notification targeting. Internal only.
// Query: roomId=... | role=staff | all=true | buildingId=... [&floor=N]
func (s *Service) recipients(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var (
		sql  = `SELECT u.id FROM users u LEFT JOIN rooms r ON r.id=u.room_id WHERE `
		args []any
	)
	switch {
	case q.Get("roomId") != "":
		sql += `u.room_id=$1 AND u.role='resident'`
		args = append(args, q.Get("roomId"))
	case q.Get("role") == "staff":
		sql += `u.role='staff'`
	case q.Get("buildingId") != "":
		sql += `u.role='resident' AND r.building_id=$1`
		args = append(args, q.Get("buildingId"))
		if f := q.Get("floor"); f != "" && f != "0" {
			sql += ` AND r.floor=$2`
			args = append(args, f)
		}
	case q.Get("all") == "true":
		sql += `u.role='resident'`
	default:
		httpx.Error(w, 400, "missing selector")
		return
	}
	rows, err := s.db.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		ids = append(ids, id)
	}
	httpx.JSON(w, 200, map[string]any{"userIds": ids})
}

// Seed creates demo data when the users table is empty.
func (s *Service) Seed(ctx context.Context) error {
	var n int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	type seeded struct{ roomID string }
	var rooms []seeded
	for _, bname := range []string{"A", "B"} {
		bid := uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO buildings(id,name) VALUES($1,$2)`, bid, "Building "+bname); err != nil {
			return err
		}
		for floor := 1; floor <= 2; floor++ {
			for i := 1; i <= 3; i++ {
				rid := uuid.NewString()
				num := fmt.Sprintf("%s%d0%d", bname, floor, i)
				if _, err := tx.Exec(ctx, `INSERT INTO rooms(id,building_id,floor,number) VALUES($1,$2,$3,$4)`, rid, bid, floor, num); err != nil {
					return err
				}
				rooms = append(rooms, seeded{rid})
			}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO users(id,email,password_hash,name,role) VALUES($1,'staff@condo.local',$2,'Juristic Staff','staff')`, uuid.NewString(), string(hash)); err != nil {
		return err
	}
	// resident1..N: first room of building A, A102.., then building B rooms
	for i, rm := range rooms {
		if _, err := tx.Exec(ctx, `INSERT INTO users(id,email,password_hash,name,role,room_id) VALUES($1,$2,$3,$4,'resident',$5)`,
			uuid.NewString(), fmt.Sprintf("resident%d@condo.local", i+1), string(hash), fmt.Sprintf("Resident %d", i+1), rm.roomID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("seeded demo data", "rooms", len(rooms))
	return nil
}

var ErrNotFound = errors.New("not found")
