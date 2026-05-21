package logging

import (
	"log"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
)

// RegisterAutoLog attaches a slew listener that creates an Observation
// whenever a GoTo slew begins (SlewSlewing).  It returns an unsubscribe
// function that should be called when the session ends.
func RegisterAutoLog(
	slewSvc slew.GoToService,
	store LogStore,
	sessionID int64,
) func() {
	return slewSvc.OnSlew(func(e slew.SlewEvent) {
		if e.Type != slew.SlewSlewing {
			return
		}
		o := &Observation{
			SessionID:  sessionID,
			Timestamp:  e.Timestamp,
			TargetName: e.TargetName,
			CatalogID:  e.CatalogID,
			RAHours:    e.TargetRA,
			DecDegrees: e.TargetDec,
			AutoLogged: true,
		}
		if err := store.AddObservation(o); err != nil {
			log.Printf("autolog: failed to record observation: %v", err)
		}
	})
}
