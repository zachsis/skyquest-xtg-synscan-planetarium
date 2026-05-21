package logging

import (
	"database/sql"
	"fmt"
	"io"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);
INSERT OR IGNORE INTO schema_version VALUES (1);

CREATE TABLE IF NOT EXISTS sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at    TEXT    NOT NULL,
    ended_at      TEXT,
    observer_lat  REAL,
    observer_lon  REAL,
    observer_elev REAL,
    location_name TEXT,
    notes         TEXT    DEFAULT ''
);

CREATE TABLE IF NOT EXISTS observations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES sessions(id),
    timestamp       TEXT    NOT NULL,
    target_name     TEXT    NOT NULL,
    catalog_id      TEXT,
    ra_hours        REAL    NOT NULL,
    dec_degrees     REAL    NOT NULL,
    notes           TEXT    DEFAULT '',
    seeing          INTEGER,
    transparency    INTEGER,
    equipment_notes TEXT    DEFAULT '',
    auto_logged     INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_observations_session
    ON observations(session_id);
CREATE INDEX IF NOT EXISTS idx_observations_timestamp
    ON observations(timestamp);
`

// SQLiteLogStore is a LogStore backed by a SQLite database.
type SQLiteLogStore struct {
	db  *sql.DB
	mu  sync.Mutex
}

// OpenSQLiteStore opens (or creates) the SQLite database at path and
// applies the schema.  Use ":memory:" in tests.
func OpenSQLiteStore(path string) (*SQLiteLogStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("logging: open db: %w", err)
	}

	// WAL mode + busy timeout for concurrent access.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("logging: set WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("logging: set busy timeout: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("logging: apply schema: %w", err)
	}

	return &SQLiteLogStore{db: db}, nil
}

// Close releases the underlying database connection.
func (s *SQLiteLogStore) Close() error {
	return s.db.Close()
}

// --- Session operations ---

// CreateSession inserts a new session and populates s.ID.
func (s *SQLiteLogStore) CreateSession(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		`INSERT INTO sessions
			(started_at, observer_lat, observer_lon, observer_elev,
			 location_name, notes)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sess.StartedAt.UTC().Format(time.RFC3339),
		sess.ObserverLat,
		sess.ObserverLon,
		sess.ObserverElev,
		sess.LocationName,
		sess.Notes,
	)
	if err != nil {
		return fmt.Errorf("logging: create session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("logging: create session last id: %w", err)
	}
	sess.ID = id
	return nil
}

// CloseSession sets ended_at on the given session.
func (s *SQLiteLogStore) CloseSession(id int64, endedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		`UPDATE sessions SET ended_at = ? WHERE id = ?`,
		endedAt.UTC().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return fmt.Errorf("logging: close session: %w", err)
	}
	return nil
}

// DeleteEmptySession removes a session only if it has no observations.
func (s *SQLiteLogStore) DeleteEmptySession(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM observations WHERE session_id = ?`, id,
	).Scan(&count); err != nil {
		return fmt.Errorf("logging: count observations: %w", err)
	}
	if count > 0 {
		return nil // not empty, leave it
	}

	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("logging: delete empty session: %w", err)
	}
	return nil
}

// --- Observation operations ---

// AddObservation inserts a new observation and populates o.ID.
func (s *SQLiteLogStore) AddObservation(o *Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	autoLogged := 0
	if o.AutoLogged {
		autoLogged = 1
	}

	res, err := s.db.Exec(
		`INSERT INTO observations
			(session_id, timestamp, target_name, catalog_id,
			 ra_hours, dec_degrees, notes, seeing, transparency,
			 equipment_notes, auto_logged)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.SessionID,
		o.Timestamp.UTC().Format(time.RFC3339),
		o.TargetName,
		o.CatalogID,
		o.RAHours,
		o.DecDegrees,
		o.Notes,
		intPtrToSQL(o.Seeing),
		intPtrToSQL(o.Transparency),
		o.EquipmentNotes,
		autoLogged,
	)
	if err != nil {
		return fmt.Errorf("logging: add observation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("logging: add observation last id: %w", err)
	}
	o.ID = id
	return nil
}

