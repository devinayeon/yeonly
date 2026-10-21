package api

import (
	"os"
	"strings"
)

// allowedOrigins returns the configured browser origins. An empty result means
// every origin is allowed, which preserves local development and the previous
// wildcard behaviour until ALLOWED_ORIGINS is set in production.
func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" || raw == "*" {
		return nil
	}

	origins := make([]string, 0, 2)
	for _, part := range strings.Split(raw, ",") {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, strings.TrimSuffix(origin, "/"))
		}
	}
	return origins
}

func originAllowed(origin string) bool {
	origins := allowedOrigins()
	if len(origins) == 0 {
		return true
	}

	origin = strings.TrimSuffix(origin, "/")
	for _, allowed := range origins {
		if origin == allowed {
			return true
		}
	}
	return false
}
