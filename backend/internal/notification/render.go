package notification

import (
	"encoding/json"
	"fmt"
	"net/url"

	"condo/pkg/events"
)

// Message is the user-facing content derived from a domain event.
type Message struct {
	Type, Title, Body, RefType, RefID string
	// Recipients is the identity-service query selecting who receives it.
	Recipients url.Values
}

// Render maps a domain event to a notification message. ok=false means the
// event type is not one the adapter notifies on.
func Render(env events.Envelope) (m Message, ok bool, err error) {
	switch env.Type {
	case events.ParcelReceived:
		var p events.ParcelPayload
		if err = json.Unmarshal(env.Payload, &p); err != nil {
			return
		}
		body := "A new parcel is waiting for you at the juristic office."
		if p.Carrier != "" {
			body = fmt.Sprintf("A new parcel from %s is waiting for you at the juristic office.", p.Carrier)
		}
		return Message{Type: env.Type, Title: "New parcel arrived", Body: body, RefType: "parcel", RefID: p.ParcelID,
			Recipients: url.Values{"roomId": {p.RoomID}}}, true, nil
	case events.ParcelAccepted:
		var p events.ParcelPayload
		if err = json.Unmarshal(env.Payload, &p); err != nil {
			return
		}
		return Message{Type: env.Type, Title: "Parcel picked up", Body: "A resident confirmed pickup of a parcel.", RefType: "parcel", RefID: p.ParcelID,
			Recipients: url.Values{"role": {"staff"}}}, true, nil
	case events.AnnouncementPublished:
		var a events.AnnouncementPayload
		if err = json.Unmarshal(env.Payload, &a); err != nil {
			return
		}
		q := url.Values{}
		switch a.TargetType {
		case "BUILDING", "FLOOR":
			q.Set("buildingId", a.BuildingID)
			if a.TargetType == "FLOOR" {
				q.Set("floor", fmt.Sprint(a.Floor))
			}
		default:
			q.Set("all", "true")
		}
		return Message{Type: env.Type, Title: "New announcement", Body: a.Title, RefType: "announcement", RefID: a.AnnouncementID, Recipients: q}, true, nil
	case events.UtilityBillIssued:
		var b events.BillPayload
		if err = json.Unmarshal(env.Payload, &b); err != nil {
			return
		}
		return Message{Type: env.Type, Title: "New utility bill", Body: fmt.Sprintf("Your bill for %s is %.2f THB, due %s.", b.Cycle, b.Total, b.DueDate),
			RefType: "bill", RefID: b.BillID, Recipients: url.Values{"roomId": {b.RoomID}}}, true, nil
	case events.PaymentCompleted:
		var b events.BillPayload
		if err = json.Unmarshal(env.Payload, &b); err != nil {
			return
		}
		return Message{Type: env.Type, Title: "Payment received", Body: fmt.Sprintf("We received your payment of %.2f THB. Thank you!", b.Total),
			RefType: "bill", RefID: b.BillID, Recipients: url.Values{"roomId": {b.RoomID}}}, true, nil
	}
	return Message{}, false, nil
}
