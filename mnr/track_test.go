package mnr

import (
	"testing"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/keshaldee/commute/model"
	"github.com/keshaldee/commute/mtarr"
	"google.golang.org/protobuf/proto"
)

func TestOverlaySetsTrackFromExtension(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	dep := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	trains := []model.Train{{TripID: "T1", Departure: dep, Status: "On time"}}

	tid := "T1"
	stop := "B"
	delay := int32(0)
	stu := &gtfs.TripUpdate_StopTimeUpdate{
		StopId:    &stop,
		Departure: &gtfs.TripUpdate_StopTimeEvent{Delay: &delay},
	}
	proto.SetExtension(stu, mtarr.E_MtaRailroadStopTimeUpdate, &mtarr.MtaRailroadStopTimeUpdate{
		Track: proto.String("19"),
	})

	feed := &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*gtfs.FeedEntity{{
			Id: proto.String("e1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip:           &gtfs.TripDescriptor{TripId: &tid},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{stu},
			},
		}},
	}

	got := Overlay(trains, feed, "B")
	if got[0].Track != "19" {
		t.Fatalf("expected track %q, got %q", "19", got[0].Track)
	}
}

func TestOverlaySetsTrackWithoutTimingEvent(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	dep := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	trains := []model.Train{{TripID: "T1", Departure: dep, Status: "On time"}}

	tid := "T1"
	stop := "B"
	stu := &gtfs.TripUpdate_StopTimeUpdate{StopId: &stop} // no Departure/Arrival
	proto.SetExtension(stu, mtarr.E_MtaRailroadStopTimeUpdate, &mtarr.MtaRailroadStopTimeUpdate{
		Track: proto.String("19"),
	})

	feed := &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*gtfs.FeedEntity{{
			Id: proto.String("e1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip:           &gtfs.TripDescriptor{TripId: &tid},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{stu},
			},
		}},
	}

	got := Overlay(trains, feed, "B")
	if got[0].Track != "19" {
		t.Fatalf("expected track %q without a timing event, got %q", "19", got[0].Track)
	}
	if got[0].Realtime {
		t.Fatal("no timing event means the train should not be marked realtime")
	}
}

func TestOverlayLeavesTrackEmptyWhenAbsent(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	dep := time.Date(2026, 5, 25, 7, 10, 0, 0, loc)
	trains := []model.Train{{TripID: "T1", Departure: dep, Status: "On time"}}

	tid := "T1"
	stop := "B"
	delay := int32(0)
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

	got := Overlay(trains, feed, "B")
	if got[0].Track != "" {
		t.Fatalf("expected empty track, got %q", got[0].Track)
	}
}
