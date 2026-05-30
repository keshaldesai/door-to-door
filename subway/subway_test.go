package subway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func feedBytes(t *testing.T, routeID, header string, effect gtfs.Alert_Effect) []byte {
	t.Helper()
	r := routeID
	e := effect
	h := header
	msg := &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*gtfs.FeedEntity{{
			Id: proto.String("a1"),
			Alert: &gtfs.Alert{
				Effect:         &e,
				InformedEntity: []*gtfs.EntitySelector{{RouteId: &r}},
				HeaderText: &gtfs.TranslatedString{
					Translation: []*gtfs.TranslatedString_Translation{
						{Text: &h, Language: proto.String("en")},
					},
				},
			},
		}},
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func serve(t *testing.T, body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
}

func TestFetchReportsAlertForRoute(t *testing.T) {
	srv := serve(t, feedBytes(t, "X", "trains are delayed", gtfs.Alert_SIGNIFICANT_DELAYS))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X"}
	got := c.Fetch(context.Background())
	if got.Status != "Delays" {
		t.Fatalf("status = %q", got.Status)
	}
	if len(got.Alerts) != 1 || got.Alerts[0] != "trains are delayed" {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}

func TestFetchNoEffectAlertKeepsGoodService(t *testing.T) {
	srv := serve(t, feedBytes(t, "X", "Planned work this weekend", gtfs.Alert_NO_EFFECT))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X"}
	got := c.Fetch(context.Background())
	if got.Status != "Good Service" {
		t.Fatalf("status = %q, want Good Service", got.Status)
	}
	if len(got.Alerts) != 1 || got.Alerts[0] != "Planned work this weekend" {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}

func TestFetchIgnoresOtherRoutes(t *testing.T) {
	srv := serve(t, feedBytes(t, "Y", "Y train delays", gtfs.Alert_SIGNIFICANT_DELAYS))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X"}
	got := c.Fetch(context.Background())
	if got.Status != "Good Service" {
		t.Fatalf("status = %q, want Good Service", got.Status)
	}
	if len(got.Alerts) != 0 {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}

// alertEntity builds an alert FeedEntity for routeID with optional stop_ids.
func alertEntity(id, header string, effect gtfs.Alert_Effect, routeID string, stopIDs []string) *gtfs.FeedEntity {
	r := routeID
	sel := []*gtfs.EntitySelector{{RouteId: &r}}
	for _, s := range stopIDs {
		sid := s
		sel = append(sel, &gtfs.EntitySelector{StopId: &sid})
	}
	e := effect
	h := header
	return &gtfs.FeedEntity{
		Id: proto.String(id),
		Alert: &gtfs.Alert{
			Effect:         &e,
			InformedEntity: sel,
			HeaderText: &gtfs.TranslatedString{
				Translation: []*gtfs.TranslatedString_Translation{{Text: &h, Language: proto.String("en")}},
			},
		},
	}
}

func serveEntities(t *testing.T, ents ...*gtfs.FeedEntity) *httptest.Server {
	t.Helper()
	msg := &gtfs.FeedMessage{Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")}, Entity: ents}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return serve(t, b)
}

func TestFetchFiltersToRiderStops(t *testing.T) {
	srv := serveEntities(t,
		alertEntity("a1", "Far-end work", gtfs.Alert_NO_EFFECT, "X", []string{"S1", "S2"}),
		alertEntity("a2", "Service change at the destination", gtfs.Alert_NO_EFFECT, "X", []string{"S5", "S6"}),
	)
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X", StopIDs: []string{"S3", "S4", "S5", "S6"}}
	got := c.Fetch(context.Background())
	if len(got.Alerts) != 1 || got.Alerts[0] != "Service change at the destination" {
		t.Fatalf("alerts = %v, want only the touching-stop alert", got.Alerts)
	}
}

func TestFetchKeepsLineWideAlertWhenFiltering(t *testing.T) {
	srv := serveEntities(t,
		alertEntity("a1", "Trains delayed systemwide", gtfs.Alert_SIGNIFICANT_DELAYS, "X", nil),
	)
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X", StopIDs: []string{"S6"}}
	got := c.Fetch(context.Background())
	if got.Status != "Delays" || len(got.Alerts) != 1 {
		t.Fatalf("line-wide alert dropped: status=%q alerts=%v", got.Status, got.Alerts)
	}
}

func TestFetchDedupesRepeatedAlertText(t *testing.T) {
	srv := serveEntities(t,
		alertEntity("a1", "Same notice", gtfs.Alert_NO_EFFECT, "X", []string{"S6"}),
		alertEntity("a2", "Same notice", gtfs.Alert_NO_EFFECT, "X", []string{"S5"}),
	)
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "X", StopIDs: []string{"S3", "S4", "S5", "S6"}}
	got := c.Fetch(context.Background())
	if len(got.Alerts) != 1 {
		t.Fatalf("expected dedupe to 1 alert, got %v", got.Alerts)
	}
}
