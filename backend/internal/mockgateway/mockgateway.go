// Package mockgateway simulates the external Payment Gateway: it hosts a
// checkout page and sends HMAC-signed webhooks back to the payment adapter.
package mockgateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"condo/pkg/httpx"
	"condo/pkg/webhook"

	"github.com/go-chi/chi/v5"
)

type session struct {
	Reference   string  `json:"reference"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	ReturnURL   string  `json:"returnUrl"`
	WebhookURL  string  `json:"webhookUrl"`
	Status      string  `json:"status"` // PENDING | SUCCEEDED | FAILED
}

type Server struct {
	mu        sync.Mutex
	sessions  map[string]*session
	secret    string
	publicURL string
	http      *http.Client
}

func New(secret, publicURL string) *Server {
	return &Server{sessions: map[string]*session{}, secret: secret, publicURL: publicURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Post("/sessions", s.create)
	r.Get("/sessions/{ref}", s.status)
	r.Get("/checkout/{ref}", s.checkout)
	r.Post("/checkout/{ref}/complete", s.complete)
	return r
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in session
	if err := httpx.Decode(r, &in); err != nil || in.Reference == "" || in.Amount <= 0 || in.WebhookURL == "" {
		httpx.Error(w, 400, "reference, amount and webhookUrl are required")
		return
	}
	in.Status = "PENDING"
	s.mu.Lock()
	s.sessions[in.Reference] = &in
	s.mu.Unlock()
	httpx.JSON(w, 201, map[string]string{"reference": in.Reference, "checkoutUrl": s.publicURL + "/checkout/" + url.PathEscape(in.Reference)})
}

func (s *Server) get(ref string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[ref]
	if !ok {
		return session{}, false
	}
	return *ss, true
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	ss, ok := s.get(chi.URLParam(r, "ref"))
	if !ok {
		httpx.Error(w, 404, "unknown session")
		return
	}
	httpx.JSON(w, 200, map[string]any{"reference": ss.Reference, "status": ss.Status, "amount": ss.Amount})
}

func (s *Server) checkout(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")
	ss, ok := s.get(ref)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ss.Status != "PENDING" {
		fmt.Fprintf(w, page(`<h1>Payment %s</h1><p>This session is already completed.</p>`), html.EscapeString(ss.Status))
		return
	}
	fmt.Fprintf(w, page(`<h1>Mock Payment Gateway</h1>
<p>%s</p><p class="amt">%.2f THB</p>
<form method="post" action="/checkout/%s/complete">
<button name="result" value="success" class="ok">Pay now</button>
<button name="result" value="fail" class="no">Fail payment</button>
</form><p class="hint">Test environment. No real money is moved.</p>`),
		html.EscapeString(ss.Description), ss.Amount, url.PathEscape(ref))
}

func page(body string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Mock Gateway</title>
<style>body{font-family:system-ui,sans-serif;max-width:380px;margin:12vh auto;padding:0 16px;text-align:center}
.amt{font-size:2rem;font-weight:700}button{font-size:1rem;padding:.8rem 1.2rem;margin:.3rem;border:0;border-radius:8px;color:#fff;cursor:pointer}
.ok{background:#16a34a}.no{background:#dc2626}.hint{color:#666;font-size:.85rem}</style></head><body>` + body + `</body></html>`
}

func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")
	s.mu.Lock()
	ss, ok := s.sessions[ref]
	if !ok {
		s.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	if ss.Status == "PENDING" {
		if r.FormValue("result") == "success" {
			ss.Status = "SUCCEEDED"
		} else {
			ss.Status = "FAILED"
		}
	}
	snap := *ss
	s.mu.Unlock()

	s.sendWebhook(snap)

	back := snap.ReturnURL
	if back == "" {
		http.Redirect(w, r, "/checkout/"+url.PathEscape(ref), http.StatusSeeOther)
		return
	}
	u, err := url.Parse(back)
	if err != nil {
		http.Redirect(w, r, "/checkout/"+url.PathEscape(ref), http.StatusSeeOther)
		return
	}
	q := u.Query()
	q.Set("reference", ref)
	q.Set("status", snap.Status)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// sendWebhook retries a few times, as a real gateway would.
func (s *Server) sendWebhook(ss session) {
	body, _ := json.Marshal(map[string]any{"reference": ss.Reference, "status": ss.Status, "amount": ss.Amount, "paidAt": time.Now().UTC().Format(time.RFC3339)})
	for attempt := 1; attempt <= 3; attempt++ {
		ts := time.Now().Unix()
		req, _ := http.NewRequest(http.MethodPost, ss.WebhookURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Timestamp", fmt.Sprint(ts))
		req.Header.Set("X-Signature", webhook.Sign(s.secret, ts, body))
		res, err := s.http.Do(req)
		if err == nil {
			res.Body.Close()
			if res.StatusCode < 300 {
				return
			}
		}
		slog.Warn("webhook delivery failed", "attempt", attempt, "err", err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
}
