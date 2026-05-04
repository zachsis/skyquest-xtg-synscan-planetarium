package astro

import "time"

// LocationProvider gives the observer's geographic location.
// This avoids a circular dependency between astro and config packages.
type LocationProvider interface {
	GetLatitude() float64
	GetLongitude() float64
	GetElevation() float64
}

// AstroService provides convenience methods for coordinate transforms
// that automatically use the observer's configured location.
type AstroService struct {
	Location LocationProvider
}

// NewAstroService creates a new AstroService.
func NewAstroService(loc LocationProvider) *AstroService {
	return &AstroService{Location: loc}
}

func (s *AstroService) location() GeographicLocation {
	return GeographicLocation{
		Latitude:  s.Location.GetLatitude(),
		Longitude: s.Location.GetLongitude(),
		Elevation: s.Location.GetElevation(),
	}
}

// EquatorialToHorizontal converts RA/Dec to Alt/Az for the observer's location.
func (s *AstroService) EquatorialToHorizontal(eq Equatorial, t time.Time) Horizontal {
	return EquatorialToHorizontal(eq, s.location(), t)
}

// RiseSetTimes calculates rise/transit/set for the observer's location.
func (s *AstroService) RiseSetTimes(eq Equatorial, date time.Time) (rise, transit, set time.Time, err error) {
	return RiseSetTimes(eq, s.location(), date)
}

// LST returns the current Local Sidereal Time in hours.
func (s *AstroService) LST(t time.Time) float64 {
	return LocalSiderealTime(s.Location.GetLongitude(), t)
}

// AltAz converts RA/Dec (hours, degrees) to Alt/Az for the observer.
func (s *AstroService) AltAz(ra, dec float64, t time.Time) (alt, az float64) {
	h := s.EquatorialToHorizontal(Equatorial{RA: ra, Dec: dec}, t)
	return h.Alt, h.Az
}

// RiseSet returns rise and set times for the observer (omits transit).
func (s *AstroService) RiseSet(ra, dec float64, date time.Time) (rise, set time.Time, err error) {
	rise, _, set, err = s.RiseSetTimes(Equatorial{RA: ra, Dec: dec}, date)
	return
}

// IsAboveHorizon returns true if the object is above the horizon.
func (s *AstroService) IsAboveHorizon(ra, dec float64, t time.Time) bool {
	alt, _ := s.AltAz(ra, dec, t)
	return alt > 0
}
