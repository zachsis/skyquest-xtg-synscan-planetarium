package logging

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func TestExportCSV_AllSessions(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{
		StartedAt:    time.Date(2026, 1, 15, 20, 0, 0, 0, time.UTC),
		LocationName: "Backyard",
	}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	seeing := 3
	ob := &Observation{
		SessionID:  sess.ID,
		Timestamp:  time.Date(2026, 1, 15, 21, 30, 0, 0, time.UTC),
		TargetName: "M31",
		CatalogID:  "M31",
		RAHours:    0.712,
		DecDegrees: 41.269,
		Notes:      "hazy",
		Seeing:     &seeing,
		AutoLogged: true,
	}
	if err := store.AddObservation(ob); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	var buf bytes.Buffer
	if err := store.ExportCSV(&buf, nil); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}

	// header + 1 data row
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	if records[0][0] != "Session Date" {
		t.Errorf("header[0]: got %q", records[0][0])
	}

	row := records[1]
	if row[0] != "2026-01-15" {
		t.Errorf("Session Date: got %q", row[0])
	}
	if row[1] != "Backyard" {
		t.Errorf("Session Location: got %q", row[1])
	}
	if row[3] != "M31" {
		t.Errorf("Target Name: got %q", row[3])
	}
	if row[4] != "M31" {
		t.Errorf("Catalog ID: got %q", row[4])
	}
	if row[7] != "3" {
		t.Errorf("Seeing: got %q, want 3", row[7])
	}
	if row[8] != "" {
		t.Errorf("Transparency: got %q, want empty", row[8])
	}
	if row[9] != "hazy" {
		t.Errorf("Notes: got %q", row[9])
	}
}

func TestExportCSV_FilterBySession(t *testing.T) {
	store := openMemStore(t)

	sess1 := &Session{
		StartedAt:    time.Date(2026, 1, 10, 20, 0, 0, 0, time.UTC),
		LocationName: "Site A",
	}
	sess2 := &Session{
		StartedAt:    time.Date(2026, 1, 11, 20, 0, 0, 0, time.UTC),
		LocationName: "Site B",
	}
	for _, s := range []*Session{sess1, sess2} {
		if err := store.CreateSession(s); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
	}

	for _, pair := range []struct {
		sess *Session
		name string
	}{
		{sess1, "M42"},
		{sess2, "M45"},
	} {
		if err := store.AddObservation(&Observation{
			SessionID:  pair.sess.ID,
			Timestamp:  time.Now().UTC(),
			TargetName: pair.name,
			RAHours:    1.0,
			DecDegrees: 0.0,
			AutoLogged: true,
		}); err != nil {
			t.Fatalf("AddObservation: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := store.ExportCSV(&buf, &sess1.ID); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}

	// header + 1 row for sess1 only
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d\n%s", len(records), buf.String())
	}
	if records[1][3] != "M42" {
		t.Errorf("expected M42, got %q", records[1][3])
	}
}

func TestExportCSV_Empty(t *testing.T) {
	store := openMemStore(t)

	var buf bytes.Buffer
	if err := store.ExportCSV(&buf, nil); err != nil {
		t.Fatalf("ExportCSV empty: %v", err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	// only header
	if len(records) != 1 {
		t.Fatalf("expected 1 row (header only), got %d", len(records))
	}
}
