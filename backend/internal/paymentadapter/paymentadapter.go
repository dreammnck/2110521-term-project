// Package paymentadapter is the Payment Adapter: it talks to the external
// Payment Gateway and verifies its webhooks before notifying Billing (ADR-05).
package paymentadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"condo/pkg/httpx"
	"condo/pkg/webhook"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Config struct {
	GatewayURL    string // internal URL of the gateway API
	WebhookSecret string
	WebhookURL    string // URL the gateway calls back (this adapter)
	BillingURL    string
}

type Service struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Service {
	return &Service{cfg: cfg, http: &http.Client{Timeout: 8 * time.Second}}
}

func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Post("/payments", s.createPayment)
	r.Get("/payments/{reference}/status", s.getStatus)
	r.Post("/webhooks/gateway", s.webhook)
	return r
}

// createPayment opens a checkout session at the gateway.
func (s *Service) createPayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BillID      string  `json:"billId"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		ReturnURL   string  `json:"returnUrl"`
	}
	if err := httpx.Decode(r, &in); err != nil || in.BillID == "" || in.Amount <= 0 {
		httpx.Error(w, 400, "billId and amount are required")
		return
	}
	ref := "pay_" + uuid.NewString()
	body, _ := json.Marshal(map[string]any{"reference": ref, "amount": in.Amount, "description": in.Description, "returnUrl": in.ReturnURL, "webhookUrl": s.cfg.WebhookURL})
	res, err := s.http.Post(s.cfg.GatewayURL+"/sessions", "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("gateway unreachable", "err", err)
		httpx.Error(w, 502, "payment gateway unavailable")
		return
	}
	defer res.Body.Close()
	var out struct {
		CheckoutURL string `json:"checkoutUrl"`
	}
	if res.StatusCode != 201 || json.NewDecoder(res.Body).Decode(&out) != nil {
		httpx.Error(w, 502, "payment gateway rejected the request")
		return
	}
	httpx.JSON(w, 201, map[string]string{"paymentReference": ref, "checkoutUrl": out.CheckoutURL})
}

func (s *Service) getStatus(w http.ResponseWriter, r *http.Request) {
	res, err := s.http.Get(s.cfg.GatewayURL + "/sessions/" + chi.URLParam(r, "reference"))
	if err != nil {
		httpx.Error(w, 502, "payment gateway unavailable")
		return
	}
	defer res.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}

// webhook authenticates the gateway callback, then forwards the result to Billing.
// A non-2xx response makes the gateway retry, so Billing failures are surfaced.
func (s *Service) webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, 400, "unreadable body")
		return
	}
	if err := webhook.Verify(s.cfg.WebhookSecret, r.Header.Get("X-Timestamp"), r.Header.Get("X-Signature"), body, time.Now()); err != nil {
		slog.Warn("rejected webhook", "err", err)
		httpx.Error(w, 401, "invalid webhook")
		return
	}
	var in struct {
		Reference string `json:"reference"`
		Status    string `json:"status"`
		PaidAt    string `json:"paidAt"`
	}
	if err := json.Unmarshal(body, &in); err != nil || in.Reference == "" {
		httpx.Error(w, 400, "invalid payload")
		return
	}
	if in.Status != "SUCCEEDED" && in.Status != "FAILED" {
		httpx.JSON(w, 200, map[string]string{"ignored": in.Status})
		return
	}
	fwd, _ := json.Marshal(map[string]string{"paymentReference": in.Reference, "status": in.Status, "paidAt": in.PaidAt})
	res, err := s.http.Post(s.cfg.BillingURL+"/internal/payments/callback", "application/json", bytes.NewReader(fwd))
	if err != nil {
		httpx.Error(w, 502, "billing unavailable")
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		httpx.Error(w, 502, fmt.Sprintf("billing rejected callback (%d)", res.StatusCode))
		return
	}
	httpx.JSON(w, 200, map[string]bool{"ok": true})
}
