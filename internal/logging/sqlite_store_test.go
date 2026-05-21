package logging

import (
	"testing"
	"time"
)

func openMemStore(t *testing.T) *SQLiteLogStore {
	t.Helper()
	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestCreateAndListSession(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{
		StartedAt:    time.Now().UTC().Truncate(time.Second),
		ObserverLat:  40.7,
		ObserverLon:  -74.0,
		ObserverElev: 10.0,
		LocationName: "Test Site",
		Notes:        "clear night",
	}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.ID == 0 {
		t.Fatal("CreateSession: ID not populated")
	}

	summaries, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("ListSessions: got %d, want 1", len(summaries))
	}
	if summaries[0].ID != sess.ID {
		t.Errorf("ID: got %d, want %d", summaries[0].ID, sess.ID)
	}
	if summaries[0].LocationName != "Test Site" {
		t.Errorf("LocationName: got %q", summaries[0].LocationName)
	}
	if summaries[0].ObservationCount != 0 {
		t.Errorf("ObservationCount: got %d, want 0", summaries[0].ObservationCount)
	}
}

func TestCloseSession(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	ended := time.Now().UTC().Truncate(time.Second)
	if err := store.CloseSession(sess.ID, ended); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	summaries, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if summaries[0].EndedAt == nil {
		t.Fatal("EndedAt should be set after CloseSession")
	}
	if !summaries[0].EndedAt.Equal(ended) {
		t.Errorf(
			"EndedAt: got %v, want %v",
			summaries[0].EndedAt, ended,
		)
	}
}

func TestDeleteEmptySession(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := store.DeleteEmptySession(sess.ID); err != nil {
		t.Fatalf("DeleteEmptySession: %v", err)
	}

	summaries, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(summaries))
	}
}

func TestDeleteEmptySession_NonEmpty(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	ob := &Observation{
		SessionID:  sess.ID,
		Timestamp:  time.Now().UTC(),
		TargetName: "M31",
		RAHours:    0.712,
		DecDegrees: 41.27,
		AutoLogged: true,
	}
	if err := store.AddObservation(ob); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Should be a no-op since the session has an observation.
	if err := store.DeleteEmptySession(sess.ID); err != nil {
		t.Fatalf("DeleteEmptySession: %v", err)
	}

	summaries, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected session to survive, got %d", len(summaries))
	}
}

func TestAddAndListObservations(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	seeing := 4
	trans := 3
	ob := &Observation{
		SessionID:      sess.ID,
		Timestamp:      time.Now().UTC().Truncate(time.Second),
		TargetName:     "Orion Nebula",
		CatalogID:      "M42",
		RAHours:        5.588,
		DecDegrees:     -5.391,
		Notes:          "beautiful",
		Seeing:         &seeing,
		Transparency:   &trans,
		EquipmentNotes: "8mm EP",
		AutoLogged:     false,
	}
	if err := store.AddObservation(ob); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if ob.ID == 0 {
		t.Fatal("AddObservation: ID not populated")
	}

	// ObservationCount should be 1 now.
	summaries, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if summaries[0].ObservationCount != 1 {
		t.Errorf(
			"ObservationCount: got %d, want 1",
			summaries[0].ObservationCount,
		)
	}

	obs, err := store.ListObservations(sess.ID)
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("ListObservations: got %d, want 1", len(obs))
	}
	got := obs[0]
	if got.TargetName != "Orion Nebula" {
		t.Errorf("TargetName: got %q", got.TargetName)
	}
	if got.CatalogID != "M42" {
		t.Errorf("CatalogID: got %q", got.CatalogID)
	}
	if got.Seeing == nil || *got.Seeing != seeing {
		t.Errorf("Seeing: got %v, want %d", got.Seeing, seeing)
	}
	if got.Transparency == nil || *got.Transparency != trans {
		t.Errorf(
			"Transparency: got %v, want %d", got.Transparency, trans,
		)
	}
	if got.AutoLogged {
		t.Error("AutoLogged should be false")
	}
}

func TestUpdateObservation(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	ob := &Observation{
		SessionID:  sess.ID,
		Timestamp:  time.Now().UTC(),
		TargetName: "M45",
		RAHours:    3.791,
		DecDegrees: 24.117,
		AutoLogged: true,
	}
	if err := store.AddObservation(ob); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	seeing := 5
	ob.Notes = "great cluster"
	ob.Seeing = &seeing
	ob.EquipmentNotes = "25mm EP"
	if err := store.UpdateObservation(ob); err != nil {
		t.Fatalf("UpdateObservation: %v", err)
	}

	obs, err := store.ListObservations(sess.ID)
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}
	if obs[0].Notes != "great cluster" {
		t.Errorf("Notes: got %q", obs[0].Notes)
	}
	if obs[0].Seeing == nil || *obs[0].Seeing != 5 {
		t.Errorf("Seeing: got %v", obs[0].Seeing)
	}
}

func TestNilRatings(t *testing.T) {
	store := openMemStore(t)

	sess := &Session{StartedAt: time.Now().UTC()}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	ob := &Observation{
		SessionID:  sess.ID,
		Timestamp:  time.Now().UTC(),
		TargetName: "M13",
		RAHours:    16.695,
		DecDegrees: 36.46,
		AutoLogged: true,
		// Seeing and Transparency intentionally nil.
	}
	if err := store.AddObservation(ob); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	obs, err := store.ListObservations(sess.ID)
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if obs[0].Seeing != nil {
		t.Errorf("Seeing should be nil, got %v", obs[0].Seeing)
	}
	if obs[0].Transparency != nil {
		t.Errorf("Transparency should be nil, got %v", obs[0].Transparency)
	}
}
