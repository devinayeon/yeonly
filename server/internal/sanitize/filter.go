package sanitize

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
	mdImageRegex    = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	whitespaceRegex = regexp.MustCompile(`[\t\f\v ]+`)
	blankLinesRegex = regexp.MustCompile(`\n{3,}`)
)

// Profanity list in Indonesian and English
var profanities = []string{
	"anjing", "babi", "bangsat", "kontol", "memek", "jembut", "pantek", "itil",
	"ngentot", "tai", "bajingan", "perek", "kampret", "tolol", "goblok", "bego",
	"fuck", "shit", "bitch", "asshole", "cunt", "dick", "pussy", "bastard",
}

// CleanText strips HTML, strips markdown images, normalizes spaces, and checks length
func CleanText(raw string) (string, error) {
	// Strip HTML tags
	cleaned := htmlTagRegex.ReplaceAllString(raw, "")

	// Strip Markdown images
	cleaned = mdImageRegex.ReplaceAllString(cleaned, "")

	// Unescape HTML entities
	cleaned = html.UnescapeString(cleaned)

	// Normalize horizontal spacing while preserving the composer's explicit line breaks.
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")
	lines := strings.Split(cleaned, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	cleaned = blankLinesRegex.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	cleaned = strings.TrimSpace(cleaned)

	// Validate length
	runeLen := len([]rune(cleaned))
	if runeLen < 5 {
		return "", errors.New("pertanyaan terlalu singkat (minimal 5 karakter)")
	}
	if runeLen > 300 {
		return "", errors.New("pertanyaan melebihi batas maksimal 300 karakter")
	}

	// Profanity check
	lower := strings.ToLower(cleaned)
	for _, badWord := range profanities {
		// match whole word or surrounded by spaces/punctuation
		if strings.Contains(lower, badWord) {
			return "", errors.New("pertanyaan terdeteksi mengandung kata kurang pantas, mohon gunakan bahasa yang santun")
		}
	}

	return cleaned, nil
}

// RateLimiter manages submission cooldowns per session ID
type RateLimiter struct {
	mu       sync.Mutex
	lastPost map[string]time.Time
	cooldown time.Duration
}

// NewRateLimiter creates a new RateLimiter
func NewRateLimiter(cooldown time.Duration) *RateLimiter {
	rl := &RateLimiter{
		lastPost: make(map[string]time.Time),
		cooldown: cooldown,
	}

	// Background cleaner for stale session timestamps
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for sid, t := range rl.lastPost {
				if now.Sub(t) > 30*time.Minute {
					delete(rl.lastPost, sid)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Allow checks if the session can submit, or returns time to wait
func (rl *RateLimiter) Allow(sessionID string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	last, exists := rl.lastPost[sessionID]
	if !exists {
		rl.lastPost[sessionID] = time.Now()
		return true, 0
	}

	elapsed := time.Since(last)
	if elapsed < rl.cooldown {
		remaining := int((rl.cooldown - elapsed).Seconds()) + 1
		return false, remaining
	}

	rl.lastPost[sessionID] = time.Now()
	return true, 0
}

// FormatWaitMessage returns friendly wait message
func FormatWaitMessage(seconds int) string {
	return fmt.Sprintf("Mohon tunggu %d detik sebelum mengirim pertanyaan lagi.", seconds)
}
