package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"yeonly-server/internal/model"
	"yeonly-server/internal/room"
)

func testClient(h *Hub, roomID, alias string) *Client {
	c := &Client{RoomID: roomID, SessionID: "private-session-" + alias, UserAlias: alias, Send: make(chan []byte, 256)}
	h.registerClient(c)
	return c
}

func drain(c *Client) []model.WSEvent {
	var events []model.WSEvent
	for len(c.Send) > 0 {
		var event model.WSEvent
		_ = json.Unmarshal(<-c.Send, &event)
		events = append(events, event)
	}
	return events
}

func assertTyping(t *testing.T, c *Client, alias string, active bool) {
	t.Helper()
	events := drain(c)
	if len(events) != 1 || events[0].Type != model.EventTypingUpdated {
		t.Fatalf("expected one typing event, got %+v", events)
	}
	payload := events[0].Payload.(map[string]interface{})
	if payload["alias"] != alias || payload["isTyping"] != active || payload["id"] == "" || len(payload) != 3 {
		t.Fatalf("unexpected typing payload: %+v", payload)
	}
	if payload["id"] == "private-session-"+alias {
		t.Fatal("typing must not expose session IDs")
	}
}

func TestTypingPresenceLifecycle(t *testing.T) {
	mgr := room.NewManager()
	r, key, _ := mgr.CreateRoom("Typing", false)
	other, _, _ := mgr.CreateRoom("Other room", false)
	h := NewHub(mgr)
	a := testClient(h, r.ID, "Panda #01")
	b := testClient(h, r.ID, "Koala #02")
	c := testClient(h, other.ID, "Kucing #03")
	drain(a)
	drain(b)
	drain(c)

	// The server derives identity and room from the connection, not the payload.
	h.HandleAction(a, []byte(`{"action":"TYPING","isTyping":true,"alias":"spoofed","text":"private draft"}`))
	assertTyping(t, b, a.UserAlias, true)
	if len(drain(a)) != 0 || len(drain(c)) != 0 {
		t.Fatal("typing was echoed to its author or leaked into another room")
	}
	h.setTyping(a, true)
	if len(drain(b)) != 0 {
		t.Fatal("typing heartbeat was not throttled")
	}

	newcomer := testClient(h, r.ID, "Elang #04")
	events := drain(newcomer)
	state := events[0].Payload.(map[string]interface{})
	if len(state["typingUsers"].([]interface{})) != 1 {
		t.Fatal("joining participant missed active typing")
	}
	drain(a)
	drain(b)

	h.expireTyping(time.Now().Add(typingTTL + time.Second))
	assertTyping(t, b, a.UserAlias, false)
	drain(newcomer)
	if len(h.typing) != 0 {
		t.Fatal("stale typing was retained")
	}

	h.setTyping(a, true)
	drain(b)
	drain(newcomer)
	h.HandleAction(a, []byte(`{"action":"SUBMIT_QUESTION","text":"Bagaimana cara kerja goroutine?\nBerikan contoh.","requestId":"q-1"}`))
	events = drain(b)
	if len(events) != 2 || events[0].Type != model.EventTypingUpdated || events[1].Type != model.EventQuestionAdded {
		t.Fatalf("send must replace typing with the question: %+v", events)
	}
	if events[1].Payload.(map[string]interface{})["text"] != "Bagaimana cara kerja goroutine?\nBerikan contoh." {
		t.Fatal("delivered message lost the composer's line breaks")
	}
	self := drain(a)
	if len(self) != 2 || self[1].Type != model.EventQuestionAccepted || self[1].RequestID != "q-1" {
		t.Fatalf("missing sender acknowledgement: %+v", self)
	}
	drain(newcomer)

	h.setTyping(a, true)
	h.setTyping(b, true)
	drain(a)
	drain(b)
	drain(newcomer)
	freeze, _ := json.Marshal(map[string]string{"action": model.ActionHostFreeze, "hostKey": key})
	h.HandleAction(b, freeze)
	if len(h.typing) != 0 {
		t.Fatal("freezing room did not clear all typing")
	}
	drain(a)
	drain(b)
	drain(newcomer)
	h.setTyping(a, true)
	if len(drain(b)) != 0 {
		t.Fatal("frozen room allowed typing")
	}
	h.HandleAction(a, []byte(`{"action":"SUBMIT_QUESTION","text":"This must remain a draft","requestId":"q-2"}`))
	rejected := drain(a)
	if len(rejected) != 1 || rejected[0].Type != model.EventError || rejected[0].RequestID != "q-2" {
		t.Fatalf("rejection must identify its submission: %+v", rejected)
	}

	h.HandleAction(b, freeze)
	drain(a)
	drain(b)
	drain(newcomer)
	h.setTyping(a, true)
	drain(b)
	drain(newcomer)
	h.unregisterClient(a)
	events = drain(b)
	if len(events) != 2 || events[0].Type != model.EventTypingUpdated || events[1].Type != model.EventUserCount {
		t.Fatalf("disconnect left stale typing: %+v", events)
	}
	h.setTyping(a, true)
	if len(h.typing) != 0 {
		t.Fatal("disconnected client restored typing")
	}
}

func TestTypingStopIsIdempotentAndBroadcastSafeDuringDisconnect(t *testing.T) {
	mgr := room.NewManager()
	r, _, _ := mgr.CreateRoom("Concurrency", false)
	h := NewHub(mgr)
	a := testClient(h, r.ID, "Panda")
	b := testClient(h, r.ID, "Koala")
	drain(a)
	drain(b)
	h.setTyping(a, true)
	drain(b)
	h.setTyping(a, false)
	assertTyping(t, b, a.UserAlias, false)
	h.setTyping(a, false)
	if len(drain(b)) != 0 {
		t.Fatal("duplicate stop event")
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			h.BroadcastToRoom(r.ID, model.WSEvent{Type: model.EventToast, Payload: "test"})
		}
	}()
	go func() { defer wg.Done(); h.unregisterClient(b) }()
	wg.Wait()
}
