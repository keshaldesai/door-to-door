package gtfs

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sampleFeed(t *testing.T) []byte {
	return buildZip(t, map[string]string{
		"stops.txt": "stop_id,stop_name\nA,Origin\nB,Dest\n",
		"calendar.txt": "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
			"WK,1,1,1,1,1,0,0,20260101,20261231\n",
		"calendar_dates.txt": "service_id,date,exception_type\n",
		"trips.txt":          "route_id,service_id,trip_id\nNH,WK,T1\nNH,WK,T2\n",
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
			"T1,07:10:00,07:10:00,A,1\nT1,08:20:00,08:20:00,B,2\n" +
			"T2,07:40:00,07:40:00,A,1\nT2,08:50:00,08:50:00,B,2\n",
	})
}

func TestNextDeparturesReturnsUpcomingInOrder(t *testing.T) {
	sched, err := Load(sampleFeed(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	// Monday 2026-05-25 07:00 local: both T1 (07:10) and T2 (07:40) are upcoming.
	now := time.Date(2026, 5, 25, 7, 0, 0, 0, loc)
	trains := sched.NextDepartures("A", "B", now, 5)
	if len(trains) != 2 {
		t.Fatalf("got %d trains, want 2", len(trains))
	}
	if !trains[0].Departure.Before(trains[1].Departure) {
		t.Fatalf("not sorted: %v", trains)
	}
	if trains[0].TripID != "T1" {
		t.Fatalf("first trip = %s, want T1", trains[0].TripID)
	}
}

func TestNextDeparturesSkipsPastAndWrongDirection(t *testing.T) {
	sched, err := Load(sampleFeed(t))
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 7, 30, 0, 0, loc) // T1 already departed
	trains := sched.NextDepartures("A", "B", now, 5)
	if len(trains) != 1 || trains[0].TripID != "T2" {
		t.Fatalf("got %v, want only T2", trains)
	}
	// Reverse direction has no trips serving B before A.
	if rev := sched.NextDepartures("B", "A", now, 5); len(rev) != 0 {
		t.Fatalf("reverse should be empty, got %v", rev)
	}
}

func TestNextDeparturesLimitsToN(t *testing.T) {
	sched, _ := Load(sampleFeed(t))
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 6, 0, 0, 0, loc)
	if trains := sched.NextDepartures("A", "B", now, 1); len(trains) != 1 {
		t.Fatalf("limit not applied: %d", len(trains))
	}
}

// TestOutOfOrderStopTimes verifies that stop_times.txt rows listed out of
// stop_sequence order are still treated in sequence order.
// The feed lists B (stop_sequence 2) before A (stop_sequence 1) in the CSV.
func TestOutOfOrderStopTimes(t *testing.T) {
	feed := buildZip(t, map[string]string{
		"stops.txt": "stop_id,stop_name\nA,Origin\nB,Dest\n",
		"calendar.txt": "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
			"WK,1,1,1,1,1,0,0,20260101,20261231\n",
		"trips.txt": "route_id,service_id,trip_id\nNH,WK,T1\n",
		// B row (seq 2) is listed BEFORE A row (seq 1) -- out of order
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
			"T1,08:20:00,08:20:00,B,2\n" +
			"T1,07:10:00,07:10:00,A,1\n",
	})
	sched, err := Load(feed)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 6, 0, 0, 0, loc)
	trains := sched.NextDepartures("A", "B", now, 5)
	if len(trains) != 1 {
		t.Fatalf("got %d trains, want 1", len(trains))
	}
	if trains[0].TripID != "T1" {
		t.Fatalf("TripID = %s, want T1", trains[0].TripID)
	}
	want := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	if !trains[0].Departure.Equal(want) {
		t.Fatalf("Departure = %v, want %v", trains[0].Departure, want)
	}
}

// TestLoopRouteDuplicateOrigin verifies that a trip serving the same stop twice
// (loop route) uses the FIRST occurrence of origin and the first dest after it.
// Trip T1: A (seq1, 07:10) -> B (seq2, 08:20) -> A (seq3, 09:30)
func TestLoopRouteDuplicateOrigin(t *testing.T) {
	feed := buildZip(t, map[string]string{
		"stops.txt": "stop_id,stop_name\nA,Origin\nB,Dest\n",
		"calendar.txt": "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
			"WK,1,1,1,1,1,0,0,20260101,20261231\n",
		"trips.txt": "route_id,service_id,trip_id\nNH,WK,T1\n",
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
			"T1,07:10:00,07:10:00,A,1\n" +
			"T1,08:20:00,08:20:00,B,2\n" +
			"T1,09:30:00,09:30:00,A,3\n",
	})
	sched, err := Load(feed)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 6, 0, 0, 0, loc)
	trains := sched.NextDepartures("A", "B", now, 5)
	if len(trains) != 1 {
		t.Fatalf("got %d trains, want 1", len(trains))
	}
	if trains[0].TripID != "T1" {
		t.Fatalf("TripID = %s, want T1", trains[0].TripID)
	}
	want := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	if !trains[0].Departure.Equal(want) {
		t.Fatalf("Departure = %v, want %v (first A departure)", trains[0].Departure, want)
	}
}

// TestCalendarDatesOnlyFeed verifies a feed with no calendar.txt (like the real
// Metro-North feed) where service is defined entirely by calendar_dates.txt:
// each running service-date is an explicit exception_type 1 entry.
func TestCalendarDatesOnlyFeed(t *testing.T) {
	feed := buildZip(t, map[string]string{
		"stops.txt": "stop_id,stop_name\nA,Origin\nB,Dest\n",
		// No calendar.txt. Service S1 runs only on 2026-05-25 (type 1 = added).
		"calendar_dates.txt": "service_id,date,exception_type\nS1,20260525,1\n",
		"trips.txt":          "route_id,service_id,trip_id\nNH,S1,T1\n",
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
			"T1,07:10:00,07:10:00,A,1\nT1,08:20:00,08:20:00,B,2\n",
	})
	sched, err := Load(feed)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")

	// On the listed date the train runs.
	now := time.Date(2026, 5, 25, 6, 0, 0, 0, loc)
	if trains := sched.NextDepartures("A", "B", now, 5); len(trains) != 1 || trains[0].TripID != "T1" {
		t.Fatalf("on listed date got %v, want one T1", trains)
	}

	// On an unlisted date the service does not run.
	other := time.Date(2026, 5, 26, 6, 0, 0, 0, loc)
	if trains := sched.NextDepartures("A", "B", other, 5); len(trains) != 0 {
		t.Fatalf("on unlisted date got %v, want none", trains)
	}
}
