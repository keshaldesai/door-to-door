package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchParsesForecastAndAlerts(t *testing.T) {
	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/points/1.2300,-4.5600", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"properties":{"forecastHourly":%q}}`, base+"/hourly")
	})
	mux.HandleFunc("/hourly", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"properties":{"periods":[
			{"temperature":55,"shortForecast":"Light Rain","probabilityOfPrecipitation":{"value":80}}]}}`)
	})
	mux.HandleFunc("/alerts/active", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[{"properties":{"event":"Flood Watch","headline":"Flooding possible"}}]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base = srv.URL

	c := &Client{HTTP: srv.Client(), Base: srv.URL, UserAgent: "test"}
	got := c.Fetch(context.Background(), 1.23, -4.56)
	if got.Err != "" {
		t.Fatalf("unexpected err: %s", got.Err)
	}
	if got.TempF != 55 || got.PrecipChance != 80 || got.Summary != "Light Rain" {
		t.Fatalf("bad forecast: %+v", got)
	}
	if len(got.Alerts) != 1 || got.Alerts[0].Event != "Flood Watch" {
		t.Fatalf("bad alerts: %+v", got.Alerts)
	}
}

func TestFetchReportsErrorOnBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, UserAgent: "test"}
	got := c.Fetch(context.Background(), 1.23, -4.56)
	if got.Err == "" {
		t.Fatal("expected error to be reported")
	}
}

func TestFetchPopulatesSunEvenOnUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, UserAgent: "test"}
	got := c.Fetch(context.Background(), 40.7128, -74.0060)
	if got.Err == "" {
		t.Fatal("expected upstream error to be reported")
	}
	if got.SunriseAt.IsZero() || got.SunsetAt.IsZero() {
		t.Fatalf("expected sun times populated despite error: rise=%v set=%v", got.SunriseAt, got.SunsetAt)
	}
}

func TestFetchAnchorsSunToConfiguredLocation(t *testing.T) {
	// With Loc=America/Anchorage, sunrise/sunset should be on the Anchorage
	// civil date, even when the system clock is in another zone. Smoke test:
	// for NYC coords on the same wall clock, the Anchorage-anchored civil
	// day picks a sun-day at most 1 calendar day apart from the local-anchored
	// one, and the returned times are never zero (algorithm always solves).
	anchorage, err := time.LoadLocation("America/Anchorage")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, UserAgent: "test", Loc: anchorage}
	got := c.Fetch(context.Background(), 40.7128, -74.0060)
	if got.SunriseAt.IsZero() || got.SunsetAt.IsZero() {
		t.Fatalf("sun times not populated: %+v", got)
	}
	// The civil date used should be the Anchorage date of now.
	wantDay := time.Now().In(anchorage).Day()
	gotDay := got.SunriseAt.In(anchorage).Day()
	if wantDay != gotDay {
		t.Fatalf("sunrise civil day = %d, want %d (Anchorage of now)", gotDay, wantDay)
	}
}
