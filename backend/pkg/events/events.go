// Package events implements the event-driven notification backbone (ADR-03)
// on top of a RabbitMQ topic exchange.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "condo.events"

// Routing keys / event types.
const (
	ParcelReceived        = "parcel.received"
	ParcelAccepted        = "parcel.accepted"
	AnnouncementPublished = "announcement.published"
	UtilityBillIssued     = "billing.bill_issued"
	PaymentCompleted      = "billing.payment_completed"
)

type Envelope struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
}

type ParcelPayload struct {
	ParcelID   string `json:"parcelId"`
	RoomID     string `json:"roomId"`
	PickedUpBy string `json:"pickedUpBy,omitempty"`
	Carrier    string `json:"carrier,omitempty"`
}

type AnnouncementPayload struct {
	AnnouncementID string `json:"announcementId"`
	Title          string `json:"title"`
	TargetType     string `json:"targetType"`
	BuildingID     string `json:"buildingId,omitempty"`
	Floor          int    `json:"floor,omitempty"`
}

type BillPayload struct {
	BillID  string  `json:"billId"`
	RoomID  string  `json:"roomId"`
	Cycle   string  `json:"cycle,omitempty"`
	Total   float64 `json:"total"`
	DueDate string  `json:"dueDate,omitempty"`
}

// Publisher lazily connects and reconnects to RabbitMQ. Publish failures are
// returned to the caller, which logs them (best-effort delivery).
type Publisher struct {
	url string
	mu  sync.Mutex
	con *amqp.Connection
	ch  *amqp.Channel
}

func NewPublisher(url string) *Publisher { return &Publisher{url: url} }

func (p *Publisher) ensure() error {
	if p.ch != nil && !p.ch.IsClosed() {
		return nil
	}
	con, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}
	ch, err := con.Channel()
	if err != nil {
		con.Close()
		return err
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		con.Close()
		return err
	}
	p.con, p.ch = con, ch
	return nil
}

func (p *Publisher) Publish(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env, err := json.Marshal(Envelope{ID: uuid.NewString(), Type: eventType, OccurredAt: time.Now().UTC(), Payload: body})
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensure(); err != nil {
		return fmt.Errorf("amqp connect: %w", err)
	}
	return p.ch.PublishWithContext(ctx, Exchange, eventType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         env,
	})
}

// PublishLogged publishes and logs (but does not propagate) failures.
func (p *Publisher) PublishLogged(ctx context.Context, eventType string, payload any) {
	if err := p.Publish(ctx, eventType, payload); err != nil {
		slog.Error("publish event failed", "type", eventType, "err", err)
	}
}

// Consume binds a durable queue to the given routing keys and invokes handle
// for each envelope. A handler error causes a requeue; success acks.
// It reconnects automatically until ctx is cancelled.
func Consume(ctx context.Context, url, queue string, keys []string, handle func(context.Context, Envelope) error) {
	for ctx.Err() == nil {
		if err := consumeOnce(ctx, url, queue, keys, handle); err != nil {
			slog.Warn("consumer stopped, retrying", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func consumeOnce(ctx context.Context, url, queue string, keys []string, handle func(context.Context, Envelope) error) error {
	con, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	defer con.Close()
	ch, err := con.Channel()
	if err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	for _, k := range keys {
		if err := ch.QueueBind(queue, k, Exchange, false, nil); err != nil {
			return err
		}
	}
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	closed := con.NotifyClose(make(chan *amqp.Error, 1))
	slog.Info("consuming events", "queue", queue, "keys", keys)
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-closed:
			return fmt.Errorf("connection closed: %v", e)
		case m, ok := <-msgs:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}
			var env Envelope
			if err := json.Unmarshal(m.Body, &env); err != nil {
				slog.Error("bad event, dropping", "err", err)
				_ = m.Nack(false, false)
				continue
			}
			if err := handle(ctx, env); err != nil {
				slog.Error("event handler failed, requeue", "type", env.Type, "id", env.ID, "err", err)
				time.Sleep(time.Second)
				_ = m.Nack(false, true)
				continue
			}
			_ = m.Ack(false)
		}
	}
}
