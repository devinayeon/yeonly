package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"yeonly-server/internal/model"
	"yeonly-server/internal/sanitize"
)

const (
	codeChars = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLen   = 6
)

type Manager struct {
	mu          sync.RWMutex
	rooms       map[string]*model.Room
	rateLimiter *sanitize.RateLimiter
}

func NewManager() *Manager {
	m := &Manager{
		rooms:       make(map[string]*model.Room),
		rateLimiter: sanitize.NewRateLimiter(8 * time.Second),
	}

	// Start automated garbage collection
	go m.startCleaner(10*time.Minute, 2*time.Hour)

	return m
}

func (m *Manager) generateCode() string {
	b := make([]byte, codeLen)
	_, _ = rand.Read(b)
	var sb strings.Builder
	for i := 0; i < codeLen; i++ {
		sb.WriteByte(codeChars[int(b[i])%len(codeChars)])
	}
	return sb.String()
}

func (m *Manager) generateHostKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateRoom creates an ephemeral room with unique 6-character code
func (m *Manager) CreateRoom(name string, isPublic bool) (*model.Room, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanedName := strings.TrimSpace(name)
	if cleanedName == "" {
		cleanedName = "Ruang Diskusi Anonim"
	}
	if len([]rune(cleanedName)) > 60 {
		cleanedName = string([]rune(cleanedName)[:60])
	}

	var code string
	for attempts := 0; attempts < 100; attempts++ {
		c := m.generateCode()
		if _, exists := m.rooms[c]; !exists {
			code = c
			break
		}
	}
	if code == "" {
		return nil, "", errors.New("gagal membuat kode ruang, silakan coba lagi")
	}

	hostKey := m.generateHostKey()
	now := time.Now()

	room := &model.Room{
		Members:         make(map[string]model.Participant),
		HostParticipant: model.Participant{ID: rand.Text(), Token: rand.Text(), Alias: "Yeonly", IsHost: true},
		ID:              code,
		Name:            cleanedName,
		HostKey:         hostKey,
		CreatedAt:       now,
		LastActiveAt:    now,
		IsFrozen:        false,
		IsPublic:        isPublic,
		Questions:       make(map[string]*model.Question),
		PulseVotes:      make(map[string]model.PulseVote),
	}

	m.rooms[code] = room
	return room, hostKey, nil
}

// GetRoom retrieves a room by its code
func (m *Manager) GetRoom(id string) (*model.Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room, exists := m.rooms[strings.ToUpper(strings.TrimSpace(id))]
	return room, exists
}

// ListPublicRooms returns active public rooms
func (m *Manager) ListPublicRooms() []model.PublicRoomSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []model.PublicRoomSummary
	for _, r := range m.rooms {
		r.RLock()
		if r.IsPublic {
			list = append(list, model.PublicRoomSummary{
				ID:            r.ID,
				Name:          r.Name,
				QuestionCount: len(r.Questions),
				ActiveUsers:   0, // Filled from live, distinct participants by the API.
				CreatedAt:     r.CreatedAt,
			})
		}
		r.RUnlock()
	}

	// Sort newest first
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return list
}

// VerifyHost checks if hostKey matches
func (m *Manager) VerifyHost(roomID, hostKey string) bool {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return false
	}
	room.RLock()
	defer rUnlockSafe(room)
	return room.HostKey != "" && room.HostKey == strings.TrimSpace(hostKey)
}

func rUnlockSafe(r *model.Room) {
	r.RUnlock()
}

// AddQuestion validates, sanitizes, and stores a new question
func (m *Manager) AddQuestion(roomID, sessionID, alias, rawText string) (*model.Question, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return nil, errors.New("ruang tidak ditemukan atau telah kedaluwarsa")
	}

	room.Lock()
	defer room.Unlock()

	if room.IsFrozen {
		return nil, errors.New("sesi pertanyaan saat ini sedang dijeda oleh dosen/host")
	}

	// Rate limit check
	allowed, waitSec := m.rateLimiter.Allow(sessionID)
	if !allowed {
		return nil, errors.New(sanitize.FormatWaitMessage(waitSec))
	}

	// Sanitize text
	clean, err := sanitize.CleanText(rawText)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	qID := fmt.Sprintf("q_%d", now.UnixNano())

	q := &model.Question{
		AuthorID:    sessionID,
		AuthorRole:  "student",
		ID:          qID,
		RoomID:      room.ID,
		AuthorAlias: alias,
		SessionID:   sessionID,
		Text:        clean,
		Upvotes:     0,
		Voters:      make(map[string]bool),
		Status:      model.StatusUnanswered,
		CreatedAt:   now,
	}

	if sessionID == room.HostParticipant.ID {
		q.AuthorRole = "host"
	}
	room.Questions[qID] = q
	room.LastActiveAt = now

	return q, nil
}

