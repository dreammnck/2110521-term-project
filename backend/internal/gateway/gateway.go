// Package gateway is the API Gateway: it authenticates requests, routes REST
// calls to backend services and translates REST<->gRPC for announcements (ADR-02).
package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"condo/pkg/auth"
	"condo/pkg/httpx"

	pb "condo/gen/announcement/v1"

	"github.com/go-chi/chi/v5"
)

type Config struct {
	JWTSecret       string
	IdentityURL     string
	ParcelURL       string
	BillingURL      string
	NotificationURL string
	AllowedOrigins  []string
}

type Gateway struct {
	cfg   Config
	ann   pb.AnnouncementServiceClient
	proxy map[string]*httputil.ReverseProxy
}

func New(cfg Config, ann pb.AnnouncementServiceClient) (*Gateway, error) {
	g := &Gateway{cfg: cfg, ann: ann, proxy: map[string]*httputil.ReverseProxy{}}
	for name, raw := range map[string]string{"identity": cfg.IdentityURL, "parcel": cfg.ParcelURL, "billing": cfg.BillingURL, "notification": cfg.NotificationURL} {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		g.proxy[name] = newProxy(u)
	}
	return g, nil
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path = strings.TrimPrefix(pr.In.URL.Path, "/api/v1")
			pr.Out.URL.RawPath = ""
			pr.Out.Host = target.Host
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			httpx.Error(w, http.StatusBadGateway, "upstream service unavailable")
		},
	}
}

func (g *Gateway) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(g.cors)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })

	r.Route("/api/v1", func(api chi.Router) {
		// Public.
		api.Method(http.MethodPost, "/auth/login", g.proxy["identity"])

		// Everything below requires a valid JWT. Only whitelisted prefixes are routed,
		// so /internal/* endpoints of backend services are never reachable.
		api.Group(func(p chi.Router) {
			p.Use(g.authenticate)
			p.Handle("/users/*", g.proxy["identity"])
			p.Handle("/rooms", g.proxy["identity"])
			p.Handle("/parcels", g.proxy["parcel"])
			p.Handle("/parcels/*", g.proxy["parcel"])
			p.Handle("/utilities/*", g.proxy["billing"])
			p.Handle("/payments", g.proxy["billing"])
			p.Handle("/payments/*", g.proxy["billing"])
			p.Handle("/notifications", g.proxy["notification"])
			p.Handle("/notifications/*", g.proxy["notification"])

			p.Method(http.MethodPost, "/announcements", staffOnly(g.publishAnnouncement))
			p.Get("/announcements", g.listAnnouncements)
			p.Get("/announcements/{id}", g.getAnnouncement)
			p.Method(http.MethodPatch, "/announcements/{id}", staffOnly(g.updateAnnouncement))
			p.Method(http.MethodPost, "/announcements/{id}/archive", staffOnly(g.archiveAnnouncement))
			p.Post("/announcements/{id}/read", g.markAnnouncementRead)
		})
	})
	return r
}

func staffOnly(h http.HandlerFunc) http.Handler { return httpx.RequireRoleFunc(auth.RoleStaff, h) }

// authenticate verifies the bearer token and injects trusted identity headers,
// first removing any identity headers supplied by the client.
func (g *Gateway) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, h := range []string{httpx.HeaderUserID, httpx.HeaderRole, httpx.HeaderRoomID, httpx.HeaderBuildingID, httpx.HeaderFloor} {
			r.Header.Del(h)
		}
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" || tok == r.Header.Get("Authorization") {
			httpx.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		c, err := auth.Parse(g.cfg.JWTSecret, tok)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		r.Header.Set(httpx.HeaderUserID, c.UserID)
		r.Header.Set(httpx.HeaderRole, c.Role)
		r.Header.Set(httpx.HeaderRoomID, c.RoomID)
		r.Header.Set(httpx.HeaderBuildingID, c.BuildingID)
		r.Header.Set(httpx.HeaderFloor, strconv.Itoa(c.Floor))
		next.ServeHTTP(w, r)
	})
}

func (g *Gateway) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range g.cfg.AllowedOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && allowed[o] {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
