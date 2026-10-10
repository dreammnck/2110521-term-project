package webhook

import (
	"strconv"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	now := time.Now()
	body := []byte(`{"reference":"pay_1","status":"SUCCEEDED"}`)
	ts := now.Unix()
	sig := Sign("secret", ts, body)
	tsStr := strconv.FormatInt(ts, 10)

	if err := Verify("secret", tsStr, sig, body, now); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := Verify("other", tsStr, sig, body, now); err != ErrBadSignature {
		t.Errorf("wrong secret: %v", err)
	}
	if err := Verify("secret", tsStr, sig, []byte(`{"tampered":true}`), now); err != ErrBadSignature {
		t.Errorf("tampered body: %v", err)
	}
	if err := Verify("secret", tsStr, sig, body, now.Add(10*time.Minute)); err != ErrStale {
		t.Errorf("stale timestamp: %v", err)
	}
	if err := Verify("secret", "abc", sig, body, now); err != ErrBadSignature {
		t.Errorf("bad timestamp: %v", err)
	}
}
