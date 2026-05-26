package mnr

import (
	"testing"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/keshaldee/commute/model"
	"google.golang.org/protobuf/proto"
)

func TestOverlayAppliesDelay(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	dep := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	trains := []model.Train{{TripID: "T1", Departure: dep, Status: "On time"}}

	delay := int32(360) // 6 minutes late
	tid := "T1"
	stop := "FD"
	feed := &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*gtfs.FeedEntity{{
			Id: proto.String("e1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip: &gtfs.TripDescriptor{TripId: &tid},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{
					StopId:    &stop,
					Departure: &gtfs.TripUpdate_StopTimeEvent{Delay: &delay},
				}},
			},
		}},
	}

	got := Overlay(trains, feed, "FD")
	if !got[0].Realtime {
		t.Fatal("expected realtime flag")
	}
	if got[0].DelayMin != 6 || got[0].Status != "Delayed 6m" {
		t.Fatalf("got delay=%d status=%q", got[0].DelayMin, got[0].Status)
	}
	if !got[0].Departure.Equal(dep.Add(6 * time.Minute)) {
		t.Fatalf("departure not adjusted: %v", got[0].Departure)
	}
}

func TestOverlayLeavesUnmatchedTrains(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	trains := []model.Train{{TripID: "T9", Departure: time.Now().In(loc), Status: "On time"}}
	feed := &gtfs.FeedMessage{Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")}}
	got := Overlay(trains, feed, "FD")
	if got[0].Realtime || got[0].Status != "On time" {
		t.Fatalf("unmatched train changed: %+v", got[0])
	}
}
