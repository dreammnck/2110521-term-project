package notification

import (
	"encoding/json"
	"testing"

	"condo/pkg/events"
)

func env(t string, p any) events.Envelope {
	b, _ := json.Marshal(p)
	return events.Envelope{ID: "e1", Type: t, Payload: b}
}

func TestRender(t *testing.T) {
	m, ok, err := Render(env(events.ParcelReceived, events.ParcelPayload{ParcelID: "p", RoomID: "r1", Carrier: "Kerry"}))
	if err != nil || !ok || m.Recipients.Get("roomId") != "r1" || m.RefID != "p" {
		t.Fatalf("parcel received: %+v ok=%v err=%v", m, ok, err)
	}
	m, _, _ = Render(env(events.ParcelAccepted, events.ParcelPayload{ParcelID: "p", RoomID: "r1"}))
	if m.Recipients.Get("role") != "staff" {
		t.Errorf("parcel accepted should notify staff: %v", m.Recipients)
	}
	m, _, _ = Render(env(events.AnnouncementPublished, events.AnnouncementPayload{AnnouncementID: "a", TargetType: "FLOOR", BuildingID: "b", Floor: 2}))
	if m.Recipients.Get("buildingId") != "b" || m.Recipients.Get("floor") != "2" {
		t.Errorf("floor target: %v", m.Recipients)
	}
	m, _, _ = Render(env(events.AnnouncementPublished, events.AnnouncementPayload{AnnouncementID: "a", TargetType: "ALL"}))
	if m.Recipients.Get("all") != "true" {
		t.Errorf("all target: %v", m.Recipients)
	}
	m, _, _ = Render(env(events.UtilityBillIssued, events.BillPayload{BillID: "b", RoomID: "r2", Total: 10}))
	if m.Recipients.Get("roomId") != "r2" {
		t.Errorf("bill issued: %v", m.Recipients)
	}
	if _, ok, _ := Render(events.Envelope{Type: "unknown.event", Payload: []byte(`{}`)}); ok {
		t.Error("unknown events must be ignored")
	}
	if _, _, err := Render(events.Envelope{Type: events.ParcelReceived, Payload: []byte(`not json`)}); err == nil {
		t.Error("malformed payload should error")
	}
}
