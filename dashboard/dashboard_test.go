package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/keshaldesai/door-to-door/model"
)

func TestBuildAssemblesAllLegsConcurrently(t *testing.T) {
	f := Fetchers{
		Weather:  func(ctx context.Context) model.Weather { return model.Weather{Summary: "Clear"} },
		Drive:    func(ctx context.Context) model.DriveLeg { return model.DriveLeg{DurationMin: 9} },
		Subway:   func(ctx context.Context) model.SubwayLeg { return model.SubwayLeg{Line: "X", Status: "Good Service"} },
		Outbound: func(ctx context.Context) model.TrainLeg { return model.TrainLeg{Origin: "Home", Dest: "Work"} },
		Inbound:  func(ctx context.Context) model.TrainLeg { return model.TrainLeg{Origin: "Work", Dest: "Home"} },
	}
	fixed := time.Date(2026, 5, 25, 7, 0, 0, 0, time.UTC)
	snap := Build(context.Background(), f, func() time.Time { return fixed })

	if !snap.GeneratedAt.Equal(fixed) {
		t.Fatalf("generatedAt = %v", snap.GeneratedAt)
	}
	if snap.Weather.Summary != "Clear" || snap.Drive.DurationMin != 9 {
		t.Fatalf("weather/drive not assembled: %+v", snap)
	}
	if snap.Subway.Status != "Good Service" || snap.Outbound.Origin != "Home" || snap.Inbound.Dest != "Home" {
		t.Fatalf("legs not assembled: %+v", snap)
	}
}

func TestBuildPreservesLegLeaveOffsetAndExpectedTrack(t *testing.T) {
	f := Fetchers{
		Outbound: func(ctx context.Context) model.TrainLeg {
			return model.TrainLeg{LeaveOffsetMin: 20, ExpectedTrack: "3"}
		},
		Inbound: func(ctx context.Context) model.TrainLeg {
			return model.TrainLeg{LeaveOffsetMin: 30}
		},
	}
	snap := Build(context.Background(), f, time.Now)
	if snap.Outbound.LeaveOffsetMin != 20 || snap.Outbound.ExpectedTrack != "3" {
		t.Fatalf("outbound = %+v", snap.Outbound)
	}
	if snap.Inbound.LeaveOffsetMin != 30 || snap.Inbound.ExpectedTrack != "" {
		t.Fatalf("inbound = %+v", snap.Inbound)
	}
}

func TestBuildToleratesNilFetchers(t *testing.T) {
	snap := Build(context.Background(), Fetchers{}, time.Now)
	if snap.Subway.Err == "" {
		t.Fatal("expected error for missing subway fetcher")
	}
}
