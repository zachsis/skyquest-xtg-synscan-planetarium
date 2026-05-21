package ui

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/logging"
)

// ratingOptions is the ordered set of radio-group options for Seeing /
// Transparency ratings.
var ratingOptions = []string{"Not rated", "1", "2", "3", "4", "5"}

// openEditDialog opens a modal dialog for editing an existing observation.
func (p *LogPanel) openEditDialog(ob *logging.Observation) {
	// Target label (read-only for auto-logged, editable for manual).
	targetEntry := widget.NewEntry()
	targetEntry.SetText(ob.TargetName)
	targetEntry.Disable() // always read-only in edit dialog

	raLabel := widget.NewLabel(
		fmt.Sprintf("%.6f h", ob.RAHours),
	)
	decLabel := widget.NewLabel(
		fmt.Sprintf("%.6f °", ob.DecDegrees),
	)

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetText(ob.Notes)
	notesEntry.SetMinRowsVisible(3)

	seeingRadio := widget.NewRadioGroup(ratingOptions, nil)
	seeingRadio.SetSelected(ratingToStr(ob.Seeing))

	transRadio := widget.NewRadioGroup(ratingOptions, nil)
	transRadio.SetSelected(ratingToStr(ob.Transparency))

	equipEntry := widget.NewMultiLineEntry()
	equipEntry.SetText(ob.EquipmentNotes)
	equipEntry.SetMinRowsVisible(2)

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Target", targetEntry),
			widget.NewFormItem("RA", raLabel),
			widget.NewFormItem("Dec", decLabel),
		),
		widget.NewSeparator(),
		widget.NewLabel("Notes"),
		notesEntry,
		widget.NewSeparator(),
		widget.NewLabel("Seeing"),
		seeingRadio,
		widget.NewLabel("Transparency"),
		transRadio,
		widget.NewSeparator(),
		widget.NewLabel("Equipment Notes"),
		equipEntry,
	)

	dialog.ShowCustomConfirm(
		"Edit Observation",
		"Save", "Cancel",
		container.NewVScroll(form),
		func(save bool) {
			if !save {
				return
			}
			ob.Notes = notesEntry.Text
			ob.Seeing = strToRating(seeingRadio.Selected)
			ob.Transparency = strToRating(transRadio.Selected)
			ob.EquipmentNotes = equipEntry.Text

			if err := p.store.UpdateObservation(ob); err != nil {
				dialog.ShowError(err, p.window)
				return
			}
			if p.selectedSession != nil {
				p.refreshObs(p.selectedSession.ID)
			}
		},
		p.window,
	)
}

// openManualEntryDialog opens a dialog to log a manual observation.
func (p *LogPanel) openManualEntryDialog() {
	if p.selectedSession == nil {
		return
	}

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("Target name or catalog ID")

	raEntry := widget.NewEntry()
	raEntry.SetPlaceHolder("decimal hours")

	decEntry := widget.NewEntry()
	decEntry.SetPlaceHolder("decimal degrees")

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetMinRowsVisible(3)

	seeingRadio := widget.NewRadioGroup(ratingOptions, nil)
	seeingRadio.SetSelected("Not rated")

	transRadio := widget.NewRadioGroup(ratingOptions, nil)
	transRadio.SetSelected("Not rated")

	equipEntry := widget.NewMultiLineEntry()
	equipEntry.SetMinRowsVisible(2)

	// When a suggestion is selected, auto-fill RA/Dec.
	var suggestList *widget.List
	var suggestions []*catalog.CelestialObject

	updateSuggestions := func(text string) {
		if p.registry == nil || text == "" {
			suggestions = nil
		} else {
			suggestions = p.registry.SearchByName(text)
			if len(suggestions) > 8 {
				suggestions = suggestions[:8]
			}
		}
		if suggestList != nil {
			suggestList.Refresh()
		}
	}

	suggestList = widget.NewList(
		func() int { return len(suggestions) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(suggestions) {
				return
			}
			o := suggestions[id]
			label := o.DisplayName()
			if o.CatalogID != "" && o.CatalogID != o.Name {
				label = fmt.Sprintf(
					"%s (%s)", o.DisplayName(), o.CatalogID,
				)
			}
			obj.(*widget.Label).SetText(label)
		},
	)
	suggestList.OnSelected = func(id widget.ListItemID) {
		if id >= len(suggestions) {
			return
		}
		o := suggestions[id]
		name := o.DisplayName()
		targetEntry.SetText(name)
		raEntry.SetText(strconv.FormatFloat(o.RAJ2000, 'f', 6, 64))
		decEntry.SetText(strconv.FormatFloat(o.DecJ2000, 'f', 6, 64))
		suggestions = nil
		suggestList.Refresh()
	}

	targetEntry.OnChanged = updateSuggestions

	form := container.NewVBox(
		widget.NewLabel("Target"),
		targetEntry,
		suggestList,
		widget.NewForm(
			widget.NewFormItem("RA (h)", raEntry),
			widget.NewFormItem("Dec (°)", decEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("Notes"),
		notesEntry,
		widget.NewSeparator(),
		widget.NewLabel("Seeing"),
		seeingRadio,
		widget.NewLabel("Transparency"),
		transRadio,
		widget.NewSeparator(),
		widget.NewLabel("Equipment Notes"),
		equipEntry,
	)

	dialog.ShowCustomConfirm(
		"Add Observation",
		"Save", "Cancel",
		container.NewVScroll(form),
		func(save bool) {
			if !save {
				return
			}
			name := targetEntry.Text
			if name == "" {
				dialog.ShowError(
					fmt.Errorf("target name is required"),
					p.window,
				)
				return
			}

			ra, err := strconv.ParseFloat(raEntry.Text, 64)
			if err != nil {
				dialog.ShowError(
					fmt.Errorf("invalid RA: %w", err), p.window,
				)
				return
			}
			dec, err := strconv.ParseFloat(decEntry.Text, 64)
			if err != nil {
				dialog.ShowError(
					fmt.Errorf("invalid Dec: %w", err), p.window,
				)
				return
			}

			ob := &logging.Observation{
				SessionID:      p.selectedSession.ID,
				Timestamp:      time.Now().UTC(),
				TargetName:     name,
				RAHours:        ra,
				DecDegrees:     dec,
				Notes:          notesEntry.Text,
				Seeing:         strToRating(seeingRadio.Selected),
				Transparency:   strToRating(transRadio.Selected),
				EquipmentNotes: equipEntry.Text,
				AutoLogged:     false,
			}

			if err := p.store.AddObservation(ob); err != nil {
				dialog.ShowError(err, p.window)
				return
			}
			p.refreshObs(p.selectedSession.ID)
		},
		p.window,
	)
}

// --- rating helpers ---

func ratingToStr(v *int) string {
	if v == nil {
		return "Not rated"
	}
	return strconv.Itoa(*v)
}

func strToRating(s string) *int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
