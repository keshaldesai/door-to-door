package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/keshaldee/commute/model"
)

func TestStatusReturnsCachedSnapshot(t *testing.T) {
	s := New(func(ctx context.Context) model.Snapshot {
		return model.Snapshot{Subway: model.SubwayLeg{Line: "X", Status: "Good Service"}}
	})
	s.refresh(context.Background()) // populate cache once

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d", rec.Code)
	}
	var got model.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Subway.Status != "Good Service" {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestRefreshLoopUpdatesCache(t *testing.T) {
	var n int64
	s := New(func(ctx context.Context) model.Snapshot {
		atomic.AddInt64(&n, 1)
		return model.Snapshot{}
	})
	ctx, cancel := context.WithCancel(context.Background())
	go s.RefreshLoop(ctx, 10*time.Millisecond)
	time.Sleep(55 * time.Millisecond)
	cancel()
	if atomic.LoadInt64(&n) < 2 {
		t.Fatalf("refresh ran %d times, want >= 2", n)
	}
}
