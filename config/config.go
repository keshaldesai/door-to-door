// Package config loads commute-helper settings from a YAML file plus secrets
// from the environment.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Home struct {
		Lat float64 `yaml:"lat"`
		Lon float64 `yaml:"lon"`
	} `yaml:"home"`
	Station struct {
		Lat float64 `yaml:"lat"`
		Lon float64 `yaml:"lon"`
	} `yaml:"station"`
	Stops struct {
		Home string `yaml:"home"`
		Work       string `yaml:"work"`
	} `yaml:"stops"`
	Subway struct {
		RouteID string   `yaml:"routeId"`
		StopIDs []string `yaml:"stopIds"`
	} `yaml:"subway"`
	Feeds struct {
		MNRStaticGTFS string `yaml:"mnrStaticGtfs"`
		MNRRealtime   string `yaml:"mnrRealtime"`
		SubwayAlerts  string `yaml:"subwayAlerts"`
	} `yaml:"feeds"`
	Weather struct {
		UserAgent string `yaml:"userAgent"`
	} `yaml:"weather"`
	TrainsToShow   int    `yaml:"trainsToShow"`
	RefreshSeconds int    `yaml:"refreshSeconds"`
	DashboardURL   string `yaml:"dashboardUrl"`
	Nudges         struct {
		Morning string `yaml:"morning"`
		Evening string `yaml:"evening"`
	} `yaml:"nudges"`
	Timezone string `yaml:"timezone"`
	Server   struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	// LeaveBeforeTrainMinutes is how many minutes before each MNR departure to
	// leave for the train, per direction. Zero hides the leave-by hint.
	LeaveBeforeTrainMinutes struct {
		Outbound int `yaml:"outbound"`
		Inbound  int `yaml:"inbound"`
	} `yaml:"leaveBeforeTrainMinutes"`
	// ExpectedTracks maps a `stops` key (e.g. the home-stop key) to the track
	// the user expects to board on. Empty means no comparison.
	ExpectedTracks map[string]string `yaml:"expectedTracks"`

	// Secrets, loaded from the environment, never from the YAML file.
	GoogleMapsKey     string `yaml:"-"`
	DiscordWebhookURL string `yaml:"-"`
}

// Load reads the YAML config at path, overlays secrets from the environment,
// applies defaults, and validates required fields.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.GoogleMapsKey = os.Getenv("GOOGLE_MAPS_KEY")
	cfg.DiscordWebhookURL = os.Getenv("DISCORD_WEBHOOK_URL")

	if cfg.TrainsToShow == 0 {
		cfg.TrainsToShow = 3
	}
	if cfg.RefreshSeconds == 0 {
		cfg.RefreshSeconds = 45
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "America/New_York"
	}
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}

	var missing []string
	if cfg.Stops.Home == "" {
		missing = append(missing, "stops.home")
	}
	if cfg.Stops.Work == "" {
		missing = append(missing, "stops.work")
	}
	if cfg.Feeds.MNRStaticGTFS == "" {
		missing = append(missing, "feeds.mnrStaticGtfs")
	}
	if cfg.Feeds.MNRRealtime == "" {
		missing = append(missing, "feeds.mnrRealtime")
	}
	if cfg.Feeds.SubwayAlerts == "" {
		missing = append(missing, "feeds.subwayAlerts")
	}
	if cfg.Weather.UserAgent == "" {
		missing = append(missing, "weather.userAgent")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required config fields: %v", missing)
	}
	return &cfg, nil
}
