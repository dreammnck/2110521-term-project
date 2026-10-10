package announcement

import "testing"

func TestVisible(t *testing.T) {
	cases := []struct {
		name     string
		tt, tb   string
		tf       int32
		building string
		floor    int32
		want     bool
	}{
		{"all", "ALL", "", 0, "A", 1, true},
		{"building match", "BUILDING", "A", 0, "A", 3, true},
		{"building mismatch", "BUILDING", "A", 0, "B", 3, false},
		{"floor match", "FLOOR", "A", 2, "A", 2, true},
		{"floor wrong floor", "FLOOR", "A", 2, "A", 1, false},
		{"floor wrong building", "FLOOR", "A", 2, "B", 2, false},
		{"no building", "BUILDING", "A", 0, "", 0, false},
		{"unknown type", "X", "", 0, "A", 1, false},
	}
	for _, c := range cases {
		if got := Visible(c.tt, c.tb, c.tf, c.building, c.floor); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestValidateTarget(t *testing.T) {
	if validateTarget("ALL", "", 0) != nil || validateTarget("BUILDING", "b", 0) != nil || validateTarget("FLOOR", "b", 2) != nil {
		t.Error("valid targets rejected")
	}
	if validateTarget("BUILDING", "", 0) == nil || validateTarget("FLOOR", "b", 0) == nil || validateTarget("ROOM", "b", 1) == nil {
		t.Error("invalid targets accepted")
	}
}
