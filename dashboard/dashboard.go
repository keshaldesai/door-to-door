// Package dashboard assembles the full Snapshot from the data fetchers,
// running them concurrently. Fetchers are injected for testability.
package dashboard

import (
	"context"
	"sync"
	"time"

	"github.com/keshaldee/commute/model"
)

// Fetchers holds one function per data source. A nil fetcher yields a leg with
// an error set, so a missing source never breaks the whole snapshot.
type Fetchers struct {
	Weather  func(context.Context) model.Weather
	Drive    func(context.Context) model.DriveLeg
	Subway   func(context.Context) model.SubwayLeg
	Outbound func(context.Context) model.TrainLeg
	Inbound  func(context.Context) model.TrainLeg
}

// Build runs all fetchers concurrently and returns the assembled snapshot.
func Build(ctx context.Context, f Fetchers, now func() time.Time) model.Snapshot {
	var snap model.Snapshot
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		if f.Weather != nil {
			snap.Weather = f.Weather(ctx)
		} else {
			snap.Weather = model.Weather{Err: "no weather fetcher"}
		}
	}()
	go func() {
		defer wg.Done()
		if f.Drive != nil {
			snap.Drive = f.Drive(ctx)
		} else {
			snap.Drive = model.DriveLeg{Err: "no drive fetcher"}
		}
	}()
	go func() {
		defer wg.Done()
		if f.Subway != nil {
			snap.Subway = f.Subway(ctx)
		} else {
			snap.Subway = model.SubwayLeg{Err: "no subway fetcher"}
		}
	}()
	go func() {
		defer wg.Done()
		if f.Outbound != nil {
			snap.Outbound = f.Outbound(ctx)
		} else {
			snap.Outbound = model.TrainLeg{Err: "no outbound fetcher"}
		}
	}()
	go func() {
		defer wg.Done()
		if f.Inbound != nil {
			snap.Inbound = f.Inbound(ctx)
		} else {
			snap.Inbound = model.TrainLeg{Err: "no inbound fetcher"}
		}
	}()

	wg.Wait()
	snap.GeneratedAt = now()
	return snap
}
