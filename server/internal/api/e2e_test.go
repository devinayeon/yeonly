package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"yeonly-server/internal/model"
	"yeonly-server/internal/room"
	"yeonly-server/internal/ws"
)

func readRoomState(t *testing.T, conn *websocket.Conn) model.RoomStatePayload {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event model.WSEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("failed to read initial event: %v", err)
	}
	if event.Type != model.EventRoomState {
		t.Fatalf("expected ROOM_STATE, got %s", event.Type)
	}
	raw, err := json.Marshal(event.Payload)
	if err != nil {
		t.Fatalf("failed to encode room state: %v", err)
	}
	var state model.RoomStatePayload
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("failed to decode room state: %v", err)
	}
	return state
}

func TestFullE2EFlow(t *testing.T) {
	mgr := room.NewManager()
	hub := ws.NewHub(mgr)
	go hub.Run()

	server := NewServer(mgr, hub)
	handler := server.SetupRoutes("")

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Create Room via HTTP POST /api/rooms
	createBody := strings.NewReader(`{"name": "E2E Sesi Kelas", "isPublic": true}`)
	resp, err := http.Post(ts.URL+"/api/rooms", "application/json", createBody)
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var createData struct {
		RoomID  string `json:"roomId"`
		HostKey string `json:"hostKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createData); err != nil {
		t.Fatalf("failed to parse room creation response: %v", err)
	}
	resp.Body.Close()

	if createData.RoomID == "" || createData.HostKey == "" {
		t.Fatalf("invalid room or host key: %+v", createData)
	}

	// 2. Connect Student via WebSocket without a client-generated identity.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + createData.RoomID
	studentConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial student ws: %v", err)
	}
	defer studentConn.Close()

	initialState := readRoomState(t, studentConn)
	if initialState.ParticipantID == "" || initialState.ParticipantToken == "" {
		t.Fatalf("server did not issue participant identity: %+v", initialState)
	}

	// 3. Connect Host via WebSocket using only the host key.
	hostWsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + createData.RoomID + "&hostKey=" + createData.HostKey
	hostConn, _, err := websocket.DefaultDialer.Dial(hostWsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial host ws: %v", err)
	}
	defer hostConn.Close()

	// 4. Student submits question
	submitMsg := map[string]string{
		"action": model.ActionSubmitQuestion,
		"text":   "Apakah concurrency di Go memanfaatkan multicore secara otomatis?",
	}
	if err := studentConn.WriteJSON(submitMsg); err != nil {
		t.Fatalf("failed to submit question: %v", err)
	}

	// Host receives QUESTION_ADDED
	var qEvent model.WSEvent
	_ = hostConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		if err := hostConn.ReadJSON(&qEvent); err != nil {
			t.Fatalf("failed to read question added event: %v", err)
		}
		if qEvent.Type == model.EventQuestionAdded {
			break
		}
	}

	qPayloadBytes, _ := json.Marshal(qEvent.Payload)
	var q model.Question
	_ = json.Unmarshal(qPayloadBytes, &q)

	if q.Text != "Apakah concurrency di Go memanfaatkan multicore secara otomatis?" {
		t.Fatalf("unexpected question text: %s", q.Text)
	}

	// 5. Student upvotes the question
	upvoteMsg := map[string]string{
		"action":     model.ActionUpvoteQuestion,
		"questionId": q.ID,
	}
	if err := studentConn.WriteJSON(upvoteMsg); err != nil {
		t.Fatalf("failed to write upvote: %v", err)
	}

	// 6. Student casts pulse vote
	pulseMsg := map[string]string{
		"action": model.ActionUpdatePulse,
		"vote":   string(model.PulseUnderstood),
	}
	if err := studentConn.WriteJSON(pulseMsg); err != nil {
		t.Fatalf("failed to send pulse: %v", err)
	}

	// 7. Test Export Recap endpoint
	recapResp, err := http.Get(ts.URL + "/api/rooms/" + createData.RoomID + "/recap")
	if err != nil {
		t.Fatalf("failed to get recap: %v", err)
	}
	if recapResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for recap, got %d", recapResp.StatusCode)
	}
	recapResp.Body.Close()
}

func TestIndependentUpvotesMultiUser(t *testing.T) {
	mgr := room.NewManager()
	hub := ws.NewHub(mgr)
	go hub.Run()

	server := NewServer(mgr, hub)
	handler := server.SetupRoutes("")

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Create Room
	createBody := strings.NewReader(`{"name": "Independent Votes Test", "isPublic": true}`)
	resp, err := http.Post(ts.URL+"/api/rooms", "application/json", createBody)
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	var createData struct {
		RoomID  string `json:"roomId"`
		HostKey string `json:"hostKey"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createData)
	resp.Body.Close()

	// 2. Connect Host (Yeonly)
	hostWsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + createData.RoomID + "&hostKey=" + createData.HostKey
	hostConn, _, err := websocket.DefaultDialer.Dial(hostWsURL, nil)
	if err != nil {
		t.Fatalf("host dial failed: %v", err)
	}
	defer hostConn.Close()

	// 3. Connect Student 1 without a client-generated identity.
	s1WsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + createData.RoomID
	s1Conn, _, err := websocket.DefaultDialer.Dial(s1WsURL, nil)
	if err != nil {
		t.Fatalf("s1 dial failed: %v", err)
	}
	defer s1Conn.Close()
	s1State := readRoomState(t, s1Conn)
	if s1State.ParticipantID == "" || s1State.ParticipantToken == "" {
		t.Fatalf("s1 did not receive an identity: %+v", s1State)
	}

	// Add a question directly via manager
	q, err := mgr.AddQuestion(createData.RoomID, "sess_author", "Pinguin", "Test Question?")
	if err != nil {
		t.Fatalf("failed to add question: %v", err)
	}

	// 4. Host upvotes question -> upvotes = 1
	if err := hostConn.WriteJSON(map[string]string{
		"action":     model.ActionUpvoteQuestion,
		"questionId": q.ID,
	}); err != nil {
		t.Fatalf("host upvote failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	r, _ := mgr.GetRoom(createData.RoomID)
	r.RLock()
	if r.Questions[q.ID].Upvotes != 1 {
		t.Fatalf("expected 1 upvote after host vote, got %d", r.Questions[q.ID].Upvotes)
	}
	r.RUnlock()

	// 5. Student 1 upvotes question -> upvotes must increase to 2!
	if err := s1Conn.WriteJSON(map[string]string{
		"action":     model.ActionUpvoteQuestion,
		"questionId": q.ID,
	}); err != nil {
		t.Fatalf("s1 upvote failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	r.RLock()
	if r.Questions[q.ID].Upvotes != 2 {
		t.Fatalf("expected 2 upvotes after student 1 vote, got %d", r.Questions[q.ID].Upvotes)
	}
	r.RUnlock()

	// 6. Connect Student 2 without a client-generated identity.
	s2WsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + createData.RoomID
	s2Conn, _, err := websocket.DefaultDialer.Dial(s2WsURL, nil)
	if err != nil {
		t.Fatalf("s2 dial failed: %v", err)
	}
	defer s2Conn.Close()
	s2State := readRoomState(t, s2Conn)
	if s2State.ParticipantID == "" || s2State.ParticipantToken == "" {
		t.Fatalf("s2 did not receive an identity: %+v", s2State)
	}
	if s2State.ParticipantID == s1State.ParticipantID || s2State.ParticipantToken == s1State.ParticipantToken {
		t.Fatalf("students received the same identity: %+v vs %+v", s1State, s2State)
	}

	// Student 2 upvotes -> upvotes must increase to 3!
	if err := s2Conn.WriteJSON(map[string]string{
		"action":     model.ActionUpvoteQuestion,
		"questionId": q.ID,
	}); err != nil {
		t.Fatalf("s2 upvote failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	r.RLock()
	if r.Questions[q.ID].Upvotes != 3 {
		t.Fatalf("expected 3 upvotes after student 2 vote, got %d", r.Questions[q.ID].Upvotes)
	}
	r.RUnlock()

	// 7. Student 1 un-upvotes -> upvotes must decrease back to 2, leaving Host & Student 2 votes intact
	if err := s1Conn.WriteJSON(map[string]string{
		"action":     model.ActionUpvoteQuestion,
		"questionId": q.ID,
	}); err != nil {
		t.Fatalf("s1 toggle upvote failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	r.RLock()
	if r.Questions[q.ID].Upvotes != 2 {
		t.Fatalf("expected 2 upvotes after student 1 un-vote, got %d", r.Questions[q.ID].Upvotes)
	}
	r.RUnlock()
}

