package drive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchParsesDurationInTraffic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "k" {
			t.Errorf("missing key param")
		}
		w.Write([]byte(`{"status":"OK","rows":[{"elements":[
			{"status":"OK","duration":{"value":480},"duration_in_traffic":{"value":620}}]}]}`))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Key: "k"}
	got := c.Fetch(context.Background(), 1.23, -4.56, 3.0, -4.0)
	if got.Err != "" {
		t.Fatalf("err: %s", got.Err)
	}
	if got.DurationMin != 10 { // 620s rounds to 10m
		t.Fatalf("durationMin = %d", got.DurationMin)
	}
}

func TestFetchReportsApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"REQUEST_DENIED","error_message":"bad key","rows":[]}`))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Key: "k"}
	got := c.Fetch(context.Background(), 1, 2, 3, 4)
	if got.Err == "" {
		t.Fatal("expected error")
	}
}
