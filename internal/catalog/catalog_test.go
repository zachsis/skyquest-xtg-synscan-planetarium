package catalog

import (
	"math"
	"testing"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
)

func testCatalog(t *testing.T) *Catalog {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Latitude = 34.0
	cfg.Longitude = -118.0
	cfg.LocationConfigured = true
	svc := astro.NewAstroService(cfg)
	cat, err := NewCatalog(svc)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

func TestCatalogLoads(t *testing.T) {
	cat := testCatalog(t)
	if cat.Len() < 14000 {
		t.Errorf("catalog has only %d stars, expected >= 14000", cat.Len())
	}
}

func TestKnownStars(t *testing.T) {
	cat := testCatalog(t)

	tests := []struct {
		name   string
		wantRA float64 // approximate, hours
		wantDec float64 // approximate, degrees
	}{
		{"Sirius", 6.75, -16.72},
		{"Vega", 18.62, 38.78},
		{"Polaris", 2.53, 89.26},
		{"Betelgeuse", 5.92, 7.41},
	}

	for _, tc := range tests {
		results := cat.SearchByName(tc.name)
		if len(results) == 0 {
			t.Errorf("SearchByName(%q) returned no results", tc.name)
			continue
		}
		found := false
		for _, s := range results {
			if math.Abs(s.RA-tc.wantRA) < 0.01 && math.Abs(s.Dec-tc.wantDec) < 0.1 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SearchByName(%q): no match near RA=%.2f Dec=%.1f; got %v",
				tc.name, tc.wantRA, tc.wantDec, results)
		}
	}
}

func TestStarsInRadius(t *testing.T) {
	cat := testCatalog(t)
	// Search around Sirius (RA~6.75h, Dec~-16.72) with 1 degree radius.
	results := cat.StarsInRadius(6.75, -16.72, 1.0)
	if len(results) == 0 {
		t.Fatal("StarsInRadius returned no stars near Sirius")
	}
	// Sirius itself should be in the results.
	found := false
	for _, s := range results {
		if s.Name == "Sirius" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Sirius not found in StarsInRadius results")
	}
}

func TestStarsInRadiusWraparound(t *testing.T) {
	cat := testCatalog(t)
	// Search near RA=0.1h (close to 0h/24h boundary) with 5 degree radius.
	// Should pick up stars near both RA=23.9h and RA=0.3h.
	results := cat.StarsInRadius(0.1, 15.0, 5.0)
	// There should be at least some stars in this region.
	if len(results) == 0 {
		t.Error("StarsInRadius near RA=0h returned no results; wrap-around may be broken")
	}
}

func TestStarsBrighterThan(t *testing.T) {
	cat := testCatalog(t)
	bright := cat.StarsBrighterThan(1.0)
	if len(bright) < 10 {
		t.Errorf("StarsBrighterThan(1.0) returned %d stars, expected >= 10", len(bright))
	}
	for _, s := range bright {
		if s.Mag > 1.0 {
			t.Errorf("star %q has mag %.2f > 1.0", s.Name, s.Mag)
		}
	}
}

func TestSearchByName(t *testing.T) {
	cat := testCatalog(t)

	// Exact match.
	results := cat.SearchByName("Sirius")
	if len(results) == 0 {
		t.Error("SearchByName(Sirius) returned no results")
	}

	// Case insensitive.
	results = cat.SearchByName("sirius")
	if len(results) == 0 {
		t.Error("SearchByName(sirius) returned no results (case-insensitive)")
	}

	// Empty query.
	results = cat.SearchByName("")
	if len(results) != 0 {
		t.Errorf("SearchByName(\"\") returned %d results, expected 0", len(results))
	}
}

func TestStarsAboveHorizon(t *testing.T) {
	cat := testCatalog(t)
	// Use a fixed time to get deterministic results.
	fixedTime := time.Date(2024, 6, 15, 3, 0, 0, 0, time.UTC)
	above := cat.StarsAboveHorizon(fixedTime)
	if len(above) < 1000 {
		t.Errorf("StarsAboveHorizon returned %d stars, expected >= 1000", len(above))
	}
	if len(above) > cat.Len() {
		t.Errorf("StarsAboveHorizon returned %d stars, more than total %d", len(above), cat.Len())
	}
}

func TestStarByHipID(t *testing.T) {
	cat := testCatalog(t)
	// Sirius HIP ID = 32349
	s, ok := cat.StarByHipID(32349)
	if !ok {
		t.Fatal("StarByHipID(32349) not found")
	}
	if s.Name != "Sirius" {
		t.Errorf("HIP 32349: got name %q, want Sirius", s.Name)
	}
}

func TestAngularDistance(t *testing.T) {
	// Same point should give 0.
	d := angularDistance(6.0, 45.0, 6.0, 45.0)
	if d > 1e-10 {
		t.Errorf("same point: got %g, want 0", d)
	}
	// Poles are 180 degrees apart.
	d = angularDistance(0, 90, 0, -90)
	if math.Abs(d-180) > 0.01 {
		t.Errorf("poles: got %g, want 180", d)
	}
}
