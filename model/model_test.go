package model

import (
	"encoding/json"
	"reflect"
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
		Outbound:    TrainLeg{Origin: "Home", Dest: "Work", Source: "realtime", LeaveOffsetMin: 20, ExpectedTrack: "3"},
		Inbound:     TrainLeg{Origin: "Work", Dest: "Home", Source: "scheduled", LeaveOffsetMin: 30},
	}
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(snap, got) {
		t.Fatalf("round trip mismatch\ngot:  %+v\nwant: %+v", got, snap)
	}
}
