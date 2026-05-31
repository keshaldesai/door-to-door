package weather

import (
	"math"
	"time"
)

// Sun returns sunrise and sunset (UTC) at lat/lon for the civil day
// containing t (interpreted in t's location). Uses the standard NOAA
// solar-position formulas with a -0.833 degree zenith for atmospheric
// refraction. Accurate to about a minute, which is plenty for a
// dashboard. Returns (zero, zero) for polar day/night where the sun
// never crosses the horizon on that date.
func Sun(lat, lon float64, t time.Time) (sunrise, sunset time.Time) {
	// Anchor to the civil date the user sees (in t's local zone).
	y, m, d := t.Date()

	// Julian day for noon UTC of the civil date.
	noonUTC := time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
	jd := julianDay(noonUTC)
	n := jd - 2451545.0 + 0.0008 // days since J2000

	latR := lat * math.Pi / 180
	// Mean solar noon at lon (in days since J2000).
	jStar := n - lon/360.0
	// Solar mean anomaly (deg).
	M := math.Mod(357.5291+0.98560028*jStar, 360)
	mR := M * math.Pi / 180
	// Equation of the center (deg).
	C := 1.9148*math.Sin(mR) + 0.0200*math.Sin(2*mR) + 0.0003*math.Sin(3*mR)
	// Ecliptic longitude (deg).
	lambda := math.Mod(M+C+180.0+102.9372, 360)
	lambdaR := lambda * math.Pi / 180
	// Solar transit (Julian date of local solar noon).
	jTransit := 2451545.0 + jStar + 0.0053*math.Sin(mR) - 0.0069*math.Sin(2*lambdaR)
	// Declination of the sun.
	sinDec := math.Sin(lambdaR) * math.Sin(23.4397*math.Pi/180)
	decR := math.Asin(sinDec)
	// Hour angle for sun at -0.833 deg (refraction + solar radius).
	cosH := (math.Sin(-0.833*math.Pi/180) - math.Sin(latR)*math.Sin(decR)) /
		(math.Cos(latR) * math.Cos(decR))
	if cosH > 1 || cosH < -1 {
		return time.Time{}, time.Time{}
	}
	H := math.Acos(cosH) * 180 / math.Pi

	jRise := jTransit - H/360.0
	jSet := jTransit + H/360.0
	return julianToTime(jRise), julianToTime(jSet)
}

// julianDay returns the Julian Date for the given UTC time.
func julianDay(t time.Time) float64 {
	// Days since 2000-01-01 12:00 UTC, plus J2000 epoch.
	const j2000 = 2451545.0
	t = t.UTC()
	secs := t.Sub(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)).Seconds()
	return j2000 + secs/86400.0
}

// julianToTime converts a Julian Date to time.Time (UTC).
func julianToTime(jd float64) time.Time {
	secs := (jd - 2451545.0) * 86400.0
	return time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(secs * float64(time.Second)))
}
