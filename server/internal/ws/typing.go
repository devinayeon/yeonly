package ws

import (
	"time"

	"yeonly-server/internal/model"
)

const typingTTL = 5 * time.Second
const typingBroadcastInterval = time.Second

type typingState struct {
	updatedAt   time.Time
	broadcastAt time.Time
}

func typingPayload(c *Client, active bool) model.TypingPayload {
	return model.TypingPayload{ID: c.PresenceID, Alias: c.UserAlias, IsTyping: active}
}

func (h *Hub) setTyping(c *Client, active bool) {
	if !active {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.stopTypingLocked(c)
		return
	}
	r, exists := h.Manager.GetRoom(c.RoomID)
	if !exists {
		return
	}
	r.RLock()
	defer r.RUnlock()
	if r.IsFrozen {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.rooms[c.RoomID][c] {
		return
	}
	now := time.Now()
	state := h.typing[c]
	state.updatedAt = now
	if now.Sub(state.broadcastAt) >= typingBroadcastInterval {
		state.broadcastAt = now
		h.broadcastLocked(c.RoomID, model.WSEvent{
			Type:    model.EventTypingUpdated,
			Payload: typingPayload(c, true),
		}, c)
	}
	h.typing[c] = state
}

func (h *Hub) stopTypingLocked(c *Client) {
	if _, exists := h.typing[c]; !exists {
		return
	}
	delete(h.typing, c)
	h.broadcastLocked(c.RoomID, model.WSEvent{
		Type:    model.EventTypingUpdated,
		Payload: typingPayload(c, false),
	}, c)
}

func (h *Hub) clearRoomTyping(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.typing {
		if c.RoomID == roomID {
			h.stopTypingLocked(c)
		}
	}
}

func (h *Hub) expireTyping(now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c, state := range h.typing {
		if now.Sub(state.updatedAt) >= typingTTL {
			h.stopTypingLocked(c)
		}
	}
}
