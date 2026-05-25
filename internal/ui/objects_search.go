package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// typeFilterOptions maps dropdown option labels to ObjectType slices.
var typeFilterOptions = []struct {
	label string
	types []catalog.ObjectType
}{
	{"All Types", nil},
	{"Stars", []catalog.ObjectType{catalog.ObjectTypeStar, catalog.ObjectTypeDoubleStar}},
	{"Galaxies", []catalog.ObjectType{catalog.ObjectTypeGalaxy}},
	{"Nebulae", []catalog.ObjectType{catalog.ObjectTypeNebula, catalog.ObjectTypeSupernovaRem}},
	{"Open Clusters", []catalog.ObjectType{catalog.ObjectTypeOpenCluster}},
	{"Globular Clusters", []catalog.ObjectType{catalog.ObjectTypeGlobularClust}},
	{"Planetary Nebulae", []catalog.ObjectType{catalog.ObjectTypePlanetaryNeb}},
	{"Planets", []catalog.ObjectType{catalog.ObjectTypePlanet}},
	{"Sun/Moon", []catalog.ObjectType{catalog.ObjectTypeSun, catalog.ObjectTypeMoon}},
}

// catalogFilterOptions maps dropdown labels to CatalogSource strings.
var catalogFilterOptions = []struct {
	label  string
	source string
}{
	{"All Catalogs", ""},
	{"Messier", "messier"},
	{"NGC/IC", "ngc"},
	{"Named Stars", "named_stars"},
	{"Solar System", "solar_system"},
}

// magnitudeFilterOptions maps dropdown labels to MaxMagnitude values.
var magnitudeFilterOptions = []struct {
	label string
	value float64
}{
	{"Any Magnitude", 0},
	{"< 4.0 (naked eye)", 4.0},
	{"< 6.0 (binocular)", 6.0},
	{"< 8.0", 8.0},
	{"< 10.0", 10.0},
	{"< 12.0", 12.0},
}

// searchBar holds the search entry and debounce state.
type searchBar struct {
	entry    *widget.Entry
	debounce *time.Timer
	onChange func(text string)
}

func newSearchBar(onChange func(text string)) *searchBar {
	sb := &searchBar{onChange: onChange}
	sb.entry = widget.NewEntry()
	sb.entry.SetPlaceHolder("Search by name, ID, or constellation...")
	sb.entry.OnChanged = func(text string) {
		if sb.debounce != nil {
			sb.debounce.Stop()
		}
		sb.debounce = time.AfterFunc(200*time.Millisecond, func() {
			onChange(text)
		})
	}
	return sb
}

// objectFilterBar holds all filter widgets and their state.
type objectFilterBar struct {
	typeSelect    *widget.Select
	catalogSelect *widget.Select
	magSelect     *widget.Select
	horizonCheck  *widget.Check
	tonightBtn    *widget.Button

	// current selections (index into option slices)
	typeIdx    int
	catalogIdx int
	magIdx     int
}

func newObjectFilterBar(onChange func()) *objectFilterBar {
	fb := &objectFilterBar{}
	ready := false

	typeLabels := make([]string, len(typeFilterOptions))
	for i, o := range typeFilterOptions {
		typeLabels[i] = o.label
	}
	fb.typeSelect = widget.NewSelect(typeLabels, func(s string) {
		for i, o := range typeFilterOptions {
			if o.label == s {
				fb.typeIdx = i
				break
			}
		}
		if ready {
			onChange()
		}
	})
	fb.typeSelect.SetSelected("All Types")

	catalogLabels := make([]string, len(catalogFilterOptions))
	for i, o := range catalogFilterOptions {
		catalogLabels[i] = o.label
	}
	fb.catalogSelect = widget.NewSelect(catalogLabels, func(s string) {
		for i, o := range catalogFilterOptions {
			if o.label == s {
				fb.catalogIdx = i
				break
			}
		}
		if ready {
			onChange()
		}
	})
	fb.catalogSelect.SetSelected("All Catalogs")

	magLabels := make([]string, len(magnitudeFilterOptions))
	for i, o := range magnitudeFilterOptions {
		magLabels[i] = o.label
	}
	fb.magSelect = widget.NewSelect(magLabels, func(s string) {
		for i, o := range magnitudeFilterOptions {
			if o.label == s {
				fb.magIdx = i
				break
			}
		}
		if ready {
			onChange()
		}
	})
	fb.magSelect.SetSelected("Any Magnitude")

	fb.horizonCheck = widget.NewCheck("Above horizon only", func(_ bool) {
		if ready {
			onChange()
		}
	})

	ready = true
	return fb
}

// types returns the currently selected ObjectType slice (nil = all).
func (fb *objectFilterBar) types() []catalog.ObjectType {
	return typeFilterOptions[fb.typeIdx].types
}

// catalogSource returns the currently selected catalog source string.
func (fb *objectFilterBar) catalogSource() string {
	return catalogFilterOptions[fb.catalogIdx].source
}

// maxMagnitude returns the currently selected maximum magnitude (0 = no filter).
func (fb *objectFilterBar) maxMagnitude() float64 {
	return magnitudeFilterOptions[fb.magIdx].value
}

// aboveHorizon returns whether the above-horizon checkbox is checked.
func (fb *objectFilterBar) aboveHorizon() bool {
	return fb.horizonCheck.Checked
}

// buildToolbarRow1 returns the top filter row (search + type + catalog + mag).
func buildToolbarRow1(
	sb *searchBar,
	fb *objectFilterBar,
) fyne.CanvasObject {
	return container.NewHBox(
		sb.entry,
		widget.NewLabel("Type:"),
		fb.typeSelect,
		widget.NewLabel("Catalog:"),
		fb.catalogSelect,
		widget.NewLabel("Mag:"),
		fb.magSelect,
	)
}

// buildToolbarRow2 returns the second filter row (horizon + tonight).
func buildToolbarRow2(
	fb *objectFilterBar,
	tonightBtn *widget.Button,
) fyne.CanvasObject {
	return container.NewHBox(fb.horizonCheck, tonightBtn)
}
