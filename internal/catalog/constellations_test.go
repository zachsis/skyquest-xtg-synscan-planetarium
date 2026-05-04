package catalog

import "testing"

func TestConstellationLineCount(t *testing.T) {
	if len(ConstellationLines) < 600 {
		t.Errorf("ConstellationLines has %d segments, expected >= 600", len(ConstellationLines))
	}
}

func TestAllConstellationsPresent(t *testing.T) {
	conNames := make(map[string]bool)
	for _, line := range ConstellationLines {
		conNames[line.Constellation] = true
	}
	if len(conNames) < 80 {
		t.Errorf("only %d unique constellations, expected >= 80", len(conNames))
	}
}

func TestWellKnownConstellations(t *testing.T) {
	counts := make(map[string]int)
	for _, line := range ConstellationLines {
		counts[line.Constellation]++
	}

	tests := []struct {
		name     string
		minSegs  int
	}{
		{"Ori", 8},
		{"UMa", 6},
		{"Cas", 4},
		{"Sco", 8},
		{"Sgr", 6},
	}
	for _, tc := range tests {
		if counts[tc.name] < tc.minSegs {
			t.Errorf("%s: %d segments, want >= %d", tc.name, counts[tc.name], tc.minSegs)
		}
	}
}

func TestStarsByConstellation(t *testing.T) {
	m := StarsByConstellation(ConstellationLines)
	if len(m) < 80 {
		t.Errorf("StarsByConstellation: %d constellations, want >= 80", len(m))
	}
	// Orion should have at least 7 unique HIP IDs.
	if ori, ok := m["Ori"]; !ok {
		t.Error("Orion (Ori) not in map")
	} else if len(ori) < 7 {
		t.Errorf("Ori has %d unique stars, want >= 7", len(ori))
	}
}

func TestConstellationStarsInCatalog(t *testing.T) {
	cat := testCatalog(t)
	var missing int
	for _, line := range ConstellationLines {
		if _, ok := cat.StarByHipID(line.Hip1); !ok {
			missing++
		}
		if _, ok := cat.StarByHipID(line.Hip2); !ok {
			missing++
		}
	}
	// Allow some misses (faint stars filtered out at mag 7.0), but not many.
	total := len(ConstellationLines) * 2
	pct := float64(missing) / float64(total) * 100
	if pct > 15 {
		t.Errorf("%.1f%% of constellation HIP IDs not in catalog (%d/%d); too many missing", pct, missing, total)
	}
	t.Logf("constellation HIP ID coverage: %.1f%% (%d missing of %d)", 100-pct, missing, total)
}
