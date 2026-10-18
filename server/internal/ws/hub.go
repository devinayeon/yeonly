package ws

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	"yeonly-server/internal/model"
	"yeonly-server/internal/room"
)

type Hub struct {
	Manager    *room.Manager
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool
	typing     map[*Client]typingState
	Register   chan *Client
	Unregister chan *Client
}

func NewHub(manager *room.Manager) *Hub {
	return &Hub{
		Manager:    manager,
		rooms:      make(map[string]map[*Client]bool),
		typing:     make(map[*Client]typingState),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			h.expireTyping(now)
			h.closeExpiredRooms()
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.Unregister:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) registerClient(c *Client) {
	h.mu.Lock()
	c.PresenceID = rand.Text()
	if _, ok := h.rooms[c.RoomID]; !ok {
		h.rooms[c.RoomID] = make(map[*Client]bool)
	}
	h.rooms[c.RoomID][c] = true
	userCount := h.activeUsersLocked(c.RoomID)
	h.mu.Unlock()

	// Send initial room state to the newly connected client
	h.SendInitialState(c)

	// Broadcast updated user count to room
	h.BroadcastToRoom(c.RoomID, model.WSEvent{
		Type: model.EventUserCount,
		Payload: map[string]int{
			"count": userCount,
		},
	})
}

func (h *Hub) unregisterClient(c *Client) {
	h.mu.Lock()
	h.stopTypingLocked(c)
	var userCount int
	if clients, ok := h.rooms[c.RoomID]; ok {
		if _, exists := clients[c]; exists {
			delete(clients, c)
			close(c.Send)
		}
		userCount = h.activeUsersLocked(c.RoomID)
		if userCount == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	h.mu.Unlock()

	if userCount > 0 {
		h.BroadcastToRoom(c.RoomID, model.WSEvent{
			Type: model.EventUserCount,
			Payload: map[string]int{
				"count": userCount,
			},
		})
	}
}

func (h *Hub) BroadcastToRoom(roomID string, event model.WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.broadcastLocked(roomID, event, nil)
}

// The lock protects both membership and channel lifetime during non-blocking sends.
func (h *Hub) broadcastLocked(roomID string, event model.WSEvent, except *Client) {
	msgBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("json marshal error: %v", err)
		return
	}

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}
	for c := range clients {
		if c == except {
			continue
		}
		select {
		case c.Send <- msgBytes:
		default:
			// Buffer full, unregister asynchronously
			go func(cl *Client) {
				h.Unregister <- cl
			}(c)
		}
	}
}

func (h *Hub) SendToClient(c *Client, event model.WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.rooms[c.RoomID][c] {
		return
	}
	msgBytes, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case c.Send <- msgBytes:
	default:
	}
}

func (h *Hub) SendInitialState(c *Client) {
	r, exists := h.Manager.GetRoom(c.RoomID)
	if !exists {
		h.SendToClient(c, model.WSEvent{
			Type:    model.EventError,
			Payload: "Ruang kuliah tidak ditemukan atau sudah ditutup.",
		})
		return
	}

	r.RLock()
	qList := make([]*model.Question, 0, len(r.Questions))
	votedQuestionIDs := make([]string, 0)
	voterKey := c.VoterKey()
	for _, q := range r.Questions {
		qList = append(qList, q.ForViewer(voterKey))
		if q.Voters != nil && q.Voters[voterKey] {
			votedQuestionIDs = append(votedQuestionIDs, q.ID)
		}
	}
	isFrozen := r.IsFrozen
	roomName := r.Name
	r.RUnlock()

	// Sort questions by upvotes descending
	sort.Slice(qList, func(i, j int) bool {
		if qList[i].Upvotes == qList[j].Upvotes {
			return qList[i].CreatedAt.Before(qList[j].CreatedAt)
		}
		return qList[i].Upvotes > qList[j].Upvotes
	})

	h.mu.RLock()
	activeCount := h.activeUsersLocked(c.RoomID)
	typingUsers := make([]model.TypingPayload, 0)
	for peer, state := range h.typing {
		if peer != c && peer.RoomID == c.RoomID && time.Since(state.updatedAt) < typingTTL {
			typingUsers = append(typingUsers, typingPayload(peer, true))
		}
	}
	h.mu.RUnlock()

	pulseStats := h.Manager.GetPulseStats(c.RoomID)

	payload := model.RoomStatePayload{
		ParticipantID:    c.SessionID,
		ParticipantToken: c.Token,
		ID:               c.RoomID,
		Name:             roomName,
		IsFrozen:         isFrozen,
		Questions:        qList,
		VotedQuestionIDs: votedQuestionIDs,
		PulseStats:       pulseStats,
		ActiveUsers:      activeCount,
		UserAlias:        c.UserAlias,
		IsHost:           c.IsHost,
		TypingUsers:      typingUsers,
	}

	h.SendToClient(c, model.WSEvent{
		Type:    model.EventRoomState,
		Payload: payload,
	})
}

