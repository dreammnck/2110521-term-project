package gateway

import (
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestHTTPStatus(t *testing.T) {
	cases := map[codes.Code]int{
		codes.OK:               http.StatusOK,
		codes.InvalidArgument:  http.StatusBadRequest,
		codes.NotFound:         http.StatusNotFound,
		codes.AlreadyExists:    http.StatusConflict,
		codes.PermissionDenied: http.StatusForbidden,
		codes.Unauthenticated:  http.StatusUnauthorized,
		codes.Unavailable:      http.StatusServiceUnavailable,
		codes.DeadlineExceeded: http.StatusGatewayTimeout,
		codes.Internal:         http.StatusInternalServerError,
	}
	for c, want := range cases {
		if got := HTTPStatus(c); got != want {
			t.Errorf("%v: got %d want %d", c, got, want)
		}
	}
}
