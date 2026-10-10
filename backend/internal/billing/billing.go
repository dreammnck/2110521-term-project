// Package billing implements Utility Billing & Payment (use case 2).
package billing

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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

type Config struct {
	WaterRate, ElectricRate float64
	PaymentAdapterURL       string
	ReturnURL               string
}

type Service struct {
	db   *pgxpool.Pool
	pub  *events.Publisher
	cfg  Config
	http *http.Client
}

func New(db *pgxpool.Pool, pub *events.Publisher, cfg Config) *Service {
	return &Service{db: db, pub: pub, cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}}
}

type Bill struct {
	ID             string    `json:"id"`
	RoomID         string    `json:"roomId"`
	Cycle          string    `json:"cycle"`
	WaterUnits     float64   `json:"waterUnits"`
	WaterAmount    float64   `json:"waterAmount"`
	ElectricUnits  float64   `json:"electricUnits"`
	ElectricAmount float64   `json:"electricAmount"`
	Total          float64   `json:"total"`
	DueDate        string    `json:"dueDate"`
	Status         string    `json:"status"`
	IssuedAt       time.Time `json:"issuedAt"`
}

const billCols = `id, room_id::text, cycle, water_units::float8, water_amount::float8, electric_units::float8, electric_amount::float8, total::float8,
 to_char(due_date,'YYYY-MM-DD'),
 CASE WHEN status='UNPAID' AND due_date < current_date THEN 'OVERDUE' ELSE status END, issued_at`

func scanBill(row pgx.Row, b *Bill) error {
	return row.Scan(&b.ID, &b.RoomID, &b.Cycle, &b.WaterUnits, &b.WaterAmount, &b.ElectricUnits, &b.ElectricAmount, &b.Total, &b.DueDate, &b.Status, &b.IssuedAt)
}

func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Method(http.MethodPost, "/utilities/bills", httpx.RequireRoleFunc(auth.RoleStaff, s.issue))
	r.Method(http.MethodGet, "/utilities/bills", httpx.RequireRoleFunc(auth.RoleStaff, s.listAll))
	r.Method(http.MethodGet, "/utilities/bills/me", httpx.RequireRoleFunc(auth.RoleResident, s.listMine))
	r.Get("/utilities/bills/{id}", s.get)
	r.Method(http.MethodPost, "/payments", httpx.RequireRoleFunc(auth.RoleResident, s.createPayment))
	r.Get("/payments/{billId}/receipt", s.receipt)
	r.Post("/internal/payments/callback", s.callback) // internal only; never routed by the gateway
	return r
}

// issue creates one bill (body is BillInput) or a batch (body {"items":[...]}).
func (s *Service) issue(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := httpx.Decode(r, &raw); err != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	var batch struct {
		Items []BillInput `json:"items"`
	}
	_ = json.Unmarshal(raw, &batch)
	isBatch := batch.Items != nil
	items := batch.Items
	if !isBatch {
		var one BillInput
		if err := json.Unmarshal(raw, &one); err != nil {
			httpx.Error(w, 400, "invalid body")
			return
		}
		items = []BillInput{one}
	}
	if len(items) == 0 || len(items) > 500 {
		httpx.Error(w, 400, "between 1 and 500 bills per request")
		return
	}
	for i, it := range items {
		if err := it.Validate(); err != nil {
			httpx.Error(w, 400, fmt.Sprintf("item %d: %v", i, err))
			return
		}
	}
	created := []Bill{}
	skipped := []string{}
	for _, it := range items {
		wa, ea, total := Charges(it.WaterUnits, it.ElectricUnits, s.cfg.WaterRate, s.cfg.ElectricRate)
		var b Bill
		err := scanBill(s.db.QueryRow(r.Context(), `
INSERT INTO utility_bills(id,room_id,cycle,water_units,water_amount,electric_units,electric_amount,total,due_date)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (room_id,cycle) DO NOTHING RETURNING `+billCols,
			uuid.NewString(), it.RoomID, it.Cycle, it.WaterUnits, wa, it.ElectricUnits, ea, total, it.DueDate), &b)
		if errors.Is(err, pgx.ErrNoRows) {
			skipped = append(skipped, it.RoomID)
			continue
		}
		if err != nil {
			slog.Error("issue bill", "err", err)
			httpx.Error(w, 500, "internal error")
			return
		}
		created = append(created, b)
		s.pub.PublishLogged(r.Context(), events.UtilityBillIssued, events.BillPayload{BillID: b.ID, RoomID: b.RoomID, Cycle: b.Cycle, Total: b.Total, DueDate: b.DueDate})
	}
	if !isBatch {
		if len(created) == 0 {
			httpx.Error(w, 409, "a bill already exists for this room and cycle")
			return
		}
		httpx.JSON(w, 201, created[0])
		return
	}
	httpx.JSON(w, 201, map[string]any{"created": created, "skippedRoomIds": skipped})
}

