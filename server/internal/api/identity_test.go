package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"yeonly-server/internal/model"
	"yeonly-server/internal/room"
	"yeonly-server/internal/ws"
)

func TestParticipantTokenContract(t *testing.T) {
	mgr := room.NewManager()
	hub := ws.NewHub(mgr)
	go hub.Run()

	server := NewServer(mgr, hub)
	ts := httptest.NewServer(server.SetupRoutes(""))
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/rooms",
		"application/json",
		strings.NewReader(`{"name":"Identity Contract","isPublic":true}`),
	)
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	defer resp.Body.Close()

	var created struct {
		RoomID  string `json:"roomId"`
		HostKey string `json:"hostKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode room: %v", err)
	}
	if created.RoomID == "" || created.HostKey == "" {
		t.Fatalf("invalid room creation response: %+v", created)
	}

	baseWS := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + created.RoomID

	// A first student connection without a token must receive a server identity.
	first, _, err := websocket.DefaultDialer.Dial(baseWS, nil)
	if err != nil {
		t.Fatalf("failed to dial first student: %v", err)
	}
	state := readRoomState(t, first)
	first.Close()

	if state.ParticipantID == "" || state.ParticipantToken == "" {
		t.Fatalf("server did not issue identity: %+v", state)
	}
	if state.ParticipantID == state.ParticipantToken {
		t.Fatal("public participant ID must not be the bearer token")
	}

	// Reconnecting with the bearer token must retain the same identity.
	reconnectURL := baseWS + "&session=" + url.QueryEscape(state.ParticipantToken)
	second, _, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
	if err != nil {
		t.Fatalf("failed to reconnect with token: %v", err)
	}
	reconnected := readRoomState(t, second)
	second.Close()

	if reconnected.ParticipantID != state.ParticipantID {
		t.Fatalf("token reconnect changed identity: %q != %q", reconnected.ParticipantID, state.ParticipantID)
	}

	// The public participant ID is not a credential and must be rejected.
	badURL := baseWS + "&session=" + url.QueryEscape(state.ParticipantID)
	bad, _, err := websocket.DefaultDialer.Dial(badURL, nil)
	if err != nil {
		t.Fatalf("failed to dial invalid identity: %v", err)
	}
	defer bad.Close()
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	var rejected model.WSEvent
	if err := bad.ReadJSON(&rejected); err != nil {
		t.Fatalf("failed to read rejection: %v", err)
	}
	if rejected.Type != "IDENTITY_EXPIRED" {
		t.Fatalf("expected IDENTITY_EXPIRED, got %s", rejected.Type)
	}

	// The host authenticates with the host key, not a participant token.
	hostURL := baseWS + "&hostKey=" + url.QueryEscape(created.HostKey)
	host, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatalf("failed to dial host: %v", err)
	}
	hostState := readRoomState(t, host)
	host.Close()

	if !hostState.IsHost {
		t.Fatal("host key did not produce a host identity")
	}
	if hostState.ParticipantID == state.ParticipantID {
		t.Fatal("host and student received the same identity")
	}
}
