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
				Effect: &e,
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

func TestFetchReportsAlertForRoute7(t *testing.T) {
	srv := serve(t, feedBytes(t, "7", "subways are delayed", gtfs.Alert_SIGNIFICANT_DELAYS))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "7"}
	got := c.Fetch(context.Background())
	if got.Status != "Delays" {
		t.Fatalf("status = %q", got.Status)
	}
	if len(got.Alerts) != 1 || got.Alerts[0] != "subways are delayed" {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}

func TestFetchNoEffectAlertKeepsGoodService(t *testing.T) {
	srv := serve(t, feedBytes(t, "7", "Planned work this weekend", gtfs.Alert_NO_EFFECT))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "7"}
	got := c.Fetch(context.Background())
	if got.Status != "Good Service" {
		t.Fatalf("status = %q, want Good Service", got.Status)
	}
	if len(got.Alerts) != 1 || got.Alerts[0] != "Planned work this weekend" {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}

func TestFetchIgnoresOtherRoutes(t *testing.T) {
	srv := serve(t, feedBytes(t, "L", "L train delays", gtfs.Alert_SIGNIFICANT_DELAYS))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), URL: srv.URL, RouteID: "7"}
	got := c.Fetch(context.Background())
	if got.Status != "Good Service" {
		t.Fatalf("status = %q, want Good Service", got.Status)
	}
	if len(got.Alerts) != 0 {
		t.Fatalf("alerts = %v", got.Alerts)
	}
}
