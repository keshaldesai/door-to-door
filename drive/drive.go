// Package drive returns a traffic-aware drive duration between two points using
// the Google Maps Distance Matrix API.
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/keshaldee/commute/model"
)

type Client struct {
	HTTP *http.Client
	Base string // e.g. https://maps.googleapis.com/maps/api/distancematrix/json
	Key  string
}

func (c *Client) Fetch(ctx context.Context, oLat, oLon, dLat, dLon float64) model.DriveLeg {
	out := model.DriveLeg{UpdatedAt: time.Now()}
	if c.Key == "" {
		out.Err = "no Google Maps key configured"
		return out
	}

	q := url.Values{}
	q.Set("origins", fmt.Sprintf("%g,%g", oLat, oLon))
	q.Set("destinations", fmt.Sprintf("%g,%g", dLat, dLon))
	q.Set("departure_time", "now")
	q.Set("key", c.Key)
	u := c.Base + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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

	var body struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Rows         []struct {
			Elements []struct {
				Status            string `json:"status"`
				Duration          struct{ Value int `json:"value"` } `json:"duration"`
				DurationInTraffic struct{ Value int `json:"value"` } `json:"duration_in_traffic"`
			} `json:"elements"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		out.Err = err.Error()
		return out
	}
	if body.Status != "OK" {
		out.Err = fmt.Sprintf("distance matrix: %s %s", body.Status, body.ErrorMessage)
		return out
	}
	if len(body.Rows) == 0 || len(body.Rows[0].Elements) == 0 {
		out.Err = "distance matrix: empty result"
		return out
	}
	el := body.Rows[0].Elements[0]
	if el.Status != "OK" {
		out.Err = fmt.Sprintf("distance matrix element: %s", el.Status)
		return out
	}
	secs := el.DurationInTraffic.Value
	if secs == 0 {
		secs = el.Duration.Value
	}
	out.DurationMin = int(math.Round(float64(secs) / 60))
	return out
}
