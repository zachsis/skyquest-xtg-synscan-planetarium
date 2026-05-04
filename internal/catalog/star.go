package catalog

// Star is one entry in the embedded star catalog.
type Star struct {
	HipID         int
	Name          string  // Proper name, empty string if none
	RA            float64 // Right ascension in hours [0, 24), J2000
	Dec           float64 // Declination in degrees [-90, +90], J2000
	Mag           float64 // Apparent visual magnitude
	SpectralType  string  // e.g., "G2V"
	Constellation string  // IAU abbreviation, e.g., "Ori"
}