// UpvoteQuestion toggles or adds upvote
func (m *Manager) UpvoteQuestion(roomID, questionID, sessionID string) (*model.Question, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return nil, errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	q, exists := room.Questions[questionID]
	if !exists {
		return nil, errors.New("pertanyaan tidak ditemukan")
	}

	if q.Voters == nil {
		q.Voters = make(map[string]bool)
	}

	if q.Voters[sessionID] {
		// Undo upvote
		delete(q.Voters, sessionID)
		if q.Upvotes > 0 {
			q.Upvotes--
		}
	} else {
		// Add upvote
		q.Voters[sessionID] = true
		q.Upvotes++
	}

	room.LastActiveAt = time.Now()
	return q, nil
}

// SetQuestionStatus updates question status (Host only)
func (m *Manager) SetQuestionStatus(roomID, questionID string, status model.QuestionStatus, hostKey string) (*model.Question, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return nil, errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	if room.HostKey != strings.TrimSpace(hostKey) {
		return nil, errors.New("akses ditolak: hanya dosen/host yang berhak mengubah status pertanyaan")
	}

	q, exists := room.Questions[questionID]
	if !exists {
		return nil, errors.New("pertanyaan tidak ditemukan")
	}

	q.Status = status
	room.LastActiveAt = time.Now()
	return q, nil
}

// DeleteQuestion removes a question (Host only)
func (m *Manager) DeleteQuestion(roomID, questionID, hostKey string) error {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	if room.HostKey != strings.TrimSpace(hostKey) {
		return errors.New("akses ditolak: hanya dosen/host yang berhak menghapus pertanyaan")
	}

	if _, exists := room.Questions[questionID]; !exists {
		return errors.New("pertanyaan tidak ditemukan")
	}

	delete(room.Questions, questionID)
	room.LastActiveAt = time.Now()
	return nil
}

// SetPulse records understanding feedback from a participant
func (m *Manager) SetPulse(roomID, sessionID string, vote model.PulseVote) (model.PulseStats, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return model.PulseStats{}, errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	if room.PulseVotes == nil {
		room.PulseVotes = make(map[string]model.PulseVote)
	}

	room.PulseVotes[sessionID] = vote
	room.LastActiveAt = time.Now()

	return m.calculatePulseStats(room), nil
}

// ClearPulse resets pulse stats (Host only)
func (m *Manager) ClearPulse(roomID, hostKey string) (model.PulseStats, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return model.PulseStats{}, errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	if room.HostKey != strings.TrimSpace(hostKey) {
		return model.PulseStats{}, errors.New("akses ditolak")
	}

	room.PulseVotes = make(map[string]model.PulseVote)
	room.LastActiveAt = time.Now()

	return model.PulseStats{}, nil
}

// ToggleFreeze toggles room freeze (Host only)
func (m *Manager) ToggleFreeze(roomID, hostKey string) (bool, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return false, errors.New("ruang tidak ditemukan")
	}

	room.Lock()
	defer room.Unlock()

	if room.HostKey != strings.TrimSpace(hostKey) {
		return false, errors.New("akses ditolak")
	}

	room.IsFrozen = !room.IsFrozen
	room.LastActiveAt = time.Now()
	return room.IsFrozen, nil
}

// GetPulseStats calculates current percentages
func (m *Manager) GetPulseStats(roomID string) model.PulseStats {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return model.PulseStats{}
	}

	room.RLock()
	defer room.RUnlock()

	return m.calculatePulseStats(room)
}