// Inbound action payload structures
type actionEnvelope struct {
	Action     string `json:"action"`
	RequestID  string `json:"requestId,omitempty"`
	IsTyping   bool   `json:"isTyping,omitempty"`
	Text       string `json:"text,omitempty"`
	QuestionID string `json:"questionId,omitempty"`
	Vote       string `json:"vote,omitempty"`
	HostKey    string `json:"hostKey,omitempty"`
}

func (h *Hub) HandleAction(c *Client, raw []byte) {
	var env actionEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		h.SendToClient(c, model.WSEvent{
			Type:    model.EventError,
			Payload: "Format payload pesan tidak valid.",
		})
		return
	}

	if len(env.Action) >= 5 && env.Action[:5] == "HOST_" {
		if !c.IsHost {
			h.SendToClient(c, model.WSEvent{Type: model.EventError, Payload: "Akses host diperlukan."})
			return
		}
		env.HostKey = c.HostKey
	}
	switch env.Action {
	case "HOST_CLOSE":
		if err := h.Manager.DeleteRoom(c.RoomID, env.HostKey); err != nil {
			h.SendToClient(c, model.WSEvent{Type: model.EventError, Payload: err.Error()})
			return
		}
		h.CloseRoom(c.RoomID)

	case model.ActionTyping:
		h.setTyping(c, env.IsTyping)

	case model.ActionSubmitQuestion:
		q, err := h.Manager.AddQuestion(c.RoomID, c.SessionID, c.UserAlias, env.Text)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:      model.EventError,
				Payload:   err.Error(),
				RequestID: env.RequestID,
			})
			return
		}
		h.setTyping(c, false)
		// Broadcast new question to all clients in the room
		h.BroadcastQuestion(c.RoomID, q.ID, model.EventQuestionAdded)
		if env.RequestID != "" {
			h.SendToClient(c, model.WSEvent{
				Type:      model.EventQuestionAccepted,
				RequestID: env.RequestID,
			})
		}

	case model.ActionUpvoteQuestion:
		voterKey := c.VoterKey()
		q, err := h.Manager.UpvoteQuestion(c.RoomID, env.QuestionID, voterKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		// Broadcast updated question to everyone in the room
		h.BroadcastQuestion(c.RoomID, q.ID, model.EventQuestionUpdated)

	case model.ActionUpdatePulse:
		stats, err := h.Manager.SetPulse(c.RoomID, c.SessionID, model.PulseVote(env.Vote))
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		// Broadcast updated pulse stats
		h.BroadcastToRoom(c.RoomID, model.WSEvent{
			Type:    model.EventPulseUpdated,
			Payload: stats,
		})

	case model.ActionHostPin:
		hostKey := env.HostKey
		if hostKey == "" && c.IsHost {
			// Host verification handled by manager
		}
		q, err := h.Manager.SetQuestionStatus(c.RoomID, env.QuestionID, model.StatusPinned, hostKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		h.BroadcastQuestion(c.RoomID, q.ID, model.EventQuestionUpdated)

	case model.ActionHostResolve:
		q, err := h.Manager.SetQuestionStatus(c.RoomID, env.QuestionID, model.StatusAnswered, env.HostKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		h.BroadcastQuestion(c.RoomID, q.ID, model.EventQuestionUpdated)

	case model.ActionHostDelete:
		err := h.Manager.DeleteQuestion(c.RoomID, env.QuestionID, env.HostKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		h.BroadcastToRoom(c.RoomID, model.WSEvent{
			Type: model.EventQuestionDeleted,
			Payload: map[string]string{
				"questionId": env.QuestionID,
			},
		})

	case model.ActionHostFreeze:
		isFrozen, err := h.Manager.ToggleFreeze(c.RoomID, env.HostKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		if isFrozen {
			h.clearRoomTyping(c.RoomID)
		}
		h.BroadcastToRoom(c.RoomID, model.WSEvent{
			Type: model.EventRoomFrozen,
			Payload: map[string]bool{
				"isFrozen": isFrozen,
			},
		})

	case model.ActionHostClearPulse:
		stats, err := h.Manager.ClearPulse(c.RoomID, env.HostKey)
		if err != nil {
			h.SendToClient(c, model.WSEvent{
				Type:    model.EventError,
				Payload: err.Error(),
			})
			return
		}
		h.BroadcastToRoom(c.RoomID, model.WSEvent{
			Type:    model.EventPulseUpdated,
			Payload: stats,
		})

	default:
		h.SendToClient(c, model.WSEvent{
			Type:    model.EventError,
			Payload: "Aksi tidak dikenali.",
		})
	}
}
