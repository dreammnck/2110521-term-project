package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"condo/pkg/auth"
	"condo/pkg/httpx"
)

func TestAuthenticateStripsSpoofedHeadersAndSetsIdentity(t *testing.T) {
	g := &Gateway{cfg: Config{JWTSecret: "s"}}
	var got httpx.Identity
	h := g.authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = httpx.IdentityFrom(r) }))

	tok, _ := auth.Sign("s", auth.Claims{UserID: "u1", Role: auth.RoleResident, RoomID: "room1", BuildingID: "b1", Floor: 2}, time.Hour)
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set(httpx.HeaderRole, "staff") // spoof attempt
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || got.Role != auth.RoleResident || got.UserID != "u1" || got.RoomID != "room1" || got.BuildingID != "b1" || got.Floor != 2 {
		t.Fatalf("code=%d identity=%+v", rec.Code, got)
	}

	for name, header := range map[string]string{"missing": "", "garbage": "Bearer nope", "no-scheme": tok} {
		req := httptest.NewRequest("GET", "/x", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Errorf("%s: expected 401, got %d", name, rec.Code)
		}
	}
}

func TestInternalRoutesNotExposed(t *testing.T) {
	g, err := New(Config{JWTSecret: "s", IdentityURL: "http://x", ParcelURL: "http://x", BillingURL: "http://x", NotificationURL: "http://x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := auth.Sign("s", auth.Claims{UserID: "u", Role: auth.RoleStaff}, time.Hour)
	for _, p := range []string{"/api/v1/internal/payments/callback", "/api/v1/internal/recipients"} {
		req := httptest.NewRequest("POST", p, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		g.Routes().ServeHTTP(rec, req)
		if rec.Code != 404 && rec.Code != 405 {
			t.Errorf("%s reachable via gateway: %d", p, rec.Code)
		}
	}
}
