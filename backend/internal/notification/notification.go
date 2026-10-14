// Package notification is the Notification Adapter: it consumes domain events,
// stores in-app notifications and hands them to the (simulated) provider.
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"condo/pkg/events"
	"condo/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Notifications live in MongoDB (document store): they are write-heavy, schema-flexible
// per event type, and only ever read back per user.
type Service struct {
	notifs      *mongo.Collection
	identityURL string
	http        *http.Client
}

func New(coll *mongo.Collection, identityURL string) *Service {
	return &Service{notifs: coll, identityURL: identityURL, http: &http.Client{Timeout: 5 * time.Second}}
}

// EnsureIndexes creates the unique (event, user) index that makes event handling
// idempotent, and the per-user inbox index.
func (s *Service) EnsureIndexes(ctx context.Context) error {
	_, err := s.notifs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "event_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	})
	return err
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
		_, err := s.notifs.InsertOne(ctx, Notification{
			ID: uuid.NewString(), EventID: env.ID, UserID: uid, Type: msg.Type, Title: msg.Title,
			Body: msg.Body, RefType: msg.RefType, RefID: msg.RefID, CreatedAt: time.Now().UTC(),
		})
		if mongo.IsDuplicateKeyError(err) {
			continue // event already processed for this user
		}
		if err != nil {
			return err
		}
		// Simulated Notification Provider (external system).
		slog.Info("notification provider: send", "to", uid, "type", msg.Type, "title", msg.Title, "body", msg.Body)
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

// Notification is the MongoDB document and the REST representation.
type Notification struct {
	ID        string     `json:"id" bson:"_id"`
	EventID   string     `json:"-" bson:"event_id"`
	UserID    string     `json:"-" bson:"user_id"`
	Type      string     `json:"type" bson:"type"`
	Title     string     `json:"title" bson:"title"`
	Body      string     `json:"body" bson:"body"`
	RefType   string     `json:"refType" bson:"ref_type"`
	RefID     string     `json:"refId" bson:"ref_id"`
	ReadAt    *time.Time `json:"readAt,omitempty" bson:"read_at,omitempty"`
	CreatedAt time.Time  `json:"createdAt" bson:"created_at"`
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
	cur, err := s.notifs.Find(r.Context(), bson.M{"user_id": uid},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(100))
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	out := []Notification{}
	if err := cur.All(r.Context(), &out); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	httpx.JSON(w, 200, out)
}

func (s *Service) markRead(w http.ResponseWriter, r *http.Request) {
	uid := httpx.IdentityFrom(r).UserID
	id := chi.URLParam(r, "id")
	own := bson.M{"_id": id, "user_id": uid}
	res, err := s.notifs.UpdateOne(r.Context(), bson.M{"_id": id, "user_id": uid, "read_at": nil},
		bson.M{"$set": bson.M{"read_at": time.Now().UTC()}})
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	if res.MatchedCount == 0 {
		// Either already read (fine) or not this user's notification (404).
		n, err := s.notifs.CountDocuments(r.Context(), own)
		if err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		if n == 0 {
			httpx.Error(w, 404, "notification not found")
			return
		}
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}

func (s *Service) markAllRead(w http.ResponseWriter, r *http.Request) {
	uid := httpx.IdentityFrom(r).UserID
	if _, err := s.notifs.UpdateMany(r.Context(), bson.M{"user_id": uid, "read_at": nil},
		bson.M{"$set": bson.M{"read_at": time.Now().UTC()}}); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}
