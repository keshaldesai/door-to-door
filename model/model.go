// Package model holds the shared data types assembled by the orchestrator and
// serialized to the dashboard. Journeys are a frontend composition over these
// shared legs: the subway status and the drive estimate are each computed
// once and reused in both directions.
package model

import "time"

// Train is one upcoming Metro-North departure.
type Train struct {
	TripID    string    `json:"-"`
	Departure time.Time `json:"departure"`
	Status    string    `json:"status"` // "On time", "Delayed 6m", "Early 2m"
	DelayMin  int       `json:"delayMin"`
	Track     string    `json:"track"`    // "" when not yet posted
	Realtime  bool      `json:"realtime"` // true when realtime-adjusted
}

// TrainLeg is the next few trains between two stops in one direction.
type TrainLeg struct {
	Origin string  `json:"origin"`
	Dest   string  `json:"dest"`
	Trains []Train `json:"trains"`
	Source string  `json:"source"` // "realtime" | "scheduled"
	// LeaveOffsetMin is how many minutes before the train you should leave for
	// it. Zero means no leave-by hint is shown.
	LeaveOffsetMin int `json:"leaveOffsetMin,omitempty"`
	// ExpectedTrack is the platform the user expects to board on at this leg's
	// origin. Empty means no comparison; non-empty plus a posted track that
	// differs is a warning.
	ExpectedTrack string    `json:"expectedTrack,omitempty"`
	Err           string    `json:"err,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SubwayLeg is the subway-line service status plus any active alert text.
type SubwayLeg struct {
	Line      string    `json:"line"`
	Status    string    `json:"status"` // "Good Service" | "Delays" | "Service Change"
	Alerts    []string  `json:"alerts"`
	Err       string    `json:"err,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DriveLeg is the traffic-aware drive time between home and the station.
type DriveLeg struct {
	DurationMin int       `json:"durationMin"`
	Err         string    `json:"err,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WeatherAlert is one active NWS alert.
type WeatherAlert struct {
	Event    string `json:"event"`
	Headline string `json:"headline"`
}

// Weather is the current short forecast plus active alerts.
type Weather struct {
	Summary      string         `json:"summary"`
	TempF        int            `json:"tempF"`
	PrecipChance int            `json:"precipChance"`
	Alerts       []WeatherAlert `json:"alerts"`
	SunriseAt    time.Time      `json:"sunriseAt"`
	SunsetAt     time.Time      `json:"sunsetAt"`
	Err          string         `json:"err,omitempty"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// Snapshot is the full dashboard state at a moment in time.
type Snapshot struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Weather     Weather   `json:"weather"`
	Drive       DriveLeg  `json:"drive"`
	Subway      SubwayLeg `json:"subway"`
	Outbound    TrainLeg  `json:"outbound"` // home -> work
	Inbound     TrainLeg  `json:"inbound"`  // work -> home
}
