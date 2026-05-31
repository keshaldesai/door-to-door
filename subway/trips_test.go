package subway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// tripUpdate constructs a TripUpdate entity for the given route/direction with
// arrivals (Unix seconds) at the named stop IDs (one stop_time_update each).
func tripUpdate(id, routeID string, directionID uint32, stops []string, times []int64) *gtfs.FeedEntity {
	rid := routeID
	dir := directionID
	updates := make([]*gtfs.TripUpdate_StopTimeUpdate, 0, len(stops))
	for i, s := range stops {
		sid := s
		ts := times[i]
		updates = append(updates, &gtfs.TripUpdate_StopTimeUpdate{
			StopId:  &sid,
			Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: &ts},
		})
	}
	return &gtfs.FeedEntity{
		Id: proto.String(id),
		TripUpdate: &gtfs.TripUpdate{
			Trip:           &gtfs.TripDescriptor{RouteId: &rid, DirectionId: &dir},
			StopTimeUpdate: updates,
		},
	}
}

func serveTrips(t *testing.T, ents ...*gtfs.FeedEntity) *httptest.Server {
	t.Helper()
	msg := &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: ents,
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(b)
	}))
}

func TestTripsClientReturnsArrivalsSortedAtStop(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	srv := serveTrips(t,
		tripUpdate("t1", "6", 1, []string{"635N"}, []int64{now.Add(12 * time.Minute).Unix()}),
		tripUpdate("t2", "6", 1, []string{"635N"}, []int64{now.Add(4 * time.Minute).Unix()}),
		tripUpdate("t3", "6", 1, []string{"635N"}, []int64{now.Add(21 * time.Minute).Unix()}),
	)
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 3, now)
	if got.Err != "" {
		t.Fatalf("err = %q", got.Err)
	}
	if len(got.Arrivals) != 3 {
		t.Fatalf("len=%d, want 3", len(got.Arrivals))
	}
	if !got.Arrivals[0].Equal(now.Add(4 * time.Minute)) ||
		!got.Arrivals[1].Equal(now.Add(12 * time.Minute)) ||
		!got.Arrivals[2].Equal(now.Add(21 * time.Minute)) {
		t.Fatalf("not sorted: %v", got.Arrivals)
	}
	if got.StopID != "635N" || got.Direction != 1 {
		t.Fatalf("metadata = %+v", got)
	}
}

func TestTripsClientFiltersByRouteAndDirection(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	srv := serveTrips(t,
		tripUpdate("t-wrong-route", "4", 1, []string{"635N"}, []int64{now.Add(5 * time.Minute).Unix()}),
		tripUpdate("t-wrong-dir", "6", 0, []string{"635N"}, []int64{now.Add(5 * time.Minute).Unix()}),
		tripUpdate("t-right", "6", 1, []string{"635N"}, []int64{now.Add(7 * time.Minute).Unix()}),
	)
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 5, now)
	if len(got.Arrivals) != 1 || !got.Arrivals[0].Equal(now.Add(7*time.Minute)) {
		t.Fatalf("expected just the route+direction match, got %v", got.Arrivals)
	}
}

func TestTripsClientIgnoresOtherStops(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	srv := serveTrips(t,
		tripUpdate("t", "6", 1, []string{"631N", "633N", "635N"},
			[]int64{now.Add(2 * time.Minute).Unix(), now.Add(4 * time.Minute).Unix(), now.Add(6 * time.Minute).Unix()}),
	)
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 5, now)
	if len(got.Arrivals) != 1 || !got.Arrivals[0].Equal(now.Add(6*time.Minute)) {
		t.Fatalf("expected only the matching stop, got %v", got.Arrivals)
	}
}

func TestTripsClientDropsStaleArrivals(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	srv := serveTrips(t,
		tripUpdate("t-old", "6", 1, []string{"635N"}, []int64{now.Add(-2 * time.Minute).Unix()}),
		tripUpdate("t-grace", "6", 1, []string{"635N"}, []int64{now.Add(-10 * time.Second).Unix()}),
		tripUpdate("t-new", "6", 1, []string{"635N"}, []int64{now.Add(3 * time.Minute).Unix()}),
	)
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 5, now)
	if len(got.Arrivals) != 2 {
		t.Fatalf("expected 2 arrivals (grace + future), got %v", got.Arrivals)
	}
}

func TestTripsClientHonorsLimit(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	srv := serveTrips(t,
		tripUpdate("a", "6", 1, []string{"635N"}, []int64{now.Add(2 * time.Minute).Unix()}),
		tripUpdate("b", "6", 1, []string{"635N"}, []int64{now.Add(5 * time.Minute).Unix()}),
		tripUpdate("c", "6", 1, []string{"635N"}, []int64{now.Add(9 * time.Minute).Unix()}),
		tripUpdate("d", "6", 1, []string{"635N"}, []int64{now.Add(13 * time.Minute).Unix()}),
	)
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 2, now)
	if len(got.Arrivals) != 2 {
		t.Fatalf("expected limit=2, got %v", got.Arrivals)
	}
}

func TestTripsClientReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &TripsClient{HTTP: srv.Client(), URL: srv.URL, RouteID: "6"}
	got := c.Fetch(context.Background(), "635N", 1, 3, time.Now())
	if got.Err == "" {
		t.Fatal("expected err on 500")
	}
}
