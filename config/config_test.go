package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
home:
  lat: 0.0
  lon: -0.0
stops:
  home: "100"
  work: "1"
subway:
  routeId: "7"
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
	if cfg.Stops.Work != "1" || cfg.Subway.RouteID != "7" {
		t.Fatalf("bad stops/route: %+v", cfg)
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

func TestLoadAllowsLeaveOffsetAndExpectedTracksOmitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Same as sample but without the two new optional sections.
	yaml := `
home:
  lat: 0.0
  lon: -0.0
stops:
  home: "100"
  work: "1"
subway:
  routeId: "7"
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
