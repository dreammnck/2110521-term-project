// Package webhook signs and verifies gateway callbacks (ADR-05: callbacks
// must be authenticated before payment status is updated).
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

const Tolerance = 5 * time.Minute

func Sign(secret string, ts int64, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strconv.FormatInt(ts, 10)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

var (
	ErrBadSignature = errors.New("invalid signature")
	ErrStale        = errors.New("timestamp outside tolerance")
)

// Verify checks the signature and that the timestamp is fresh relative to now.
func Verify(secret, tsHeader, sigHeader string, body []byte, now time.Time) error {
	ts, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return ErrBadSignature
	}
	d := now.Sub(time.Unix(ts, 0))
	if d < 0 {
		d = -d
	}
	if d > Tolerance {
		return ErrStale
	}
	want := Sign(secret, ts, body)
	if !hmac.Equal([]byte(want), []byte(sigHeader)) {
		return ErrBadSignature
	}
	return nil
}
