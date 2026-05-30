// Package notify posts commute summaries to a Discord webhook.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	heading := "Outbound"
	if which == "evening" {
		leg = snap.Inbound
		heading = "Inbound"
	}
	// Build a "<heading> (<origin> -> <dest>)" header from the leg's own
	// labels so no station name is hardcoded here.
	if leg.Origin != "" && leg.Dest != "" {
		fmt.Fprintf(&b, "**Commute %s (%s -> %s)**\n", heading, leg.Origin, leg.Dest)
	} else {
		fmt.Fprintf(&b, "**Commute %s**\n", heading)
	}

	if len(leg.Trains) > 0 {
		t := leg.Trains[0]
		line := fmt.Sprintf("Next train %s - %s", t.Departure.Format("3:04 PM"), t.Status)
		// Collect the parenthetical extras: leave time, then track. Each is
		// omitted when not applicable so the nudge stays short.
		var extras []string
		if leg.LeaveOffsetMin > 0 {
			leaveAt := t.Departure.Add(-time.Duration(leg.LeaveOffsetMin) * time.Minute)
			extras = append(extras, "leave "+leaveAt.Format("3:04 PM"))
		}
		if t.Track != "" {
			if leg.ExpectedTrack != "" && t.Track != leg.ExpectedTrack {
				extras = append(extras, fmt.Sprintf("track %s - expected %s", t.Track, leg.ExpectedTrack))
			} else {
				extras = append(extras, "track "+t.Track)
			}
		}
		if len(extras) > 0 {
			line += " (" + strings.Join(extras, ", ") + ")"
		}
		fmt.Fprintln(&b, line)
	} else if leg.Err != "" {
		fmt.Fprintf(&b, "Train data unavailable: %s\n", leg.Err)
	}

	subwayLabel := "Subway"
	if snap.Subway.Line != "" {
		subwayLabel = snap.Subway.Line + " train"
	}
	fmt.Fprintf(&b, "%s: %s\n", subwayLabel, snap.Subway.Status)

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
