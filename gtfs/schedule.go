package gtfs

import (
	"sort"
	"time"

	"github.com/keshaldee/commute/model"
)

// NextDepartures returns up to n scheduled departures from origin to dest whose
// origin departure is at or after now, sorted ascending by departure time.
func (s *Schedule) NextDepartures(origin, dest string, now time.Time, n int) []model.Train {
	yyyymmdd := now.Year()*10000 + int(now.Month())*100 + now.Day()
	weekday := int(now.Weekday()) // Sunday=0
	loc := now.Location()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var trains []model.Train
	for tripID, stops := range s.stopTimes {
		if !s.runsOn(s.tripService[tripID], yyyymmdd, weekday) {
			continue
		}
		oIdx, dIdx := -1, -1
		for i, st := range stops {
			if st.StopID == origin {
				oIdx = i
			}
			if st.StopID == dest {
				dIdx = i
			}
		}
		if oIdx == -1 || dIdx == -1 || oIdx >= dIdx {
			continue // does not serve both stops in the right order
		}
		dep := midnight.Add(time.Duration(stops[oIdx].DepartSecs) * time.Second)
		if dep.Before(now) {
			continue
		}
		trains = append(trains, model.Train{
			TripID:    tripID,
			Departure: dep,
			Status:    "On time",
		})
	}
	sort.Slice(trains, func(i, j int) bool {
		return trains[i].Departure.Before(trains[j].Departure)
	})
	if len(trains) > n {
		trains = trains[:n]
	}
	return trains
}

func (s *Schedule) runsOn(serviceID string, yyyymmdd, weekday int) bool {
	if ex, ok := s.exceptions[serviceID][yyyymmdd]; ok {
		return ex // explicit add/remove overrides the weekly pattern
	}
	days, ok := s.serviceDays[serviceID]
	if !ok {
		return false
	}
	if yyyymmdd < s.serviceStart[serviceID] || yyyymmdd > s.serviceEnd[serviceID] {
		return false
	}
	return days[weekday]
}