func (m *Manager) calculatePulseStats(r *model.Room) model.PulseStats {
	var stats model.PulseStats
	total := len(r.PulseVotes)
	if total == 0 {
		return stats
	}

	for _, v := range r.PulseVotes {
		switch v {
		case model.PulseUnderstood:
			stats.Paham++
		case model.PulseConfused:
			stats.Bingung++
		case model.PulseTooFast:
			stats.Kecepatan++
		}
	}

	stats.Total = total
	stats.PahamPct = (float64(stats.Paham) / float64(total)) * 100
	stats.BingungPct = (float64(stats.Bingung) / float64(total)) * 100
	stats.KecepatanPct = (float64(stats.Kecepatan) / float64(total)) * 100

	return stats
}

// ExportRecap generates clean Markdown document of all questions
func (m *Manager) ExportRecap(roomID string) (string, error) {
	room, exists := m.GetRoom(roomID)
	if !exists {
		return "", errors.New("ruang tidak ditemukan")
	}

	room.RLock()
	defer room.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Rekap Tanya Jawab — %s\n\n", room.Name))
	sb.WriteString(fmt.Sprintf("- **Kode Ruang:** `%s`\n", room.ID))
	sb.WriteString(fmt.Sprintf("- **Waktu Dibuat:** %s\n", room.CreatedAt.Format("02 Jan 2006, 15:04 WIB")))
	sb.WriteString(fmt.Sprintf("- **Total Pertanyaan:** %d\n", len(room.Questions)))

	stats := m.calculatePulseStats(room)
	if stats.Total > 0 {
		sb.WriteString(fmt.Sprintf("- **Pemahaman Kelas:** Paham: %.0f%% | Bingung: %.0f%% | Kecepatan: %.0f%% (Total %d respon)\n",
			stats.PahamPct, stats.BingungPct, stats.KecepatanPct, stats.Total))
	}
	sb.WriteString("\n---\n\n")

	// Group questions by status
	var unanswered, pinned, answered []*model.Question
	for _, q := range room.Questions {
		switch q.Status {
		case model.StatusPinned:
			pinned = append(pinned, q)
		case model.StatusAnswered:
			answered = append(answered, q)
		default:
			unanswered = append(unanswered, q)
		}
	}

	// Sort questions by upvotes descending
	sortQuestions := func(list []*model.Question) {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Upvotes == list[j].Upvotes {
				return list[i].CreatedAt.Before(list[j].CreatedAt)
			}
			return list[i].Upvotes > list[j].Upvotes
		})
	}
	sortQuestions(pinned)
	sortQuestions(unanswered)
	sortQuestions(answered)

	writeGroup := func(title string, list []*model.Question) {
		sb.WriteString(fmt.Sprintf("## %s (%d)\n\n", title, len(list)))
		if len(list) == 0 {
			sb.WriteString("_Tidak ada pertanyaan di kategori ini._\n\n")
			return
		}
		for i, q := range list {
			sb.WriteString(fmt.Sprintf("%d. **[%d Upvotes]** %s\n", i+1, q.Upvotes, q.Text))
			sb.WriteString(fmt.Sprintf("   _Oleh: %s • %s_\n\n", q.AuthorAlias, q.CreatedAt.Format("15:04:05")))
		}
	}

	writeGroup("📌 Sedang Dibahas", pinned)
	writeGroup("💬 Pertanyaan Belum Terjawab", unanswered)
	writeGroup("✅ Pertanyaan Telah Dijawab", answered)

	sb.WriteString("\n---\n_Dibuat dengan Yeonly — Platform Chat Anonim untuk Ruang Kuliah._\n")
	return sb.String(), nil
}

// DeleteRoom immediately cleans up a room (Host only)
func (m *Manager) DeleteRoom(roomID, hostKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.rooms[roomID]
	if !exists {
		return errors.New("ruang tidak ditemukan")
	}

	if room.HostKey != strings.TrimSpace(hostKey) {
		return errors.New("akses ditolak")
	}

	delete(m.rooms, roomID)
	return nil
}

// Garbage collector for inactive rooms
func (m *Manager) startCleaner(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		m.CleanupIdleRooms(timeout)
	}
}

// CleanupIdleRooms purges inactive rooms
func (m *Manager) CleanupIdleRooms(timeout time.Duration) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted []string
	now := time.Now()
	for id, r := range m.rooms {
		r.RLock()
		idle := now.Sub(r.LastActiveAt) > timeout
		r.RUnlock()

		if idle {
			delete(m.rooms, id)
			deleted = append(deleted, id)
		}
	}
	return deleted
}
