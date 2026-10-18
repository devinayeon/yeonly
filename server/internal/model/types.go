package model

import (
	"sync"
	"time"
)

type QuestionStatus string

const (
	StatusUnanswered QuestionStatus = "unanswered"
	StatusPinned     QuestionStatus = "pinned"
	StatusAnswered   QuestionStatus = "answered"
)

type PulseVote string

const (
	PulseUnderstood PulseVote = "paham"
	PulseConfused   PulseVote = "bingung"
	PulseTooFast    PulseVote = "kecepatan"
)

type PulseStats struct {
	Paham        int     `json:"paham"`
	Bingung      int     `json:"bingung"`
	Kecepatan    int     `json:"kecepatan"`
	Total        int     `json:"total"`
	PahamPct     float64 `json:"pahamPct"`
	BingungPct   float64 `json:"bingungPct"`
	KecepatanPct float64 `json:"kecepatanPct"`
}

type Question struct {
	ID          string          `json:"id"`
	RoomID      string          `json:"roomId"`
	AuthorAlias string          `json:"authorAlias"`
	AuthorID    string          `json:"authorId"`
	AuthorRole  string          `json:"authorRole"`
	HasVoted    bool            `json:"hasVoted"`
	SessionID   string          `json:"-"`
	Text        string          `json:"text"`
	Upvotes     int             `json:"upvotes"`
	Voters      map[string]bool `json:"-"` // Private participant IDs; never sent to clients.
	Status      QuestionStatus  `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
}

type Room struct {
	Members         map[string]Participant `json:"-"`
	HostParticipant Participant            `json:"-"`
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	HostKey         string                 `json:"-"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastActiveAt    time.Time              `json:"lastActiveAt"`
	IsFrozen        bool                   `json:"isFrozen"`
	IsPublic        bool                   `json:"isPublic"`
	Questions       map[string]*Question   `json:"questions"`
	PulseVotes      map[string]PulseVote   `json:"-"`
	mu              sync.RWMutex
}

// Lock wraps RWMutex
func (r *Room) Lock()    { r.mu.Lock() }
func (r *Room) Unlock()  { r.mu.Unlock() }
func (r *Room) RLock()   { r.mu.RLock() }
func (r *Room) RUnlock() { r.mu.RUnlock() }

// PublicRoomSummary is sent to public lobbies
type PublicRoomSummary struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	QuestionCount int       `json:"questionCount"`
	ActiveUsers   int       `json:"activeUsers"`
	CreatedAt     time.Time `json:"createdAt"`
}

// RoomStatePayload is sent to joining clients
type RoomStatePayload struct {
	ParticipantToken string          `json:"participantToken"`
	ParticipantID    string          `json:"participantId"`
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	IsFrozen         bool            `json:"isFrozen"`
	Questions        []*Question     `json:"questions"`
	VotedQuestionIDs []string        `json:"votedQuestionIds"`
	PulseStats       PulseStats      `json:"pulseStats"`
	ActiveUsers      int             `json:"activeUsers"`
	UserAlias        string          `json:"userAlias"`
	IsHost           bool            `json:"isHost"`
	TypingUsers      []TypingPayload `json:"typingUsers"`
}

// TypingPayload exposes only a connection-scoped ID and anonymous alias, never a draft.
type TypingPayload struct {
	ID       string `json:"id"`
	Alias    string `json:"alias"`
	IsTyping bool   `json:"isTyping"`
}

// WSEvent is standard protocol envelope
type WSEvent struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
}

// Client action constants
const (
	ActionTyping         = "TYPING"
	ActionSubmitQuestion = "SUBMIT_QUESTION"
	ActionUpvoteQuestion = "UPVOTE_QUESTION"
	ActionUpdatePulse    = "UPDATE_PULSE"
	ActionHostPin        = "HOST_PIN"
	ActionHostResolve    = "HOST_RESOLVE"
	ActionHostDelete     = "HOST_DELETE"
	ActionHostFreeze     = "HOST_FREEZE"
	ActionHostClearPulse = "HOST_CLEAR_PULSE"
)

// Participant separates the public room identity from its private bearer credential.
type Participant struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	IsHost bool   `json:"isHost"`
	Token  string `json:"-"`
}

// ForViewer must be called while holding the room lock.
func (q *Question) ForViewer(id string) *Question {
	copy := *q
	copy.HasVoted = q.Voters[id]
	copy.Voters = nil
	return &copy
}

// Server event constants
const (
	EventTypingUpdated    = "TYPING_UPDATED"
	EventRoomState        = "ROOM_STATE"
	EventQuestionAdded    = "QUESTION_ADDED"
	EventQuestionAccepted = "QUESTION_ACCEPTED"
	EventQuestionUpdated  = "QUESTION_UPDATED"
	EventQuestionDeleted  = "QUESTION_DELETED"
	EventPulseUpdated     = "PULSE_UPDATED"
	EventRoomFrozen       = "ROOM_FROZEN"
	EventUserCount        = "USER_COUNT"
	EventVoteUpdated      = "VOTE_UPDATED"
	EventError            = "ERROR"
	EventToast            = "TOAST"
)
