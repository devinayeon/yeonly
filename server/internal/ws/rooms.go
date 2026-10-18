package ws

import (
	"encoding/json"
	"yeonly-server/internal/model"
)

func (h *Hub) activeUsersLocked(roomID string) int {
	ids := make(map[string]bool)
	for c := range h.rooms[roomID] {
		ids[c.SessionID] = true
	}
	return len(ids)
}

func (h *Hub) ActiveUsers(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.activeUsersLocked(roomID)
}

// Snapshot under the room lock, then send under the hub lock. No shared maps escape.
func (h *Hub) BroadcastQuestion(roomID, questionID, eventType string) {
	r, ok := h.Manager.GetRoom(roomID)
	if !ok {
		return
	}
	r.RLock()
	defer r.RUnlock()
	q, ok := r.Questions[questionID]
	if !ok {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[roomID] {
		message, err := json.Marshal(model.WSEvent{Type: eventType, Payload: q.ForViewer(c.VoterKey())})
		if err != nil {
			continue
		}
		select {
		case c.Send <- message:
		default:
			go func(c *Client) { h.Unregister <- c }(c)
		}
	}
}

func (h *Hub) CloseRoom(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	message, _ := json.Marshal(model.WSEvent{Type: "ROOM_CLOSED", Payload: "Sesi telah berakhir. Pilih ruang lain untuk melanjutkan."})
	for c := range h.rooms[roomID] {
		select {
		case c.Send <- message:
		default:
		}
		delete(h.typing, c)
		close(c.Send)
	}
	delete(h.rooms, roomID)
}

func (h *Hub) closeExpiredRooms() {
	h.mu.RLock()
	ids := make([]string, 0, len(h.rooms))
	for id := range h.rooms {
		ids = append(ids, id)
	}
	h.mu.RUnlock()
	for _, id := range ids {
		if _, ok := h.Manager.GetRoom(id); !ok {
			h.CloseRoom(id)
		}
	}
}
