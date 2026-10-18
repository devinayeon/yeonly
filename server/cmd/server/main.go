package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"yeonly-server/internal/api"
	"yeonly-server/internal/room"
	"yeonly-server/internal/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Try resolving web dist directory
	webDist := os.Getenv("WEB_DIST")
	if webDist == "" {
		candidates := []string{
			"./web/dist",
			"../web/dist",
			"../../web/dist",
			"web/dist",
		}
		for _, c := range candidates {
			if stat, err := os.Stat(c); err == nil && stat.IsDir() {
				webDist = c
				break
			}
		}
	}

	roomMgr := room.NewManager()
	hub := ws.NewHub(roomMgr)
	go hub.Run()

	server := api.NewServer(roomMgr, hub)
	handler := server.SetupRoutes(webDist)

	addr := fmt.Sprintf(":%s", port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("==================================================")
		fmt.Println("  ⚡ Yeonly Backend — Tanya Dosen Tanpa Waswas")
		fmt.Printf("  🌐 Server running at: http://localhost:%s\n", port)
		if webDist != "" {
			abs, _ := filepath.Abs(webDist)
			fmt.Printf("  📦 Serving static frontend from: %s\n", abs)
		} else {
			fmt.Println("  💡 Running in API-only mode (frontend via Vite dev server)")
		}
		fmt.Println("==================================================")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Yeonly server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Yeonly server stopped.")
}
