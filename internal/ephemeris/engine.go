// Package ephemeris computes real-time apparent positions for the Sun, Moon,
// and eight planets using the VSOP87B planetary theory via meeus/v3.
// It implements catalog.DynamicObjectProvider so objects can be registered
// with the catalog registry and queried like any static catalog entry.
package ephemeris

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	pp "github.com/soniakeys/meeus/v3/planetposition"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ephemeris/data"
)

const updateInterval = 60 * time.Second

// EphemerisEngine computes solar system positions and exposes them via the
// DynamicObjectProvider interface. It runs a background goroutine that
// refreshes positions every 60 seconds.
//
// The zero value is not usable; create via NewEphemerisEngine.
type EphemerisEngine struct {
	astroSvc *astro.AstroService

	// earth is the VSOP87B Earth planet, used for Sun and planet calculations.
	earth *pp.V87Planet

	// planets holds V87Planets for the 7 non-Earth planets in the same order
	// as planetsMetaList(): Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune.
	planets [7]*pp.V87Planet

	// tmpDir is cleaned up on Stop().
	tmpDir string

	// current stores the most recently computed snapshot as an atomic pointer
	// to avoid lock contention between the update loop and UI reads.
	current atomic.Pointer[[]PlanetInfo]

	cancel context.CancelFunc
}

// NewEphemerisEngine constructs and initialises the engine.
// It extracts the embedded VSOP87B data to a temporary directory, loads all
// V87Planet objects, performs an initial computation, and stores the result.
//
// Call Start to begin the background update loop.
// Call Stop to release resources when done.
func NewEphemerisEngine(astroSvc *astro.AstroService) (*EphemerisEngine, error) {
	tmpDir, err := os.MkdirTemp("", "vsop87b-*")
	if err != nil {
		return nil, fmt.Errorf("ephemeris: create temp dir: %w", err)
	}

	if err := extractVSOP87B(tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("ephemeris: extract VSOP87B data: %w", err)
	}

	earth, err := pp.LoadPlanetPath(pp.Earth, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("ephemeris: load Earth VSOP87B: %w", err)
	}

	var planets [7]*pp.V87Planet
	for i := range planetsMeta {
		planets[i], err = pp.LoadPlanetPath(planetBodyIndex[i], tmpDir)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, fmt.Errorf(
				"ephemeris: load %s VSOP87B: %w", planetsMeta[i].name, err,
			)
		}
	}

	e := &EphemerisEngine{
		astroSvc: astroSvc,
		earth:    earth,
		planets:  planets,
		tmpDir:   tmpDir,
	}

	// Initial computation so callers get data immediately.
	snapshot := e.compute(time.Now())
	e.current.Store(&snapshot)

	return e, nil
}

// Start launches the 60-second background update loop.
// The loop runs until ctx is cancelled or Stop is called.
func (e *EphemerisEngine) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	go func() {
		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				snapshot := e.compute(t)
				e.current.Store(&snapshot)
				log.Printf("ephemeris: positions updated at %s", t.Format(time.RFC3339))
			}
		}
	}()
}

// Stop cancels the background update loop and removes the temporary VSOP87B
// directory.
func (e *EphemerisEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	if e.tmpDir != "" {
		if err := os.RemoveAll(e.tmpDir); err != nil {
			log.Printf("ephemeris: cleanup temp dir: %v", err)
		}
	}
}

// ForceUpdate recomputes positions immediately (useful for testing or when the
// observer's location changes).
func (e *EphemerisEngine) ForceUpdate() {
	snapshot := e.compute(time.Now())
	e.current.Store(&snapshot)
}

// Objects implements catalog.DynamicObjectProvider.
// Returns one CelestialObject per solar system body:
// Sun, Moon, Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune (9 total).
func (e *EphemerisEngine) Objects() []catalog.CelestialObject {
	snapshot := e.Planets()
	objs := make([]catalog.CelestialObject, len(snapshot))
	for i, info := range snapshot {
		objs[i] = info.CelestialObject
	}
	return objs
}

// Source implements catalog.DynamicObjectProvider.
func (e *EphemerisEngine) Source() string {
	return "solar_system"
}

// Planets returns the full PlanetInfo slice from the most recent computation.
func (e *EphemerisEngine) Planets() []PlanetInfo {
	ptr := e.current.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

// GetPlanetInfo looks up a body by name (case-sensitive).
func (e *EphemerisEngine) GetPlanetInfo(name string) (*PlanetInfo, bool) {
	for _, info := range e.Planets() {
		if info.Name == name {
			cp := info
			return &cp, true
		}
	}
	return nil, false
}

// compute recalculates all positions at time t and returns a new snapshot.
func (e *EphemerisEngine) compute(t time.Time) []PlanetInfo {
	sun := computeSun(e.earth, e.astroSvc, t)
	moon := computeMoon(e.astroSvc, t, sun)

	results := make([]PlanetInfo, 0, 2+len(planetsMeta))
	results = append(results, sun, moon)

	for i, m := range planetsMeta {
		info := computePlanet(e.planets[i], e.earth, m, e.astroSvc, t, sun)
		results = append(results, info)
	}

	return results
}

// extractVSOP87B copies the embedded VSOP87B files into dir.
func extractVSOP87B(dir string) error {
	extensions := []string{"mer", "ven", "ear", "mar", "jup", "sat", "ura", "nep"}
	for _, ext := range extensions {
		name := "VSOP87B." + ext
		src, err := data.FS.Open(name)
		if err != nil {
			return fmt.Errorf("open embedded %s: %w", name, err)
		}
		dst, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			_ = src.Close()
			return fmt.Errorf("create %s: %w", name, err)
		}
		if _, err = io.Copy(dst, src); err != nil {
			_ = src.Close()
			_ = dst.Close()
			return fmt.Errorf("copy %s: %w", name, err)
		}
		_ = src.Close()
		if err = dst.Close(); err != nil {
			return fmt.Errorf("close %s: %w", name, err)
		}
	}
	return nil
}
