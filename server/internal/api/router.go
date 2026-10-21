package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) SetupRoutes(webDistDir string) http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.CreateRoomHandler(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan")
		}
	})

	mux.HandleFunc("/api/rooms/public", s.ListPublicRoomsHandler)

	mux.HandleFunc("/api/rooms/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/recap") {
			s.ExportRecapHandler(w, r)
			return
		}
		s.CheckRoomHandler(w, r)
	})

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.WebSocketHandler)

	// Health check
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "app": "Yeonly"})
	}
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/health", healthHandler)

	// Static frontend file server (SPA support)
	if webDistDir != "" {
		if stat, err := os.Stat(webDistDir); err == nil && stat.IsDir() {
			fileServer := http.FileServer(http.Dir(webDistDir))
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// If URL starts with /api or /ws, let mux handle it
				if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
					http.NotFound(w, r)
					return
				}

				path := filepath.Join(webDistDir, filepath.Clean(r.URL.Path))
				if _, err := os.Stat(path); os.IsNotExist(err) {
					// Fallback to index.html for SPA client-side routing
					http.ServeFile(w, r, filepath.Join(webDistDir, "index.html"))
					return
				}
				fileServer.ServeHTTP(w, r)
			})
		}
	}

	// Wrap with CORS middleware
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		switch {
		case len(allowedOrigins()) == 0:
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case originAllowed(origin):
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
