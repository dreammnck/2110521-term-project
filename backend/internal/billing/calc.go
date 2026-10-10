package billing

import (
	"errors"
	"math"
	"regexp"
	"time"
)

var cycleRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// Charges computes line amounts and total from metered units.
func Charges(waterUnits, electricUnits, waterRate, electricRate float64) (water, electric, total float64) {
	water = round2(waterUnits * waterRate)
	electric = round2(electricUnits * electricRate)
	return water, electric, round2(water + electric)
}

// EffectiveStatus derives OVERDUE from an unpaid bill past its due date.
func EffectiveStatus(status string, due, today time.Time) string {
	if status == "UNPAID" && due.Before(today) {
		return "OVERDUE"
	}
	return status
}

type BillInput struct {
	RoomID        string  `json:"roomId"`
	Cycle         string  `json:"cycle"`
	WaterUnits    float64 `json:"waterUnits"`
	ElectricUnits float64 `json:"electricUnits"`
	DueDate       string  `json:"dueDate"`
}

func (b BillInput) Validate() error {
	switch {
	case b.RoomID == "":
		return errors.New("roomId is required")
	case !cycleRe.MatchString(b.Cycle):
		return errors.New("cycle must be YYYY-MM")
	case b.WaterUnits < 0 || b.ElectricUnits < 0:
		return errors.New("units must not be negative")
	}
	if _, err := time.Parse("2006-01-02", b.DueDate); err != nil {
		return errors.New("dueDate must be YYYY-MM-DD")
	}
	return nil
}
