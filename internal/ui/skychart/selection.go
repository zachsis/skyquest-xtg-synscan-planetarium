package skychart

import (
	"errors"
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
)

// FindNearestObject returns the closest VisibleObject within hitRadiusPx of
// (px, py). Returns nil if no object is close enough.
func FindNearestObject(visible []VisibleObject, px, py float64) *VisibleObject {
	var best *VisibleObject
	bestDist := math.MaxFloat64
	for i := range visible {
		dx := visible[i].ScreenX - px
		dy := visible[i].ScreenY - py
		d := math.Sqrt(dx*dx + dy*dy)
		if d < bestDist {
			bestDist = d
			best = &visible[i]
		}
	}
	if best != nil && bestDist <= hitRadiusPx {
		return best
	}
	return nil
}

const hitRadiusPx = 15.0

// FindNearestStar returns the closest rendered star within hitRadiusPx of (px, py).
// Returns nil if no star is close enough.
func FindNearestStar(rendered []RenderedStar, px, py float64) *RenderedStar {
	var best *RenderedStar
	bestDist := math.MaxFloat64
	for i := range rendered {
		dx := rendered[i].ScreenX - px
		dy := rendered[i].ScreenY - py
		d := math.Sqrt(dx*dx + dy*dy)
		if d < bestDist {
			bestDist = d
			best = &rendered[i]
		}
	}
	if best != nil && bestDist <= hitRadiusPx {
		return best
	}
	return nil
}

// Tapped implements fyne.Tappable for star and DSO selection.
func (w *SkyChartWidget) Tapped(ev *fyne.PointEvent) {
	px := float64(ev.Position.X)
	py := float64(ev.Position.Y)

	w.mu.Lock()
	dsoOverlay := w.dsoOverlay
	cb := w.onStarClicked
	slewSvc := w.slewService
	astroSvc := w.astroSvc
	w.mu.Unlock()

	// Check DSO overlay first — DSOs are larger targets.
	if dsoOverlay != nil {
		visible := dsoOverlay.VisibleObjects()
		if vo := FindNearestObject(visible, px, py); vo != nil {
			c := fyne.CurrentApp().Driver().CanvasForObject(w)
			if c != nil {
				showObjectPopup(c, ev.Position, vo.Object, astroSvc, slewSvc)
			}
			return
		}
	}

	rs := FindNearestStar(w.renderedStars, px, py)
	if rs == nil {
		return
	}

	if cb != nil {
		cb(rs.Star)
	}

	// Resolve canvas lazily.
	c := fyne.CurrentApp().Driver().CanvasForObject(w)
	if c == nil {
		return
	}

	showStarPopup(c, ev.Position, rs.Star, astroSvc, slewSvc)
}

// showStarPopup displays star details and a GoTo button at the given position.
func showStarPopup(c fyne.Canvas, pos fyne.Position, star catalog.Star, astroSvc *astro.AstroService, slewSvc slew.GoToService) {
	name := star.Name
	if name == "" {
		name = fmt.Sprintf("HIP %d", star.HipID)
	}

	now := time.Now()
	alt, az := astroSvc.AltAz(star.RA, star.Dec, now)

	riseStr, setStr := riseSetStrings(astroSvc, star, now)

	content := container.NewVBox(
		widget.NewLabelWithStyle(name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Magnitude: %.1f", star.Mag)),
		widget.NewLabel(fmt.Sprintf("Spectral: %s", star.SpectralType)),
		widget.NewLabel(fmt.Sprintf("Constellation: %s", star.Constellation)),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("RA: %s", formatRA(star.RA))),
		widget.NewLabel(fmt.Sprintf("Dec: %s", formatDec(star.Dec))),
		widget.NewLabel(fmt.Sprintf("Alt: %.1f°  Az: %.1f°", alt, az)),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Rises: %s", riseStr)),
		widget.NewLabel(fmt.Sprintf("Sets: %s", setStr)),
	)

	var popup *widget.PopUp

	if slewSvc != nil {
		content.Add(widget.NewSeparator())
		content.Add(widget.NewButton("GoTo", func() {
			_ = slewSvc.SlewToObject(star.Name, fmt.Sprintf("HIP%d", star.HipID), star.RA, star.Dec)
			if popup != nil {
				popup.Hide()
			}
		}))
	}

	popup = widget.NewPopUp(content, c)
	popup.ShowAtPosition(pos)
}

