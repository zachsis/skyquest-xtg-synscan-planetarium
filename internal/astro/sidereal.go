package astro

import (
	"time"

	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/unit"
)

// LocalSiderealTime returns the Local Sidereal Time in hours (0-24)
// for the given longitude (degrees, positive East) and UTC time.
func LocalSiderealTime(longitude float64, t time.Time) float64 {
	jd := julian.TimeToJD(t.UTC())
	gst := sidereal.Apparent(jd)
	lst := gst + unit.TimeFromHour(longitude/15.0)
	return normalizeHours(lst.Hour())
}
