package logging

import (
	"io"
	"time"
)

// LogStore is the persistence interface for observation sessions and
// individual observations.
type LogStore interface {
	// Session lifecycle.
	CreateSession(s *Session) error
	CloseSession(id int64, endedAt time.Time) error
	DeleteEmptySession(id int64) error

	// Observation CRUD.
	AddObservation(o *Observation) error
	UpdateObservation(o *Observation) error

	// Queries.
	ListSessions() ([]SessionSummary, error)
	ListObservations(sessionID int64) ([]Observation, error)

	// Export.
	ExportCSV(w io.Writer, sessionID *int64) error

	// Close releases the underlying database connection.
	Close() error
}
