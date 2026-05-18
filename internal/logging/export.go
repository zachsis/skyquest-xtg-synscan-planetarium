package logging

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

var csvHeader = []string{
	"Session Date",
	"Session Location",
	"Timestamp",
	"Target Name",
	"Catalog ID",
	"RA",
	"Dec",
	"Seeing",
	"Transparency",
	"Notes",
	"Equipment Notes",
}

// exportCSV writes session + observation rows to w using encoding/csv.
// When sessionID is nil all sessions are exported.
func exportCSV(db *sql.DB, w io.Writer, sessionID *int64) error {
	query := `
		SELECT
			se.started_at,
			se.location_name,
			ob.timestamp,
			ob.target_name,
			ob.catalog_id,
			ob.ra_hours,
			ob.dec_degrees,
			ob.seeing,
			ob.transparency,
			ob.notes,
			ob.equipment_notes
		FROM sessions se
		JOIN observations ob ON ob.session_id = se.id`

	var rows *sql.Rows
	var err error
	if sessionID != nil {
		rows, err = db.Query(query+" WHERE se.id = ? ORDER BY ob.timestamp ASC", *sessionID)
	} else {
		rows, err = db.Query(query + " ORDER BY se.started_at DESC, ob.timestamp ASC")
	}
	if err != nil {
		return fmt.Errorf("logging: export query: %w", err)
	}
	defer rows.Close()

	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return fmt.Errorf("logging: write csv header: %w", err)
	}

	for rows.Next() {
		var (
			sessStartedStr string
			sessLocation   sql.NullString
			obTSStr        string
			targetName     string
			catalogID      sql.NullString
			raHours        float64
			decDeg         float64
			seeing         sql.NullInt64
			transparency   sql.NullInt64
			notes          sql.NullString
			equipNotes     sql.NullString
		)

		if err := rows.Scan(
			&sessStartedStr, &sessLocation,
			&obTSStr, &targetName, &catalogID,
			&raHours, &decDeg,
			&seeing, &transparency, &notes, &equipNotes,
		); err != nil {
			return fmt.Errorf("logging: scan export row: %w", err)
		}

		sessStarted, err := time.Parse(time.RFC3339, sessStartedStr)
		if err != nil {
			return fmt.Errorf(
				"logging: parse export session date: %w", err,
			)
		}
		obTS, err := time.Parse(time.RFC3339, obTSStr)
		if err != nil {
			return fmt.Errorf(
				"logging: parse export obs timestamp: %w", err,
			)
		}

		record := []string{
			sessStarted.UTC().Format("2006-01-02"),
			nullStringVal(sessLocation),
			obTS.UTC().Format(time.RFC3339),
			targetName,
			nullStringVal(catalogID),
			strconv.FormatFloat(raHours, 'f', 6, 64),
			strconv.FormatFloat(decDeg, 'f', 6, 64),
			nullIntStr(seeing),
			nullIntStr(transparency),
			nullStringVal(notes),
			nullStringVal(equipNotes),
		}

		if err := cw.Write(record); err != nil {
			return fmt.Errorf("logging: write csv record: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("logging: export rows: %w", err)
	}

	cw.Flush()
	return cw.Error()
}

func nullStringVal(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullIntStr(ni sql.NullInt64) string {
	if ni.Valid {
		return strconv.FormatInt(ni.Int64, 10)
	}
	return ""
}
