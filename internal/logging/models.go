package logging

import "time"

// Session represents a single observation session.
type Session struct {
	ID           int64
	StartedAt    time.Time
	EndedAt      *time.Time // nil if still active
	ObserverLat  float64
	ObserverLon  float64
	ObserverElev float64
	LocationName string
	Notes        string
}

// SessionSummary is a lightweight projection of Session with
// an aggregate observation count.
type SessionSummary struct {
	ID               int64
	StartedAt        time.Time
	EndedAt          *time.Time
	LocationName     string
	ObservationCount int
}

// Observation records one pointing event within a session.
type Observation struct {
	ID             int64
	SessionID      int64
	Timestamp      time.Time
	TargetName     string
	CatalogID      string
	RAHours        float64
	DecDegrees     float64
	Notes          string
	Seeing         *int // 1-5, nil if not rated
	Transparency   *int // 1-5, nil if not rated
	EquipmentNotes string
	AutoLogged     bool
}
