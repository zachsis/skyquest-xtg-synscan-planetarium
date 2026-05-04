package synscan

// TrackingMode represents the telescope's tracking state.
type TrackingMode byte

const (
	TrackingOff      TrackingMode = 0 // Tracking disabled
	TrackingSidereal TrackingMode = 1 // Sidereal rate (alt-az mode for XT12G)
)

// String returns a human-readable tracking mode name.
func (m TrackingMode) String() string {
	switch m {
	case TrackingOff:
		return "Off"
	case TrackingSidereal:
		return "Sidereal"
	default:
		return "Unknown"
	}
}
