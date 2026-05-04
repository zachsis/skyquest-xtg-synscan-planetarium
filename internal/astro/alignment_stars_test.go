package astro

import "testing"

func TestBrightAlignmentStarsCatalog(t *testing.T) {
	seen := make(map[string]bool)
	for i, s := range BrightAlignmentStars {
		if s.Name == "" {
			t.Errorf("star[%d] has empty Name", i)
		}
		if seen[s.Name] {
			t.Errorf("duplicate star name: %q", s.Name)
		}
		seen[s.Name] = true

		if s.RA < 0 || s.RA >= 24 {
			t.Errorf("star %q: RA %g out of range [0, 24)", s.Name, s.RA)
		}
		if s.Dec < -90 || s.Dec > 90 {
			t.Errorf("star %q: Dec %g out of range [-90, 90]", s.Name, s.Dec)
		}
		if s.Magnitude == 0 {
			t.Errorf("star %q: Magnitude is zero (likely missing)", s.Name)
		}
	}

	if len(BrightAlignmentStars) < 40 {
		t.Errorf("catalog has only %d stars, want at least 40", len(BrightAlignmentStars))
	}
}

func TestAlignmentModeConstants(t *testing.T) {
	if int(Align1Star) != 1 {
		t.Errorf("Align1Star = %d, want 1", Align1Star)
	}
	if int(Align2Star) != 2 {
		t.Errorf("Align2Star = %d, want 2", Align2Star)
	}
	if int(Align3Star) != 3 {
		t.Errorf("Align3Star = %d, want 3", Align3Star)
	}
}