func (s *Service) queryBills(ctx context.Context, w http.ResponseWriter, where string, args ...any) {
	rows, err := s.db.Query(ctx, `SELECT `+billCols+` FROM utility_bills `+where+` ORDER BY cycle DESC, issued_at DESC LIMIT 300`, args...)
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer rows.Close()
	out := []Bill{}
	for rows.Next() {
		var b Bill
		if err := scanBill(rows, &b); err != nil {
			httpx.Error(w, 500, "internal error")
			return
		}
		out = append(out, b)
	}
	httpx.JSON(w, 200, out)
}

// filterClause appends an effective-status / cycle filter.
func filterClause(r *http.Request, base string, args []any) (string, []any) {
	clauses := []string{base}
	if st := strings.ToUpper(r.URL.Query().Get("status")); st != "" {
		args = append(args, st)
		clauses = append(clauses, fmt.Sprintf(`(CASE WHEN status='UNPAID' AND due_date < current_date THEN 'OVERDUE' ELSE status END)=$%d`, len(args)))
	}
	if c := r.URL.Query().Get("cycle"); c != "" {
		args = append(args, c)
		clauses = append(clauses, fmt.Sprintf(`cycle=$%d`, len(args)))
	}
	if rm := r.URL.Query().Get("roomId"); rm != "" {
		args = append(args, rm)
		clauses = append(clauses, fmt.Sprintf(`room_id::text=$%d`, len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (s *Service) listAll(w http.ResponseWriter, r *http.Request) {
	where, args := filterClause(r, "TRUE", nil)
	s.queryBills(r.Context(), w, where, args...)
}

func (s *Service) listMine(w http.ResponseWriter, r *http.Request) {
	id := httpx.IdentityFrom(r)
	if id.RoomID == "" {
		httpx.JSON(w, 200, []Bill{})
		return
	}
	where, args := filterClause(r, "room_id::text=$1", []any{id.RoomID})
	s.queryBills(r.Context(), w, where, args...)
}

func (s *Service) loadBill(ctx context.Context, id httpx.Identity, billID string) (*Bill, int, string) {
	if _, err := uuid.Parse(billID); err != nil {
		return nil, 404, "bill not found"
	}
	var b Bill
	if err := scanBill(s.db.QueryRow(ctx, `SELECT `+billCols+` FROM utility_bills WHERE id=$1`, billID), &b); err != nil {
		return nil, 404, "bill not found"
	}
	if id.Role != auth.RoleStaff && b.RoomID != id.RoomID {
		return nil, 404, "bill not found"
	}
	return &b, 0, ""
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	b, code, msg := s.loadBill(r.Context(), httpx.IdentityFrom(r), chi.URLParam(r, "id"))
	if b == nil {
		httpx.Error(w, code, msg)
		return
	}
	httpx.JSON(w, 200, b)
}

// createPayment starts a payment through the payment adapter (ADR-05).
func (s *Service) createPayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BillID string `json:"billId"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	b, code, msg := s.loadBill(r.Context(), httpx.IdentityFrom(r), in.BillID)
	if b == nil {
		httpx.Error(w, code, msg)
		return
	}
	if b.Status == "PAID" {
		httpx.Error(w, 409, "bill is already paid")
		return
	}
	reqBody, _ := json.Marshal(map[string]any{"billId": b.ID, "amount": b.Total, "description": "Utility bill " + b.Cycle, "returnUrl": returnURLFor(s.cfg.ReturnURL, b.ID)})
	res, err := s.http.Post(s.cfg.PaymentAdapterURL+"/payments", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		slog.Error("payment adapter", "err", err)
		httpx.Error(w, 502, "payment provider unavailable, please try again later")
		return
	}
	defer res.Body.Close()
	var out struct {
		PaymentReference string `json:"paymentReference"`
		CheckoutURL      string `json:"checkoutUrl"`
	}
	if res.StatusCode != 201 || json.NewDecoder(res.Body).Decode(&out) != nil || out.PaymentReference == "" {
		httpx.Error(w, 502, "payment provider unavailable, please try again later")
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	if _, err := tx.Exec(r.Context(), `INSERT INTO payments(id,bill_id,payment_reference,amount) VALUES($1,$2,$3,$4)`, uuid.NewString(), b.ID, out.PaymentReference, b.Total); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE utility_bills SET status='PENDING' WHERE id=$1 AND status='UNPAID'`, b.ID); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	httpx.JSON(w, 201, map[string]string{"paymentReference": out.PaymentReference, "checkoutUrl": out.CheckoutURL})
}

// callback applies a verified payment result from the payment adapter.
// It is idempotent: a payment already in a final state is left unchanged.
func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PaymentReference string    `json:"paymentReference"`
		Status           string    `json:"status"`
		PaidAt           time.Time `json:"paidAt"`
	}
	if err := httpx.Decode(r, &in); err != nil || in.PaymentReference == "" || (in.Status != "SUCCEEDED" && in.Status != "FAILED") {
		httpx.Error(w, 400, "invalid callback")
		return
	}
	paid, bill, err := s.applyPayment(r.Context(), in.PaymentReference, in.Status, in.PaidAt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, 404, "unknown payment reference")
		return
	}
	if err != nil {
		slog.Error("payment callback", "err", err)
		httpx.Error(w, 500, "internal error")
		return
	}
	if paid {
		s.pub.PublishLogged(r.Context(), events.PaymentCompleted, events.BillPayload{BillID: bill.ID, RoomID: bill.RoomID, Cycle: bill.Cycle, Total: bill.Total})
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}

