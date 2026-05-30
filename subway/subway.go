// Package subway derives the configured subway route's service status and
// active alert text from the MTA subway service-alerts GTFS-realtime feed
// (keyless).
package subway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/keshaldee/commute/model"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	HTTP    *http.Client
	URL     string
	RouteID string
	// StopIDs limits alerts to those touching the rider's own stops. When set,
	// stop-specific alerts must reference one of these stops; line-wide alerts
	// are always kept. When empty, every alert for the route is kept.
	StopIDs []string
}

func (c *Client) Fetch(ctx context.Context) model.SubwayLeg {
	out := model.SubwayLeg{Line: c.RouteID, Status: "Good Service", UpdatedAt: time.Now()}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out.Err = fmt.Sprintf("subway feed: status %d", resp.StatusCode)
		return out
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	var feed gtfs.FeedMessage
	if err := proto.Unmarshal(body, &feed); err != nil {
		out.Err = err.Error()
		return out
	}

	stopSet := make(map[string]bool, len(c.StopIDs))
	for _, s := range c.StopIDs {
		stopSet[s] = true
	}

	worst := gtfs.Alert_NO_EFFECT
	seen := map[string]bool{}
	for _, e := range feed.GetEntity() {
		alert := e.GetAlert()
		if alert == nil || !relevant(alert, c.RouteID, stopSet) {
			continue
		}
		if text := headerText(alert); text != "" && !seen[text] {
			out.Alerts = append(out.Alerts, text)
			seen[text] = true
		}
		if eff := alert.GetEffect(); severity(eff) > severity(worst) {
			worst = eff
		}
	}
	if worst != gtfs.Alert_NO_EFFECT {
		out.Status = statusFor(worst)
	}
	return out
}

// relevant reports whether an alert affects routeID in a way the rider sees.
// When stopSet is non-empty, a stop-specific alert must touch one of those
// stops; line-wide alerts (no stop) and all alerts when stopSet is empty pass.
func relevant(a *gtfs.Alert, routeID string, stopSet map[string]bool) bool {
	routeMatch := false
	var stops []string
	for _, e := range a.GetInformedEntity() {
		if e.GetRouteId() == routeID {
			routeMatch = true
		}
		if s := e.GetStopId(); s != "" {
			stops = append(stops, s)
		}
	}
	if !routeMatch {
		return false
	}
	if len(stopSet) == 0 || len(stops) == 0 {
		return true
	}
	for _, s := range stops {
		if stopSet[s] {
			return true
		}
	}
	return false
}

func headerText(a *gtfs.Alert) string {
	for _, tr := range a.GetHeaderText().GetTranslation() {
		if tr.GetText() != "" {
			return tr.GetText()
		}
	}
	return ""
}

// severity ranks effects so the worst one drives the headline status.
func severity(e gtfs.Alert_Effect) int {
	switch e {
	case gtfs.Alert_SIGNIFICANT_DELAYS:
		return 3
	case gtfs.Alert_DETOUR, gtfs.Alert_MODIFIED_SERVICE, gtfs.Alert_REDUCED_SERVICE, gtfs.Alert_NO_SERVICE:
		return 2
	default:
		return 1
	}
}

func statusFor(e gtfs.Alert_Effect) string {
	switch e {
	case gtfs.Alert_SIGNIFICANT_DELAYS:
		return "Delays"
	default:
		return "Service Change"
	}
}
