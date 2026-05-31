// Command commute serves a self-hosted commute dashboard and posts weekday
// Discord nudges.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/keshaldesai/door-to-door/config"
	"github.com/keshaldesai/door-to-door/dashboard"
	"github.com/keshaldesai/door-to-door/drive"
	"github.com/keshaldesai/door-to-door/gtfs"
	"github.com/keshaldesai/door-to-door/mnr"
	"github.com/keshaldesai/door-to-door/model"
	"github.com/keshaldesai/door-to-door/notify"
	"github.com/keshaldesai/door-to-door/scheduler"
	"github.com/keshaldesai/door-to-door/server"
	"github.com/keshaldesai/door-to-door/subway"
	"github.com/keshaldesai/door-to-door/weather"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("timezone: %v", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load the static MNR schedule at startup; refresh it daily in the background.
	zipBytes, err := gtfs.Download(ctx, httpClient, cfg.Feeds.MNRStaticGTFS)
	if err != nil {
		log.Fatalf("download MNR GTFS: %v", err)
	}
	initialSchedule, err := gtfs.Load(zipBytes)
	if err != nil {
		log.Fatalf("parse MNR GTFS: %v", err)
	}
	var schedule atomic.Pointer[gtfs.Schedule]
	schedule.Store(initialSchedule)
	go refreshScheduleDaily(ctx, httpClient, cfg.Feeds.MNRStaticGTFS, &schedule)

	weatherClient := &weather.Client{HTTP: httpClient, Base: "https://api.weather.gov", UserAgent: cfg.Weather.UserAgent, Loc: loc}
	subwayClient := &subway.Client{HTTP: httpClient, URL: cfg.Feeds.SubwayAlerts, RouteID: cfg.Subway.RouteID, StopIDs: cfg.Subway.StopIDs}
	driveClient := &drive.Client{HTTP: httpClient, Base: "https://maps.googleapis.com/maps/api/distancematrix/json", Key: cfg.GoogleMapsKey}
	mnrClient := &mnr.Client{HTTP: httpClient, URL: cfg.Feeds.MNRRealtime}
	tripsClient := &subway.TripsClient{HTTP: httpClient, URL: cfg.Feeds.SubwayRealtime, RouteID: cfg.Subway.RouteID}

	build := func(ctx context.Context) model.Snapshot {
		now := time.Now().In(loc)
		sched := schedule.Load()
		feed, feedErr := mnrClient.Fetch(ctx)

		// trainLeg builds one direction. leaveOffsetMin and expectedTrack are
		// per-direction config; expectedTrack is for the boarding (origin) stop.
		trainLeg := func(origin, dest, originLabel, destLabel string, leaveOffsetMin int, expectedTrack string) model.TrainLeg {
			leg := model.TrainLeg{
				Origin: originLabel, Dest: destLabel, Source: "scheduled", UpdatedAt: now,
				LeaveOffsetMin: leaveOffsetMin,
				ExpectedTrack:  expectedTrack,
			}
			leg.Trains = sched.NextDepartures(origin, dest, now, cfg.TrainsToShow)
			if feedErr != nil {
				leg.Err = feedErr.Error()
				return leg
			}
			leg.Trains = mnr.Overlay(leg.Trains, feed, origin)
			leg.Source = "realtime"
			return leg
		}

		home, work := cfg.Stops.Home, cfg.Stops.Work
		fetchers := dashboard.Fetchers{
			Weather: func(ctx context.Context) model.Weather {
				return weatherClient.Fetch(ctx, cfg.Home.Lat, cfg.Home.Lon)
			},
			Drive: func(ctx context.Context) model.DriveLeg {
				// One estimate for the short home<->station leg, reused both ways.
				return driveClient.Fetch(ctx, cfg.Home.Lat, cfg.Home.Lon, cfg.Station.Lat, cfg.Station.Lon)
			},
			Subway: subwayClient.Fetch,
			Outbound: func(ctx context.Context) model.TrainLeg {
				return trainLeg(home.ID, work.ID, home.Label, work.Label,
					cfg.LeaveBeforeTrainMinutes.Outbound, cfg.ExpectedTracks["home"])
			},
			Inbound: func(ctx context.Context) model.TrainLeg {
				return trainLeg(work.ID, home.ID, work.Label, home.Label,
					cfg.LeaveBeforeTrainMinutes.Inbound, cfg.ExpectedTracks["work"])
			},
			OutboundSubway: func(ctx context.Context) model.SubwayCountdown {
				if cfg.Feeds.SubwayRealtime == "" || cfg.Subway.Outbound.StopID == "" {
					return model.SubwayCountdown{}
				}
				return tripsClient.Fetch(ctx, cfg.Subway.Outbound.StopID, cfg.Subway.Outbound.DirectionID, cfg.TrainsToShow, now)
			},
			InboundSubway: func(ctx context.Context) model.SubwayCountdown {
				if cfg.Feeds.SubwayRealtime == "" || cfg.Subway.Inbound.StopID == "" {
					return model.SubwayCountdown{}
				}
				return tripsClient.Fetch(ctx, cfg.Subway.Inbound.StopID, cfg.Subway.Inbound.DirectionID, cfg.TrainsToShow, now)
			},
		}
		snap := dashboard.Build(ctx, fetchers, func() time.Time { return now })
		snap.PrimaryDirection = primaryDirection(now, cfg.EveningSwitchAt)
		return snap
	}

	srv := server.New(build)
	go srv.RefreshLoop(ctx, time.Duration(cfg.RefreshSeconds)*time.Second)

	if cfg.DiscordWebhookURL != "" {
		go scheduler.Run(ctx, cfg.Nudges.Morning, cfg.Nudges.Evening, loc,
			srv.Snapshot,
			func(which string, snap model.Snapshot) {
				msg := notify.Summary(snap, which, cfg.DashboardURL)
				if err := notify.Post(ctx, httpClient, cfg.DiscordWebhookURL, msg); err != nil {
					log.Printf("discord post: %v", err)
				}
			})
	} else {
		log.Print("DISCORD_WEBHOOK_URL not set; nudges disabled")
	}

	httpServer := &http.Server{Addr: cfg.Server.Addr, Handler: srv.Handler()}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutCtx)
	}()

	log.Printf("commute dashboard on %s", cfg.Server.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

// primaryDirection returns "outbound" when now is before the HH:MM cutoff
// (in now's location) and "inbound" at or after. Empty cutoff means
// always "outbound". An unparseable cutoff also yields "outbound" - it
// will have been rejected at config load, so this is defensive.
func primaryDirection(now time.Time, cutoff string) string {
	if cutoff == "" {
		return "outbound"
	}
	t, err := time.Parse("15:04", cutoff)
	if err != nil {
		return "outbound"
	}
	y, m, d := now.Date()
	switchAt := time.Date(y, m, d, t.Hour(), t.Minute(), 0, 0, now.Location())
	if now.Before(switchAt) {
		return "outbound"
	}
	return "inbound"
}

// refreshScheduleDaily re-downloads and reparses the static GTFS once a day,
// swapping it in atomically. Failures are logged and the previous schedule kept.
func refreshScheduleDaily(ctx context.Context, client *http.Client, url string, dst *atomic.Pointer[gtfs.Schedule]) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			zipBytes, err := gtfs.Download(ctx, client, url)
			if err != nil {
				log.Printf("gtfs daily refresh download: %v", err)
				continue
			}
			s, err := gtfs.Load(zipBytes)
			if err != nil {
				log.Printf("gtfs daily refresh parse: %v", err)
				continue
			}
			dst.Store(s)
			log.Print("refreshed MNR static schedule")
		}
	}
}
