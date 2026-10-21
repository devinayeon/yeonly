package api

import "testing"

func TestOriginAllowlist(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	if !originAllowed("https://anything.example") {
		t.Fatal("empty configuration must allow every origin")
	}

	t.Setenv("ALLOWED_ORIGINS", "*")
	if !originAllowed("https://anything.example") {
		t.Fatal("wildcard configuration must allow every origin")
	}

	t.Setenv("ALLOWED_ORIGINS", "https://nayeonworks.tech, https://www.nayeonworks.tech/")
	if !originAllowed("https://nayeonworks.tech") {
		t.Fatal("listed origin must be allowed")
	}
	if !originAllowed("https://www.nayeonworks.tech") {
		t.Fatal("trailing slash in configuration must be ignored")
	}
	if originAllowed("https://evil.example") {
		t.Fatal("unlisted origin must be rejected")
	}
	if originAllowed("") {
		t.Fatal("requests without an Origin header must be rejected when an allowlist is active")
	}
}
