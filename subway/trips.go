package subway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/keshaldesai/door-to-door/model"
	"google.golang.org/protobuf/proto"
)

// TripsClient fetches the GTFS-realtime trip-updates feed for one MTA
// subway route group and returns upcoming arrivals at a single stop.
// It is a sibling to the alerts-only Client in subway.go.
type TripsClient struct {
	HTTP    *http.Client
	URL     string
	RouteID string
}

// Fetch returns the next up to n arrivals at stopID for trains on
// RouteID in the given directionID. Arrivals older than now-30s are
// dropped. The 30s grace covers the gap between the rider seeing the
// dashboard and the train actually pulling in. Results are sorted
// ascending by arrival time.
//
// directionID matching is strict: the trip descriptor's direction_id
// must equal directionID. For NYCT, stop_id encodes the platform side
// (trailing N/S), so direction filtering is also effectively enforced
// by the stop_id match; the trip-descriptor filter is the secondary
// guard.
func (c *TripsClient) Fetch(ctx context.Context, stopID string, directionID uint32, n int, now time.Time) model.SubwayCountdown {
	out := model.SubwayCountdown{
		StopID:    stopID,
		Direction: directionID,
		UpdatedAt: now,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out.Err = fmt.Sprintf("subway realtime: status %d", resp.StatusCode)
		return out
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	var feed gtfs.FeedMessage
	if err := proto.Unmarshal(body, &feed); err != nil {
		out.Err = err.Error()
		return out
	}

	cutoff := now.Add(-30 * time.Second)
	for _, e := range feed.GetEntity() {
		tu := e.GetTripUpdate()
		if tu == nil {
			continue
		}
		trip := tu.GetTrip()
		if trip.GetRouteId() != c.RouteID {
			continue
		}
		if trip.GetDirectionId() != directionID {
			continue
		}
		for _, st := range tu.GetStopTimeUpdate() {
			if st.GetStopId() != stopID {
				continue
			}
			ts := stopTime(st)
			if ts == 0 {
				continue
			}
			at := time.Unix(ts, 0).UTC()
			if at.Before(cutoff) {
				continue
			}
			out.Arrivals = append(out.Arrivals, at)
		}
	}

	sort.Slice(out.Arrivals, func(i, j int) bool {
		return out.Arrivals[i].Before(out.Arrivals[j])
	})
	if len(out.Arrivals) > n {
		out.Arrivals = out.Arrivals[:n]
	}
	return out
}

// stopTime returns Unix seconds for the arrival event, falling back to
// departure when arrival is unset. Zero means no usable timestamp.
func stopTime(st *gtfs.TripUpdate_StopTimeUpdate) int64 {
	if a := st.GetArrival(); a != nil && a.GetTime() != 0 {
		return a.GetTime()
	}
	if d := st.GetDeparture(); d != nil && d.GetTime() != 0 {
		return d.GetTime()
	}
	return 0
}
