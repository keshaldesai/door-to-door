// Package gtfs loads a static GTFS feed and answers next-departure queries.
package gtfs

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Schedule is an in-memory view of the parts of a GTFS feed we need.
type Schedule struct {
	// serviceDays[serviceID] = [7]bool indexed Sunday=0..Saturday=6.
	serviceDays map[string][7]bool
	// serviceRange[serviceID] = {start,end} as YYYYMMDD ints.
	serviceStart map[string]int
	serviceEnd   map[string]int
	// exceptions[serviceID][YYYYMMDD] = added(true)/removed(false).
	exceptions map[string]map[int]bool
	// tripService[tripID] = serviceID.
	tripService map[string]string
	// stopTimes[tripID] = ordered stops by stop_sequence.
	stopTimes map[string][]StopTime
}

// StopTime is one row of stop_times.txt with departure parsed to seconds past
// midnight (may exceed 86400 for trips after midnight).
type StopTime struct {
	StopID     string
	Sequence   int
	DepartSecs int
}

// Download fetches a GTFS zip over HTTP.
func Download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gtfs download: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Load parses a GTFS zip into a Schedule.
func Load(zipBytes []byte) (*Schedule, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	s := &Schedule{
		serviceDays:  map[string][7]bool{},
		serviceStart: map[string]int{},
		serviceEnd:   map[string]int{},
		exceptions:   map[string]map[int]bool{},
		tripService:  map[string]string{},
		stopTimes:    map[string][]StopTime{},
	}
	if err := s.loadCalendar(files["calendar.txt"]); err != nil {
		return nil, err
	}
	if f := files["calendar_dates.txt"]; f != nil {
		if err := s.loadCalendarDates(f); err != nil {
			return nil, err
		}
	}
	if err := s.loadTrips(files["trips.txt"]); err != nil {
		return nil, err
	}
	if err := s.loadStopTimes(files["stop_times.txt"]); err != nil {
		return nil, err
	}
	return s, nil
}

// readCSV reads a GTFS csv file and calls fn with a column->value map per row.
func readCSV(f *zip.File, fn func(map[string]string) error) error {
	if f == nil {
		return fmt.Errorf("missing required GTFS file")
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}

func (s *Schedule) loadCalendar(f *zip.File) error {
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	return readCSV(f, func(row map[string]string) error {
		id := row["service_id"]
		var d [7]bool
		for i, name := range days {
			d[i] = row[name] == "1"
		}
		s.serviceDays[id] = d
		s.serviceStart[id] = atoi(row["start_date"])
		s.serviceEnd[id] = atoi(row["end_date"])
		return nil
	})
}

func (s *Schedule) loadCalendarDates(f *zip.File) error {
	return readCSV(f, func(row map[string]string) error {
		id := row["service_id"]
		if s.exceptions[id] == nil {
			s.exceptions[id] = map[int]bool{}
		}
		s.exceptions[id][atoi(row["date"])] = row["exception_type"] == "1"
		return nil
	})
}

func (s *Schedule) loadTrips(f *zip.File) error {
	return readCSV(f, func(row map[string]string) error {
		s.tripService[row["trip_id"]] = row["service_id"]
		return nil
	})
}

func (s *Schedule) loadStopTimes(f *zip.File) error {
	return readCSV(f, func(row map[string]string) error {
		tripID := row["trip_id"]
		secs, err := parseGTFSTime(row["departure_time"])
		if err != nil {
			return nil // skip rows without a usable departure
		}
		s.stopTimes[tripID] = append(s.stopTimes[tripID], StopTime{
			StopID:     row["stop_id"],
			Sequence:   atoi(row["stop_sequence"]),
			DepartSecs: secs,
		})
		return nil
	})
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// parseGTFSTime parses "HH:MM:SS" (HH may be >= 24) to seconds past midnight.
func parseGTFSTime(v string) (int, error) {
	var h, m, sec int
	if _, err := fmt.Sscanf(v, "%d:%d:%d", &h, &m, &sec); err != nil {
		return 0, err
	}
	return h*3600 + m*60 + sec, nil
}
