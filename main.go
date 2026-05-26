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
	"syscall"
	"time"

	"github.com/keshaldee/commute/config"
	"github.com/keshaldee/commute/dashboard"
	"github.com/keshaldee/commute/drive"
	"github.com/keshaldee/commute/gtfs"
	"github.com/keshaldee/commute/mnr"
	"github.com/keshaldee/commute/model"
	"github.com/keshaldee/commute/notify"
	"github.com/keshaldee/commute/scheduler"
	"github.com/keshaldee/commute/server"
	"github.com/keshaldee/commute/subway"
	"github.com/keshaldee/commute/weather"
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

	// Load the static MNR schedule once at startup.
	zipBytes, err := gtfs.Download(ctx, httpClient, cfg.Feeds.MNRStaticGTFS)
	if err != nil {
		log.Fatalf("download MNR GTFS: %v", err)
	}
	sched, err := gtfs.Load(zipBytes)
	if err != nil {
		log.Fatalf("parse MNR GTFS: %v", err)
	}

	weatherClient := &weather.Client{HTTP: httpClient, Base: "https://api.weather.gov", UserAgent: cfg.Weather.UserAgent}
	subwayClient := &subway.Client{HTTP: httpClient, URL: cfg.Feeds.SubwayAlerts, RouteID: cfg.Subway.RouteID}
	driveClient := &drive.Client{HTTP: httpClient, Base: "https://maps.googleapis.com/maps/api/distancematrix/json", Key: cfg.GoogleMapsKey}
	mnrClient := &mnr.Client{HTTP: httpClient, URL: cfg.Feeds.MNRRealtime}

	trainLeg := func(origin, dest, originLabel, destLabel string) func(context.Context) model.TrainLeg {
		return func(ctx context.Context) model.TrainLeg {
			now := time.Now().In(loc)
			leg := model.TrainLeg{Origin: originLabel, Dest: destLabel, Source: "scheduled", UpdatedAt: now}
			leg.Trains = sched.NextDepartures(origin, dest, now, cfg.TrainsToShow)
			feed, err := mnrClient.Fetch(ctx)
			if err != nil {
				leg.Err = err.Error()
				return leg
			}
			leg.Trains = mnr.Overlay(leg.Trains, feed, origin)
			leg.Source = "realtime"
			return leg
		}
	}

	fetchers := dashboard.Fetchers{
		Weather: func(ctx context.Context) model.Weather {
			return weatherClient.Fetch(ctx, cfg.Home.Lat, cfg.Home.Lon)
		},
		Drive: func(ctx context.Context) model.DriveLeg {
			// One estimate for the short home<->station leg, reused both ways.
			return driveClient.Fetch(ctx, cfg.Home.Lat, cfg.Home.Lon, cfg.Station.Lat, cfg.Station.Lon)
		},
		Subway:   subwayClient.Fetch,
		Outbound: trainLeg(cfg.Stops.Home, cfg.Stops.Work, "Home", "Work"),
		Inbound:  trainLeg(cfg.Stops.Work, cfg.Stops.Home, "Work", "Home"),
	}

	build := func(ctx context.Context) model.Snapshot {
		return dashboard.Build(ctx, fetchers, func() time.Time { return time.Now().In(loc) })
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