// UpdateObservation writes mutable fields back to the database.
func (s *SQLiteLogStore) UpdateObservation(o *Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		`UPDATE observations
		 SET target_name = ?, catalog_id = ?, notes = ?,
		     seeing = ?, transparency = ?, equipment_notes = ?
		 WHERE id = ?`,
		o.TargetName,
		o.CatalogID,
		o.Notes,
		intPtrToSQL(o.Seeing),
		intPtrToSQL(o.Transparency),
		o.EquipmentNotes,
		o.ID,
	)
	if err != nil {
		return fmt.Errorf("logging: update observation: %w", err)
	}
	return nil
}

// --- Queries ---

// ListSessions returns all sessions, most recent first, with observation counts.
func (s *SQLiteLogStore) ListSessions() ([]SessionSummary, error) {
	rows, err := s.db.Query(`
		SELECT
			se.id,
			se.started_at,
			se.ended_at,
			se.location_name,
			COUNT(ob.id) AS obs_count
		FROM sessions se
		LEFT JOIN observations ob ON ob.session_id = se.id
		GROUP BY se.id
		ORDER BY se.started_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("logging: list sessions: %w", err)
	}
	defer rows.Close()

	var out []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		var startedStr string
		var endedStr sql.NullString

		if err := rows.Scan(
			&ss.ID, &startedStr, &endedStr, &ss.LocationName,
			&ss.ObservationCount,
		); err != nil {
			return nil, fmt.Errorf("logging: scan session: %w", err)
		}
		ss.StartedAt, err = time.Parse(time.RFC3339, startedStr)
		if err != nil {
			return nil, fmt.Errorf(
				"logging: parse session started_at: %w", err,
			)
		}
		if endedStr.Valid {
			t, err := time.Parse(time.RFC3339, endedStr.String)
			if err != nil {
				return nil, fmt.Errorf(
					"logging: parse session ended_at: %w", err,
				)
			}
			ss.EndedAt = &t
		}
		out = append(out, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("logging: list sessions rows: %w", err)
	}
	return out, nil
}

// ListObservations returns all observations for a session, earliest first.
func (s *SQLiteLogStore) ListObservations(sessionID int64) ([]Observation, error) {
	rows, err := s.db.Query(`
		SELECT
			id, session_id, timestamp, target_name, catalog_id,
			ra_hours, dec_degrees, notes, seeing, transparency,
			equipment_notes, auto_logged
		FROM observations
		WHERE session_id = ?
		ORDER BY timestamp ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"logging: list observations: %w", err,
		)
	}
	defer rows.Close()

	var out []Observation
	for rows.Next() {
		var o Observation
		var tsStr string
		var seeing, transparency sql.NullInt64
		var autoLogged int

		if err := rows.Scan(
			&o.ID, &o.SessionID, &tsStr, &o.TargetName, &o.CatalogID,
			&o.RAHours, &o.DecDegrees, &o.Notes,
			&seeing, &transparency,
			&o.EquipmentNotes, &autoLogged,
		); err != nil {
			return nil, fmt.Errorf(
				"logging: scan observation: %w", err,
			)
		}
		o.Timestamp, err = time.Parse(time.RFC3339, tsStr)
		if err != nil {
			return nil, fmt.Errorf(
				"logging: parse observation timestamp: %w", err,
			)
		}
		if seeing.Valid {
			v := int(seeing.Int64)
			o.Seeing = &v
		}
		if transparency.Valid {
			v := int(transparency.Int64)
			o.Transparency = &v
		}
		o.AutoLogged = autoLogged != 0
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"logging: list observations rows: %w", err,
		)
	}
	return out, nil
}

// ExportCSV writes CSV data to w.  If sessionID is non-nil only that session
// is exported; otherwise all sessions are exported.
func (s *SQLiteLogStore) ExportCSV(w io.Writer, sessionID *int64) error {
	return exportCSV(s.db, w, sessionID)
}

// --- helpers ---

func intPtrToSQL(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