// showObjectPopup displays CelestialObject details and a GoTo button.
func showObjectPopup(
	c fyne.Canvas,
	pos fyne.Position,
	obj catalog.CelestialObject,
	astroSvc *astro.AstroService,
	slewSvc slew.GoToService,
) {
	name := obj.DisplayName()

	now := time.Now()
	altDeg, azDeg := astroSvc.AltAz(obj.RAJ2000, obj.DecJ2000, now)

	typeName := objectTypeName(obj.Type)

	content := container.NewVBox(
		widget.NewLabelWithStyle(name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Type: %s", typeName)),
		widget.NewLabel(fmt.Sprintf("Catalog ID: %s", obj.CatalogID)),
	)

	if len(obj.AlternateIDs) > 0 {
		content.Add(widget.NewLabel(fmt.Sprintf("Also known as: %s",
			joinStrings(obj.AlternateIDs, ", "))))
	}

	if obj.HasMagnitude() {
		content.Add(widget.NewLabel(fmt.Sprintf("Magnitude: %.1f", obj.Magnitude)))
	}
	if obj.AngularSize > 0 {
		content.Add(widget.NewLabel(fmt.Sprintf("Angular size: %.1f′", obj.AngularSize)))
	}

	content.Add(widget.NewSeparator())
	content.Add(widget.NewLabel(fmt.Sprintf("RA: %s", formatRA(obj.RAJ2000))))
	content.Add(widget.NewLabel(fmt.Sprintf("Dec: %s", formatDec(obj.DecJ2000))))
	content.Add(widget.NewLabel(fmt.Sprintf("Alt: %.1f°  Az: %.1f°", altDeg, azDeg)))

	if obj.Constellation != "" {
		fullName := catalog.ConstellationName(obj.Constellation)
		content.Add(widget.NewLabel(fmt.Sprintf("Constellation: %s", fullName)))
	}

	if obj.Description != "" {
		content.Add(widget.NewSeparator())
		content.Add(widget.NewLabel(obj.Description))
	}

	// Safety warning for the Sun.
	if obj.Type == catalog.ObjectTypeSun {
		content.Add(widget.NewSeparator())
		warning := widget.NewLabelWithStyle(
			"WARNING: NEVER point telescope at Sun\nwithout a solar filter!",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		)
		content.Add(warning)
	}

	var popup *widget.PopUp

	if slewSvc != nil {
		content.Add(widget.NewSeparator())
		content.Add(widget.NewButton("GoTo", func() {
			_ = slewSvc.SlewToObject(
				obj.Name, obj.CatalogID,
				obj.RAJ2000, obj.DecJ2000,
			)
			if popup != nil {
				popup.Hide()
			}
		}))
	}

	popup = widget.NewPopUp(content, c)
	popup.ShowAtPosition(pos)
}

// objectTypeName returns a human-readable name for an ObjectType.
func objectTypeName(t catalog.ObjectType) string {
	switch t {
	case catalog.ObjectTypeGalaxy:
		return "Galaxy"
	case catalog.ObjectTypeNebula:
		return "Nebula"
	case catalog.ObjectTypePlanetaryNeb:
		return "Planetary Nebula"
	case catalog.ObjectTypeOpenCluster:
		return "Open Cluster"
	case catalog.ObjectTypeGlobularClust:
		return "Globular Cluster"
	case catalog.ObjectTypeSupernovaRem:
		return "Supernova Remnant"
	case catalog.ObjectTypePlanet:
		return "Planet"
	case catalog.ObjectTypeMoon:
		return "Moon"
	case catalog.ObjectTypeSun:
		return "Sun"
	case catalog.ObjectTypeStar:
		return "Star"
	case catalog.ObjectTypeDoubleStar:
		return "Double Star"
	default:
		return string(t)
	}
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func formatRA(hours float64) string {
	totalTenths := int(math.Round(hours * 36000))
	return formatSexagesimalStr(totalTenths, "")
}

func formatDec(deg float64) string {
	prefix := "+"
	if deg < 0 {
		prefix = "-"
		deg = -deg
	}
	return formatSexagesimalStr(int(math.Round(deg*36000)), prefix)
}

func formatSexagesimalStr(totalTenths int, prefix string) string {
	tenth := totalTenths % 10
	totalSec := totalTenths / 10
	ss := totalSec % 60
	totalMin := totalSec / 60
	mm := totalMin % 60
	hh := totalMin / 60
	return fmt.Sprintf("%s%02d:%02d:%02d.%d", prefix, hh, mm, ss, tenth)
}

func riseSetStrings(astroSvc *astro.AstroService, star catalog.Star, now time.Time) (string, string) {
	rise, set, err := astroSvc.RiseSet(star.RA, star.Dec, now)
	if err != nil {
		if errors.Is(err, astro.ErrNeverSets) {
			return "Circumpolar", "Circumpolar"
		}
		return "Never Rises", "Never Rises"
	}
	return rise.Local().Format("15:04"), set.Local().Format("15:04")
}