// applyPayment returns paid=true only when this call transitioned the bill to PAID.
func (s *Service) applyPayment(ctx context.Context, ref, st string, paidAt time.Time) (paid bool, bill Bill, err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, bill, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var billID, cur string
	if err = tx.QueryRow(ctx, `SELECT bill_id::text, payment_status FROM payments WHERE payment_reference=$1 FOR UPDATE`, ref).Scan(&billID, &cur); err != nil {
		return false, bill, err
	}
	if cur != "PENDING" {
		return false, bill, nil // duplicate webhook
	}
	if paidAt.IsZero() {
		paidAt = time.Now()
	}
	if _, err = tx.Exec(ctx, `UPDATE payments SET payment_status=$2, paid_at=CASE WHEN $2='SUCCEEDED' THEN $3::timestamptz END WHERE payment_reference=$1`, ref, st, paidAt); err != nil {
		return false, bill, err
	}
	if st == "SUCCEEDED" {
		tag, e := tx.Exec(ctx, `UPDATE utility_bills SET status='PAID' WHERE id=$1 AND status<>'PAID'`, billID)
		if e != nil {
			return false, bill, e
		}
		paid = tag.RowsAffected() > 0
	} else {
		_, err = tx.Exec(ctx, `UPDATE utility_bills SET status='UNPAID'
WHERE id=$1 AND status='PENDING' AND NOT EXISTS (SELECT 1 FROM payments WHERE bill_id=$1 AND payment_status='PENDING')`, billID)
		if err != nil {
			return false, bill, err
		}
	}
	if err = scanBill(tx.QueryRow(ctx, `SELECT `+billCols+` FROM utility_bills WHERE id=$1`, billID), &bill); err != nil {
		return false, bill, err
	}
	return paid, bill, tx.Commit(ctx)
}

type Receipt struct {
	ReceiptNo        string    `json:"receiptNo"`
	PaymentReference string    `json:"paymentReference"`
	PaidAt           time.Time `json:"paidAt"`
	Amount           float64   `json:"amount"`
	Bill             Bill      `json:"bill"`
}

func (s *Service) receipt(w http.ResponseWriter, r *http.Request) {
	b, code, msg := s.loadBill(r.Context(), httpx.IdentityFrom(r), chi.URLParam(r, "billId"))
	if b == nil {
		httpx.Error(w, code, msg)
		return
	}
	rc := Receipt{Bill: *b}
	var pid string
	err := s.db.QueryRow(r.Context(), `SELECT id::text, payment_reference, paid_at, amount::float8 FROM payments
WHERE bill_id=$1 AND payment_status='SUCCEEDED' ORDER BY paid_at LIMIT 1`, b.ID).Scan(&pid, &rc.PaymentReference, &rc.PaidAt, &rc.Amount)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, 404, "no completed payment for this bill")
		return
	}
	if err != nil {
		httpx.Error(w, 500, "internal error")
		return
	}
	rc.ReceiptNo = "RC-" + strings.ToUpper(pid[:8])
	httpx.JSON(w, 200, rc)
}

// returnURLFor appends the bill id so the resident app can show the right bill after checkout.
func returnURLFor(base, billID string) string {
	u, err := url.Parse(base)
	if err != nil || base == "" {
		return base
	}
	q := u.Query()
	q.Set("billId", billID)
	u.RawQuery = q.Encode()
	return u.String()
}
