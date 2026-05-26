# Commute Helper

A self-hosted dashboard for a fixed Metro-North MNR line commute: drive to
Home, train to Work, subway to Office Stop, and the reverse. It
shows the next few trains each way with live status and track, the subway
service status, weather alerts, and a traffic-aware drive time. It also posts a
short Discord nudge on weekday mornings and evenings.

## Build

    go build -o commute .

This produces a single binary with the dashboard assets embedded. (A Nix dev
shell is provided via flake.nix and .envrc if you use direnv.)

## Configure

1. Copy the example config and edit it:

       cp config.example.yaml config.yaml

   Set your home coordinates, your station coordinates (the drive destination),
   and the Metro-North GTFS stop_ids for your station and Work. Verify
   the MTA feed URLs against the current MTA developer docs. To find stop ids,
   download the GTFS zip from the configured URL and grep stops.txt; Grand
   Central is "1" in the Metro-North feed.

2. Set secrets in the environment (never commit these):

       export GOOGLE_MAPS_KEY=your-distance-matrix-key
       export DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...

   The Google Maps key needs the Distance Matrix API enabled. Without it the
   dashboard runs fine and the drive panel shows an error. Without the Discord
   webhook, nudges are disabled and the dashboard still works.

## Run

    ./commute            # uses ./config.yaml
    ./commute -config /path/to/config.yaml

Open http://localhost:8080 (or your configured address).

## Data sources

- Metro-North schedule and realtime: MTA static GTFS + GTFS-realtime (keyless).
- subway status: MTA subway service-alerts feed (keyless).
- Weather: National Weather Service API (keyless; requires a User-Agent).
- Drive time: Google Maps Distance Matrix API (requires a key).

## License

MIT (see LICENSE).
