// Package server serves the embedded dashboard and a /api/status JSON endpoint
// backed by a periodically refreshed in-memory snapshot.
package server

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/keshaldesai/door-to-door/model"
)

//go:embed all:static
var staticFS embed.FS

// Server holds the cached snapshot and the function that builds a fresh one.
type Server struct {
	build func(context.Context) model.Snapshot

	mu   sync.RWMutex
	snap model.Snapshot
}

// New creates a Server. build assembles a fresh snapshot.
func New(build func(context.Context) model.Snapshot) *Server {
	return &Server{build: build}
}

// Snapshot returns the current cached snapshot.
func (s *Server) Snapshot() model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Server) refresh(ctx context.Context) {
	snap := s.build(ctx)
	s.mu.Lock()
	s.snap = snap
	s.mu.Unlock()
}

// RefreshLoop refreshes immediately, then every interval until ctx is done.
func (s *Server) RefreshLoop(ctx context.Context, interval time.Duration) {
	s.refresh(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refresh(ctx)
		}
	}
}

// Handler returns the HTTP handler: the dashboard at / and JSON at /api/status.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.Snapshot()); err != nil {
			log.Printf("encode status: %v", err)
		}
	})
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	return mux
}
