package sanitize

import (
	"testing"
	"time"
)

func TestCleanText(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		expectErr bool
		expected  string
	}{
		{
			name:      "Valid regular question",
			input:     "Apakah pointer di Go memiliki arithmetics?",
			expectErr: false,
			expected:  "Apakah pointer di Go memiliki arithmetics?",
		},
		{
			name:      "HTML tags stripped",
			input:     "<script>alert('xss')</script>Bolehkah saya bertanya materi tadi?",
			expectErr: false,
			expected:  "alert('xss')Bolehkah saya bertanya materi tadi?",
		},
		{
			name:      "Markdown image stripped",
			input:     "Lihat ini ![gambar](http://bad.com/a.jpg) apa maksud fungsi ini?",
			expectErr: false,
			expected:  "Lihat ini apa maksud fungsi ini?",
		},
		{
			name:      "Too short",
			input:     "Hai",
			expectErr: true,
		},
		{
			name:     "Composer line breaks preserved with normalized spacing",
			input:    "  Apa  itu channel?\r\n\tBerikan   contoh.\n\n\n  Terima kasih.  ",
			expected: "Apa itu channel?\nBerikan contoh.\n\nTerima kasih.",
		},
		{
			name:      "Only whitespace is rejected",
			input:     " \n\r\n\t\n ",
			expectErr: true,
		},
		{
			name:      "Profanity rejected",
			input:     "Materi ini sangat anjing sekali",
			expectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := CleanText(tc.input)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.expectErr && res != tc.expected {
				t.Fatalf("expected '%s', got '%s'", tc.expected, res)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(500 * time.Millisecond)

	ok, _ := rl.Allow("session-1")
	if !ok {
		t.Fatal("first call should be allowed")
	}

	ok2, remaining := rl.Allow("session-1")
	if ok2 {
		t.Fatal("immediate second call should be blocked")
	}
	if remaining <= 0 {
		t.Fatalf("expected remaining > 0, got %d", remaining)
	}

	time.Sleep(550 * time.Millisecond)
	ok3, _ := rl.Allow("session-1")
	if !ok3 {
		t.Fatal("call after cooldown should be allowed")
	}
}
