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
