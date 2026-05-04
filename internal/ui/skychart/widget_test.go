package skychart

import (
	"image"
	"math"
	"testing"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
)

func testWidget(t *testing.T) *SkyChartWidget {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Latitude = 34.0
	cfg.Longitude = -118.0
	cfg.LocationConfigured = true
	svc := astro.NewAstroService(cfg)
	cat, err := catalog.NewCatalog(svc)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return NewSkyChartWidget(cat, cfg, svc)
}

func TestDrawChartProducesImage(t *testing.T) {
	w := testWidget(t)
	defer w.stopAutoRefresh()

	img := w.drawChart(512, 512)
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatalf("expected *image.RGBA, got %T", img)
	}
	if rgba.Bounds().Dx() != 512 || rgba.Bounds().Dy() != 512 {
		t.Errorf("bounds: got %v, want 512x512", rgba.Bounds())
	}
}

func TestDrawChartRendersStars(t *testing.T) {
	w := testWidget(t)
	defer w.stopAutoRefresh()

	w.drawChart(800, 800)
	if len(w.renderedStars) == 0 {
		t.Error("no stars were rendered")
	}
	if len(w.renderedStars) < 100 {
		t.Errorf("only %d stars rendered, expected many more", len(w.renderedStars))
	}
}

func TestStarRadius(t *testing.T) {
	r := StarRadius(-1.5)
	if math.Abs(r-6.0) > 0.01 {
		t.Errorf("mag -1.5: got radius %g, want 6.0", r)
	}
	r = StarRadius(7.0)
	if math.Abs(r-1.0) > 0.01 {
		t.Errorf("mag 7.0: got radius %g, want 1.0", r)
	}
}
