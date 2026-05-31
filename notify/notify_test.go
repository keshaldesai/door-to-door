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

	"github.com/keshaldesai/door-to-door/model"
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
		Subway:  model.SubwayLeg{Line: "X", Status: "Delays"},
		Outbound: model.TrainLeg{Origin: "Home", Dest: "Work", Trains: []model.Train{
			{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "Delayed 6m", Track: "2"},
		}},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	for _, want := range []string{"7:10 AM", "Delayed 6m", "Rain", "X train", "Delays", "Home -> Work", "http://localhost:8080"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary missing %q:\n%s", want, s)
		}
	}
}

func TestSummaryIncludesLeaveTimeWhenOffsetSet(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	snap := model.Snapshot{
		Outbound: model.TrainLeg{
			LeaveOffsetMin: 20,
			Trains: []model.Train{
				{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "On time", Track: "3"},
			},
		},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	// Leave 7:10 - 20 minutes = 6:50.
	if !strings.Contains(s, "leave 6:50 AM") {
		t.Fatalf("expected 'leave 6:50 AM' in summary:\n%s", s)
	}
	if !strings.Contains(s, "track 3") {
		t.Fatalf("expected 'track 3' in summary:\n%s", s)
	}
}

func TestSummaryWarnsWhenTrackDiffersFromExpected(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	snap := model.Snapshot{
		Outbound: model.TrainLeg{
			LeaveOffsetMin: 20,
			ExpectedTrack:  "3",
			Trains: []model.Train{
				{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "On time", Track: "5"},
			},
		},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	if !strings.Contains(s, "track 5") || !strings.Contains(s, "expected 3") {
		t.Fatalf("expected 'track 5 ... expected 3' in summary:\n%s", s)
	}
}

func TestSummaryOmitsTrackWhenNotPosted(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	snap := model.Snapshot{
		Outbound: model.TrainLeg{
			LeaveOffsetMin: 20,
			ExpectedTrack:  "3",
			Trains: []model.Train{
				{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "On time"},
			},
		},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	if strings.Contains(s, "track") {
		t.Fatalf("expected no track text when track unposted:\n%s", s)
	}
	if !strings.Contains(s, "leave 6:50 AM") {
		t.Fatalf("expected leave time still shown:\n%s", s)
	}
}

func TestSummaryOmitsLeaveWhenOffsetZero(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	snap := model.Snapshot{
		Outbound: model.TrainLeg{
			Trains: []model.Train{
				{Departure: time.Date(2026, 5, 25, 7, 10, 0, 0, loc), Status: "On time", Track: "3"},
			},
		},
	}
	s := Summary(snap, "morning", "http://localhost:8080")
	if strings.Contains(s, "leave") {
		t.Fatalf("expected no leave text when offset zero:\n%s", s)
	}
}
