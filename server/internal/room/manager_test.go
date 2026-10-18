package room

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"yeonly-server/internal/model"
)

func TestRoomLifecycle(t *testing.T) {
	mgr := NewManager()

	// 1. Create room
	room, hostKey, err := mgr.CreateRoom("Pemrograman Lanjut", true)
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	if len(room.ID) != 6 {
		t.Fatalf("expected 6-char code, got %s", room.ID)
	}
	if !mgr.VerifyHost(room.ID, hostKey) {
		t.Fatal("host key should verify")
	}

	// 2. Add question
	q, err := mgr.AddQuestion(room.ID, "session-A", "Pinguin Santai #01", "Apakah goroutine sama dengan thread OS?")
	if err != nil {
		t.Fatalf("failed to add question: %v", err)
	}
	if q.Upvotes != 0 {
		t.Fatalf("expected 0 upvotes, got %d", q.Upvotes)
	}

	// 3. Upvote question
	q2, err := mgr.UpvoteQuestion(room.ID, q.ID, "session-B")
	if err != nil {
		t.Fatalf("failed to upvote: %v", err)
	}
	if q2.Upvotes != 1 {
		t.Fatalf("expected 1 upvote, got %d", q2.Upvotes)
	}

	// 4. Toggle upvote (undo)
	q3, err := mgr.UpvoteQuestion(room.ID, q.ID, "session-B")
	if err != nil {
		t.Fatalf("failed to toggle upvote: %v", err)
	}
	if q3.Upvotes != 0 {
		t.Fatalf("expected 0 upvotes after undo, got %d", q3.Upvotes)
	}

	// 5. Host pin question
	pinned, err := mgr.SetQuestionStatus(room.ID, q.ID, model.StatusPinned, hostKey)
	if err != nil {
		t.Fatalf("failed to pin question: %v", err)
	}
	if pinned.Status != model.StatusPinned {
		t.Fatalf("expected status pinned, got %s", pinned.Status)
	}

	// 6. Pulse feedback
	stats, err := mgr.SetPulse(room.ID, "session-A", model.PulseUnderstood)
	if err != nil {
		t.Fatalf("failed to set pulse: %v", err)
	}
	if stats.Paham != 1 || stats.Total != 1 {
		t.Fatalf("expected 1 paham, got %+v", stats)
	}

	// 7. Export recap
	recap, err := mgr.ExportRecap(room.ID)
	if err != nil {
		t.Fatalf("failed to export recap: %v", err)
	}
	if len(recap) == 0 {
		t.Fatal("recap should not be empty")
	}

	// 8. Host freeze
	frozen, err := mgr.ToggleFreeze(room.ID, hostKey)
	if err != nil || !frozen {
		t.Fatalf("expected room to be frozen, got %v, err: %v", frozen, err)
	}

	// Adding question while frozen should fail
	_, err = mgr.AddQuestion(room.ID, "session-C", "Kucing #02", "Pertanyaan saat freeze?")
	if err == nil {
		t.Fatal("expected error when adding question to frozen room")
	}
}

func TestConcurrentUpvotesAndQuestions(t *testing.T) {
	mgr := NewManager()
	room, _, err := mgr.CreateRoom("Test Concurrency", true)
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}

	// Initial question
	q, err := mgr.AddQuestion(room.ID, "host-sess", "Author", "Pertanyaan untuk tes race condition")
	if err != nil {
		t.Fatalf("failed to create initial question: %v", err)
	}

	var wg sync.WaitGroup
	workers := 50

	// 50 concurrent students upvoting
	for i := 0; i < workers; i++ {
		wg.Add(1)
		sessID := fmt.Sprintf("student-%d", i)
		go func(sid string) {
			defer wg.Done()
			_, _ = mgr.UpvoteQuestion(room.ID, q.ID, sid)
			_, _ = mgr.SetPulse(room.ID, sid, model.PulseUnderstood)
		}(sessID)
	}

	wg.Wait()

	r, _ := mgr.GetRoom(room.ID)
	r.RLock()
	finalQ := r.Questions[q.ID]
	if finalQ.Upvotes != workers {
		t.Fatalf("expected %d upvotes, got %d", workers, finalQ.Upvotes)
	}
	r.RUnlock()

	stats := mgr.GetPulseStats(room.ID)
	if stats.Total != workers {
		t.Fatalf("expected %d pulse responses, got %d", workers, stats.Total)
	}
}

func TestCleanupIdleRooms(t *testing.T) {
	mgr := NewManager()
	room, _, _ := mgr.CreateRoom("Idle Room", true)

	// Artificially age the room
	room.Lock()
	room.LastActiveAt = time.Now().Add(-3 * time.Hour)
	room.Unlock()

	deleted := mgr.CleanupIdleRooms(2 * time.Hour)
	if len(deleted) != 1 || deleted[0] != room.ID {
		t.Fatalf("expected room %s to be cleaned up, got %v", room.ID, deleted)
	}

	if _, exists := mgr.GetRoom(room.ID); exists {
		t.Fatal("room should have been deleted")
	}
}
