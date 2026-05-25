// Package weather fetches active alerts and a short forecast from the National
// Weather Service API (keyless).
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/keshaldee/commute/model"
)

type Client struct {
	HTTP      *http.Client
	Base      string // e.g. https://api.weather.gov
	UserAgent string
}

func (c *Client) Fetch(ctx context.Context, lat, lon float64) model.Weather {
	out := model.Weather{UpdatedAt: time.Now()}

	pointsURL := fmt.Sprintf("%s/points/%g,%g", c.Base, lat, lon)
	var points struct {
		Properties struct {
			ForecastHourly string `json:"forecastHourly"`
		} `json:"properties"`
	}
	if err := c.getJSON(ctx, pointsURL, &points); err != nil {
		out.Err = err.Error()
		return out
	}

	var hourly struct {
		Properties struct {
			Periods []struct {
				Temperature         int    `json:"temperature"`
				ShortForecast       string `json:"shortForecast"`
				ProbabilityOfPrecip struct {
					Value int `json:"value"`
				} `json:"probabilityOfPrecipitation"`
			} `json:"periods"`
		} `json:"properties"`
	}
	if err := c.getJSON(ctx, points.Properties.ForecastHourly, &hourly); err != nil {
		out.Err = err.Error()
		return out
	}
	if len(hourly.Properties.Periods) > 0 {
		p := hourly.Properties.Periods[0]
		out.TempF = p.Temperature
		out.Summary = p.ShortForecast
		out.PrecipChance = p.ProbabilityOfPrecip.Value
	}

	alertsURL := fmt.Sprintf("%s/alerts/active?point=%s", c.Base, url.QueryEscape(fmt.Sprintf("%g,%g", lat, lon)))
	var alerts struct {
		Features []struct {
			Properties struct {
				Event    string `json:"event"`
				Headline string `json:"headline"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := c.getJSON(ctx, alertsURL, &alerts); err != nil {
		out.Err = err.Error()
		return out
	}
	for _, f := range alerts.Features {
		out.Alerts = append(out.Alerts, model.WeatherAlert{
			Event:    f.Properties.Event,
			Headline: f.Properties.Headline,
		})
	}
	return out
}

func (c *Client) getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("nws %s: %d %s", u, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
