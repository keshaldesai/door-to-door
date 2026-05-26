// Package notify posts commute summaries to a Discord webhook.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/keshaldee/commute/model"
)

// Post sends content to a Discord webhook URL.
func Post(ctx context.Context, client *http.Client, webhookURL, content string) error {
	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook: status %d", resp.StatusCode)
	}
	return nil
}

// Summary builds a short message. which is "morning" (outbound) or "evening"
// (inbound).
func Summary(snap model.Snapshot, which, dashboardURL string) string {
	var b strings.Builder
	leg := snap.Outbound
	label := "Outbound (home -> Work)"
	if which == "evening" {
		leg = snap.Inbound
		label = "Inbound (Work -> home)"
	}
	fmt.Fprintf(&b, "**Commute %s**\n", label)

	if len(leg.Trains) > 0 {
		t := leg.Trains[0]
		line := fmt.Sprintf("Next train %s - %s", t.Departure.Format("3:04"), t.Status)
		if t.Track != "" {
			line += fmt.Sprintf(" (track %s)", t.Track)
		}
		b.WriteString(line + "\n")
	} else if leg.Err != "" {
		fmt.Fprintf(&b, "Train data unavailable: %s\n", leg.Err)
	}

	fmt.Fprintf(&b, "subway: %s\n", snap.Subway.Status)

	if snap.Drive.Err == "" {
		fmt.Fprintf(&b, "Drive: %d min\n", snap.Drive.DurationMin)
	}

	if snap.Weather.Summary != "" {
		fmt.Fprintf(&b, "Weather: %s (%d%% precip)\n", snap.Weather.Summary, snap.Weather.PrecipChance)
	}
	for _, a := range snap.Weather.Alerts {
		fmt.Fprintf(&b, "WX ALERT: %s\n", a.Event)
	}

	b.WriteString(dashboardURL)
	return b.String()
}
