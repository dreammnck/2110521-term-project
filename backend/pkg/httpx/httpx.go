package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Identity headers are set by the API Gateway after it verifies the JWT.
const (
	HeaderUserID     = "X-User-Id"
	HeaderRole       = "X-User-Role"
	HeaderRoomID     = "X-Room-Id"
	HeaderBuildingID = "X-Building-Id"
	HeaderFloor      = "X-Floor"
)

type Identity struct {
	UserID     string
	Role       string
	RoomID     string
	BuildingID string
	Floor      int
}

func IdentityFrom(r *http.Request) Identity {
	floor, _ := strconv.Atoi(r.Header.Get(HeaderFloor))
	return Identity{
		UserID:     r.Header.Get(HeaderUserID),
		Role:       r.Header.Get(HeaderRole),
		RoomID:     r.Header.Get(HeaderRoomID),
		BuildingID: r.Header.Get(HeaderBuildingID),
		Floor:      floor,
	}
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

func Decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	return json.NewDecoder(r.Body).Decode(v)
}

// RequireRole rejects requests whose gateway-supplied role does not match.
func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IdentityFrom(r).Role != role {
			Error(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRoleFunc(role string, next http.HandlerFunc) http.Handler {
	return RequireRole(role, next)
}
