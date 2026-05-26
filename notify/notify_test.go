package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keshaldee/commute/model"
)

func TestPostSendsContent(t *testing.T) {
	var gotContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Content string `json:"content"`
		}
		json.Unmarshal(body, &payload)
		gotContent = payload.Content
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := Post(context.Background(), srv.Client(), srv.URL, "hello"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if gotContent != "hello" {
		t.Fatalf("content = %q", gotContent)
	}
}

func TestSummaryMorningMentionsOutboundAndLink(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	snap := model.Snapshot{
		Weather: model.Weather{Summary: "Rain", PrecipChance: 80},
		Drive:   model.DriveLeg{DurationMin: 9},
		Subway:  model.SubwayLeg{Line: "7", Status: "Delays"},
		Outbound: model.TrainLeg{Trains: []model.Train{
			{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "Delayed 6m", Track: "2"},
		}},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	for _, want := range []string{"7:10", "Delayed 6m", "Rain", "subway", "Delays", "http://localhost:8080"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary missing %q:\n%s", want, s)
		}
	}
}
