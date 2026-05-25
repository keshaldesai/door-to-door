package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSnapshotJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 25, 7, 0, 0, 0, time.UTC)
	snap := Snapshot{
		GeneratedAt: now,
		Weather:     Weather{Summary: "Rain", TempF: 55, PrecipChance: 80},
		Drive:       DriveLeg{DurationMin: 9},
		Subway:      SubwayLeg{Line: "7", Status: "Good Service"},
		Outbound:    TrainLeg{Origin: "Home", Dest: "Work", Source: "realtime"},
		Inbound:     TrainLeg{Origin: "Work", Dest: "Home", Source: "scheduled"},
	}
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Subway.Line != "7" || got.Drive.DurationMin != 9 {
		t.Fatalf("round trip lost data: %+v", got)
	}
}
