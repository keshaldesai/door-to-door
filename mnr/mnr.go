// Package mnr fetches the Metro-North GTFS-realtime feed and overlays live
// delay and status onto scheduled trains.
package mnr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/keshaldee/commute/model"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	HTTP *http.Client
	URL  string
}

// Fetch downloads and parses the MNR realtime feed.
func (c *Client) Fetch(ctx context.Context) (*gtfs.FeedMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mnr realtime: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed gtfs.FeedMessage
	if err := proto.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

// Overlay applies delay and status from feed to trains, matching by trip_id and
// using the stop-time update for originStopID. Unmatched trains are unchanged.
func Overlay(trains []model.Train, feed *gtfs.FeedMessage, originStopID string) []model.Train {
	byTrip := map[string]*gtfs.TripUpdate{}
	for _, e := range feed.GetEntity() {
		if tu := e.GetTripUpdate(); tu != nil {
			byTrip[tu.GetTrip().GetTripId()] = tu
		}
	}
	for i := range trains {
		tu, ok := byTrip[trains[i].TripID]
		if !ok {
			continue
		}
		delay, found := originDelay(tu, originStopID)
		if !found {
			continue
		}
		mins := int(delay) / 60
		trains[i].Realtime = true
		trains[i].DelayMin = mins
		trains[i].Departure = trains[i].Departure.Add(time.Duration(delay) * time.Second)
		trains[i].Status = statusFor(mins)
	}
	return trains
}

func originDelay(tu *gtfs.TripUpdate, stopID string) (int32, bool) {
	for _, stu := range tu.GetStopTimeUpdate() {
		if stu.GetStopId() != stopID {
			continue
		}
		if stu.Departure != nil {
			return stu.Departure.GetDelay(), true
		}
		if stu.Arrival != nil {
			return stu.Arrival.GetDelay(), true
		}
	}
	return 0, false
}

func statusFor(mins int) string {
	switch {
	case mins >= 2:
		return fmt.Sprintf("Delayed %dm", mins)
	case mins <= -2:
		return fmt.Sprintf("Early %dm", -mins)
	default:
		return "On time"
	}
}
