package billing

import (
	"testing"
	"time"
)

func TestCharges(t *testing.T) {
	w, e, total := Charges(10.5, 100, 18, 7)
	if w != 189 || e != 700 || total != 889 {
		t.Fatalf("got %v %v %v", w, e, total)
	}
}

func TestEffectiveStatus(t *testing.T) {
	today := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	past, future := today.AddDate(0, 0, -1), today.AddDate(0, 0, 5)
	cases := []struct {
		st   string
		due  time.Time
		want string
	}{
		{"UNPAID", past, "OVERDUE"},
		{"UNPAID", future, "UNPAID"},
		{"UNPAID", today, "UNPAID"},
		{"PAID", past, "PAID"},
		{"PENDING", past, "PENDING"},
	}
	for _, c := range cases {
		if got := EffectiveStatus(c.st, c.due, today); got != c.want {
			t.Errorf("%s due %v: got %s want %s", c.st, c.due, got, c.want)
		}
	}
}

func TestBillInputValidate(t *testing.T) {
	ok := BillInput{RoomID: "r", Cycle: "2026-10", WaterUnits: 1, ElectricUnits: 2, DueDate: "2026-11-05"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []BillInput{
		{Cycle: "2026-10", DueDate: "2026-11-05"},
		{RoomID: "r", Cycle: "2026-13", DueDate: "2026-11-05"},
		{RoomID: "r", Cycle: "2026-10", WaterUnits: -1, DueDate: "2026-11-05"},
		{RoomID: "r", Cycle: "2026-10", DueDate: "05/11/2026"},
	}
	for i, b := range bad {
		if b.Validate() == nil {
			t.Errorf("case %d should be invalid", i)
		}
	}
}

func TestReturnURLFor(t *testing.T) {
	got := returnURLFor("http://localhost:3001/bills/return", "abc")
	if got != "http://localhost:3001/bills/return?billId=abc" {
		t.Fatal(got)
	}
	if got := returnURLFor("", "abc"); got != "" {
		t.Fatalf("empty base should stay empty, got %q", got)
	}
}
