package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"yeonly-server/internal/model"
	"yeonly-server/internal/room"
	"yeonly-server/internal/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Same-origin development and unset configuration stay permissive; a
		// deployed backend should set ALLOWED_ORIGINS to the public frontend.
		return originAllowed(r.Header.Get("Origin"))
	},
}

type Server struct {
	Manager *room.Manager
	Hub     *ws.Hub
}

func NewServer(manager *room.Manager, hub *ws.Hub) *Server {
	return &Server{
		Manager: manager,
		Hub:     hub,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// CreateRoomHandler creates a new ephemeral room
func (s *Server) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan")
		return
	}

	var req struct {
		Name     string `json:"name"`
		IsPublic bool   `json:"isPublic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default values if empty body
		req.Name = "Ruang Diskusi Anonim"
		req.IsPublic = true
	}

	roomObj, hostKey, err := s.Manager.CreateRoom(req.Name, req.IsPublic)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"roomId":    roomObj.ID,
		"name":      roomObj.Name,
		"isPublic":  roomObj.IsPublic,
		"hostKey":   hostKey,
		"createdAt": roomObj.CreatedAt,
	})
}

// ListPublicRoomsHandler returns public active rooms
func (s *Server) ListPublicRoomsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan")
		return
	}

	rooms := s.Manager.ListPublicRooms()
	for i := range rooms {
		rooms[i].ActiveUsers = s.Hub.ActiveUsers(rooms[i].ID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// CheckRoomHandler validates room code existence
func (s *Server) CheckRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan")
		return
	}

	// Extract room ID from path: /api/rooms/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Kode ruang diperlukan")
		return
	}
	roomID := strings.ToUpper(parts[2])

	roomObj, exists := s.Manager.GetRoom(roomID)
	if !exists {
		writeError(w, http.StatusNotFound, "Ruang kuliah tidak ditemukan atau sudah berakhir")
		return
	}

	roomObj.RLock()
	defer roomObj.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"roomId":        roomObj.ID,
		"name":          roomObj.Name,
		"isFrozen":      roomObj.IsFrozen,
		"questionCount": len(roomObj.Questions),
		"createdAt":     roomObj.CreatedAt,
	})
}

// ExportRecapHandler downloads Markdown recap
func (s *Server) ExportRecapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan")
		return
	}

	// Path: /api/rooms/{id}/recap
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeError(w, http.StatusBadRequest, "Kode ruang diperlukan")
		return
	}
	roomID := strings.ToUpper(parts[2])

	recap, err := s.Manager.ExportRecap(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	filename := fmt.Sprintf("yeonly-rekap-%s.md", roomID)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(recap))
}

// WebSocketHandler upgrades connection and attaches client
func (s *Server) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	roomID := strings.ToUpper(strings.TrimSpace(q.Get("room")))
	sessionID := strings.TrimSpace(q.Get("session"))
	hostKey := strings.TrimSpace(q.Get("hostKey"))

	if roomID == "" {
		http.Error(w, "Kode ruang diperlukan", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	participant, err := s.Manager.JoinRoom(roomID, sessionID, hostKey)
	if err != nil {
		typeName, code := "IDENTITY_EXPIRED", 4401
		if err == room.ErrRoomMissing {
			typeName, code = "ROOM_CLOSED", 4404
		}
		_ = conn.WriteJSON(model.WSEvent{Type: typeName, Payload: err.Error()})
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, ""))
		_ = conn.Close()
		return
	}

	client := &ws.Client{
		Hub:       s.Hub,
		Conn:      conn,
		RoomID:    roomID,
		SessionID: participant.ID,
		Token:     participant.Token,
		HostKey:   hostKey,
		UserAlias: participant.Alias,
		IsHost:    participant.IsHost,
		Send:      make(chan []byte, 256),
	}

	s.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
