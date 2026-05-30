package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
home:
  lat: 40.0
  lon: -75.0
stops:
  home:
    id: "100"
    label: "Origin"
  work:
    id: "1"
    label: "Dest"
subway:
  routeId: "X"
feeds:
  mnrStaticGtfs: "http://example.com/mnr.zip"
  mnrRealtime: "http://example.com/mnr-rt"
  subwayAlerts: "http://example.com/subway-alerts"
weather:
  userAgent: "commute-helper (me@example.com)"
trainsToShow: 3
refreshSeconds: 45
dashboardUrl: "http://localhost:8080"
nudges:
  morning: "07:00"
  evening: "17:00"
timezone: "America/New_York"
server:
  addr: ":8080"
leaveBeforeTrainMinutes:
  outbound: 20
  inbound: 30
expectedTracks:
  home: "3"
`

func TestLoadReadsYAMLAndEnvSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(sample), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_MAPS_KEY", "gkey")
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord/webhook")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Stops.Home.ID != "100" || cfg.Stops.Work.ID != "1" {
		t.Fatalf("bad stop ids: %+v", cfg.Stops)
	}
	if cfg.Stops.Home.Label != "Origin" || cfg.Stops.Work.Label != "Dest" {
		t.Fatalf("bad stop labels: %+v", cfg.Stops)
	}
	if cfg.Subway.RouteID != "X" {
		t.Fatalf("bad route: %+v", cfg.Subway)
	}
	if cfg.GoogleMapsKey != "gkey" || cfg.DiscordWebhookURL != "https://discord/webhook" {
		t.Fatalf("env secrets not loaded: %+v", cfg)
	}
	if cfg.TrainsToShow != 3 {
		t.Fatalf("trainsToShow = %d", cfg.TrainsToShow)
	}
	if cfg.LeaveBeforeTrainMinutes.Outbound != 20 || cfg.LeaveBeforeTrainMinutes.Inbound != 30 {
		t.Fatalf("leaveBeforeTrainMinutes = %+v", cfg.LeaveBeforeTrainMinutes)
	}
	if got := cfg.ExpectedTracks["home"]; got != "3" {
		t.Fatalf("expectedTracks[home] = %q", got)
	}
}

func TestLoadDefaultsStopLabelsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
home:
  lat: 0
  lon: 0
stops:
  home:
    id: "100"
  work:
    id: "1"
subway:
  routeId: "X"
feeds:
  mnrStaticGtfs: "http://example.com/mnr.zip"
  mnrRealtime: "http://example.com/mnr-rt"
  subwayAlerts: "http://example.com/subway-alerts"
weather:
  userAgent: "commute-helper (me@example.com)"
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Stops.Home.Label != "Home" || cfg.Stops.Work.Label != "Work" {
		t.Fatalf("expected default labels Home/Work, got %+v", cfg.Stops)
	}
}

func TestLoadAllowsLeaveOffsetAndExpectedTracksOmitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
home:
  lat: 0
  lon: 0
stops:
  home:
    id: "100"
  work:
    id: "1"
subway:
  routeId: "X"
feeds:
  mnrStaticGtfs: "http://example.com/mnr.zip"
  mnrRealtime: "http://example.com/mnr-rt"
  subwayAlerts: "http://example.com/subway-alerts"
weather:
  userAgent: "commute-helper (me@example.com)"
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LeaveBeforeTrainMinutes.Outbound != 0 || cfg.LeaveBeforeTrainMinutes.Inbound != 0 {
		t.Fatalf("expected zero offsets, got %+v", cfg.LeaveBeforeTrainMinutes)
	}
	if len(cfg.ExpectedTracks) != 0 {
		t.Fatalf("expected empty ExpectedTracks, got %+v", cfg.ExpectedTracks)
	}
}

func TestLoadRejectsMissingRequired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("trainsToShow: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing required fields")
	}
}
