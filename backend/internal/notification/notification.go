// Package notification is the Notification Adapter: it consumes domain events,
// stores in-app notifications and hands them to the (simulated) provider.
package notification

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"condo/pkg/events"
	"condo/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var Schema string

type Service struct {
	db          *pgxpool.Pool
	identityURL string
	http        *http.Client
}

func New(db *pgxpool.Pool, identityURL string) *Service {
	return &Service{db: db, identityURL: identityURL, http: &http.Client{Timeout: 5 * time.Second}}
}

var EventKeys = []string{events.ParcelReceived, events.ParcelAccepted, events.AnnouncementPublished, events.UtilityBillIssued, events.PaymentCompleted}

// Handle processes one event. It is idempotent on (event id, user).
func (s *Service) Handle(ctx context.Context, env events.Envelope) error {
	msg, ok, err := Render(env)
	if err != nil {
		slog.Error("bad event payload, dropping", "type", env.Type, "err", err)
		return nil
	}
	if !ok {
		return nil
	}
	users, err := s.resolve(ctx, msg)
	if err != nil {
		return err
	}
	for _, uid := range users {
		tag, err := s.db.Exec(ctx, `
INSERT INTO notifications(id,event_id,user_id,type,title,body,ref_type,ref_id)
VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (event_id,user_id) DO NOTHING`,
			uuid.NewString(), env.ID, uid, msg.Type, msg.Title, msg.Body, msg.RefType, msg.RefID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			// Simulated Notification Provider (external system).
			slog.Info("notification provider: send", "to", uid, "type", msg.Type, "title", msg.Title, "body", msg.Body)
		}
	}
	return nil
}

func (s *Service) resolve(ctx context.Context, m Message) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.identityURL+"/internal/recipients?"+m.Recipients.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("identity recipients: status %d", res.StatusCode)
	}
	var out struct {
		UserIDs []string `json:"userIds"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.UserIDs, nil
}

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	RefType   string     `json:"refType"`
	RefID     string     `json:"refId"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Get("/notifications", s.list)
	r.Patch("/notifications/{id}/read", s.markRead)
	r.Post("/notifications/read-all", s.markAllRead)
	return r
}

func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	uid := httpx.IdentityFrom(r).UserID
	rows, err := s.db.Query(r.Context(), `
SELECT id,type,title,body,ref_type,ref_id,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100`, uid)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.RefType, &n.RefID, &n.ReadAt, &n.CreatedAt); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		out = append(out, n)
	}
	httpx.JSON(w, 200, out)
}

func (s *Service) markRead(w http.ResponseWriter, r *http.Request) {
	uid := httpx.IdentityFrom(r).UserID
	tag, err := s.db.Exec(r.Context(), `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND user_id=$2`, chi.URLParam(r, "id"), uid)
	if err != nil {
		httpx.Error(w, 400, "invalid id")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, 404, "notification not found")
		return
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}

func (s *Service) markAllRead(w http.ResponseWriter, r *http.Request) {
	uid := httpx.IdentityFrom(r).UserID
	if _, err := s.db.Exec(r.Context(), `UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL`, uid); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}
