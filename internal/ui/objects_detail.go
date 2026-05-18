package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
)

// detailPanel shows detailed information about the selected object.
type detailPanel struct {
	content fyne.CanvasObject

	// Labels updated on selection.
	nameLabel    *widget.Label
	typeLabel    *widget.Label
	raLabel      *widget.Label
	decLabel     *widget.Label
	altAzLabel   *widget.Label
	magLabel     *widget.Label
	sizeLabel    *widget.Label
	constLabel   *widget.Label
	riseLabel    *widget.Label
	transitLabel *widget.Label
	setLabel     *widget.Label
	descLabel    *widget.Label

	// Solar system extras.
	solarBox      *fyne.Container
	elongLabel    *widget.Label
	phaseLabel    *widget.Label
	distanceLabel *widget.Label

	// Action buttons.
	gotoBtn  *widget.Button
	chartBtn *widget.Button

	// Current result.
	current *BrowserResult

	// Dependencies.
	slewSvc       slew.GoToService
	window        fyne.Window
	onShowOnChart func(ra, dec float64, catalogID string)
	onSlewStart   func()
	onSlewEnd     func()

	// Slew event unsubscribe.
	unsubSlew func()
}

func newDetailPanel(slewSvc slew.GoToService, window fyne.Window) *detailPanel {
	dp := &detailPanel{
		slewSvc: slewSvc,
		window:  window,
	}

	dp.nameLabel = widget.NewLabel("No object selected")
	dp.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	dp.nameLabel.Wrapping = fyne.TextWrapWord

	dp.typeLabel = widget.NewLabel("")
	dp.raLabel = widget.NewLabel("")
	dp.decLabel = widget.NewLabel("")
	dp.altAzLabel = widget.NewLabel("")
	dp.magLabel = widget.NewLabel("")
	dp.sizeLabel = widget.NewLabel("")
	dp.constLabel = widget.NewLabel("")
	dp.riseLabel = widget.NewLabel("")
	dp.transitLabel = widget.NewLabel("")
	dp.setLabel = widget.NewLabel("")
	dp.descLabel = widget.NewLabel("")
	dp.descLabel.Wrapping = fyne.TextWrapWord

	dp.elongLabel = widget.NewLabel("")
	dp.phaseLabel = widget.NewLabel("")
	dp.distanceLabel = widget.NewLabel("")
	dp.solarBox = container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("Solar System"),
		container.NewGridWithColumns(2,
			widget.NewLabel("Elongation:"), dp.elongLabel,
			widget.NewLabel("Phase:"), dp.phaseLabel,
			widget.NewLabel("Distance:"), dp.distanceLabel,
		),
	)
	dp.solarBox.Hide()

	dp.gotoBtn = widget.NewButton("GoTo", dp.handleGoTo)
	dp.chartBtn = widget.NewButton("Show on Chart", dp.handleShowOnChart)

	infoGrid := container.NewGridWithColumns(2,
		widget.NewLabel("Type:"), dp.typeLabel,
		widget.NewLabel("RA (J2000):"), dp.raLabel,
		widget.NewLabel("Dec (J2000):"), dp.decLabel,
		widget.NewLabel("Alt / Az:"), dp.altAzLabel,
		widget.NewLabel("Magnitude:"), dp.magLabel,
		widget.NewLabel("Angular size:"), dp.sizeLabel,
		widget.NewLabel("Constellation:"), dp.constLabel,
		widget.NewLabel("Rise:"), dp.riseLabel,
		widget.NewLabel("Transit:"), dp.transitLabel,
		widget.NewLabel("Set:"), dp.setLabel,
	)

	actions := container.NewHBox(
		layout.NewSpacer(),
		dp.gotoBtn,
		dp.chartBtn,
		layout.NewSpacer(),
	)

	dp.content = container.NewVBox(
		dp.nameLabel,
		widget.NewSeparator(),
		infoGrid,
		dp.solarBox,
		widget.NewSeparator(),
		dp.descLabel,
		widget.NewSeparator(),
		actions,
	)

	// Subscribe to slew events to update GoTo button text.
	if slewSvc != nil {
		dp.unsubSlew = slewSvc.OnSlew(func(evt slew.SlewEvent) {
			switch evt.Type {
			case slew.SlewSlewing:
				dp.gotoBtn.SetText("Slewing...")
			case slew.SlewComplete, slew.SlewCancelled, slew.SlewFailed:
				dp.gotoBtn.SetText("GoTo")
			}
		})
	}

	return dp
}

// setResult updates the detail panel for the given BrowserResult.
func (dp *detailPanel) setResult(r BrowserResult) {
	dp.current = &r
	obj := &r.Object

	// Header.
	header := displayName(obj)
	if obj.Name != "" && obj.CatalogID != "" && obj.Name != obj.CatalogID {
		header += "  " + obj.CatalogID
	}
	if len(obj.AlternateIDs) > 0 {
		header += "  (" + strings.Join(obj.AlternateIDs, ", ") + ")"
	}
	dp.nameLabel.SetText(header)

	dp.typeLabel.SetText(typeFullName(obj.Type))
	dp.raLabel.SetText(FormatRA(obj.RAJ2000))
	dp.decLabel.SetText(FormatDec(obj.DecJ2000))
	dp.altAzLabel.SetText(fmt.Sprintf("%.2f° / %.2f°", r.Alt, r.Az))

	if math.IsNaN(obj.Magnitude) {
		dp.magLabel.SetText("--")
	} else {
		dp.magLabel.SetText(fmt.Sprintf("%.1f", obj.Magnitude))
	}

	if obj.AngularSize > 0 {
		dp.sizeLabel.SetText(fmt.Sprintf("%.1f'", obj.AngularSize))
	} else {
		dp.sizeLabel.SetText("--")
	}

	dp.constLabel.SetText(catalog.ConstellationName(obj.Constellation))

	dp.riseLabel.SetText(formatTimeOrDashes(r.RiseTime))
	dp.transitLabel.SetText(formatTimeOrDashes(r.TransitTime))
	dp.setLabel.SetText(formatTimeOrDashes(r.SetTime))

	dp.descLabel.SetText(obj.Description)

	// Solar system extras.
	if obj.CatalogSource == "solar_system" {
		dp.elongLabel.SetText(fmt.Sprintf("%.1f°", r.Elongation))
		if r.PhaseName != "" {
			dp.phaseLabel.SetText(fmt.Sprintf("%.0f%% (%s)", r.Phase*100, r.PhaseName))
		} else {
			dp.phaseLabel.SetText(fmt.Sprintf("%.0f%%", r.Phase*100))
		}
		if obj.CatalogID == "Moon" || obj.Name == "Moon" {
			dp.distanceLabel.SetText(fmt.Sprintf("%.0f km", r.Distance))
		} else {
			dp.distanceLabel.SetText(fmt.Sprintf("%.4f AU", r.Distance))
		}
		dp.solarBox.Show()
	} else {
		dp.solarBox.Hide()
	}

	// GoTo button state.
	dp.refreshGoToState()
}

func (dp *detailPanel) refreshGoToState() {
	if dp.slewSvc == nil {
		dp.gotoBtn.Disable()
		return
	}
	state := dp.slewSvc.SlewState()
	if state == slew.SlewSlewing {
		dp.gotoBtn.SetText("Slewing...")
		dp.gotoBtn.Disable()
	} else {
		dp.gotoBtn.SetText("GoTo")
		dp.gotoBtn.Enable()
	}
}

func (dp *detailPanel) handleGoTo() {
	if dp.current == nil || dp.slewSvc == nil {
		return
	}
	obj := &dp.current.Object
	alt := dp.current.Alt

	// Sun safety warning.
	if obj.Type == catalog.ObjectTypeSun {
		dialog.ShowConfirm(
			"Solar Observation Warning",
			"Pointing a telescope at the Sun without a proper solar filter "+
				"will cause permanent eye damage and equipment damage. "+
				"Are you sure you want to slew to the Sun?",
			func(confirmed bool) {
				if confirmed {
					dp.doSlew(obj)
				}
			}, dp.window)
		return
	}

	// Below-horizon confirmation.
	if alt < 0 {
		dialog.ShowConfirm(
			"Object Below Horizon",
			fmt.Sprintf(
				"%s is currently below the horizon (Alt: %.1f°). "+
					"The telescope may not be able to reach this position. Slew anyway?",
				displayName(obj), alt,
			),
			func(confirmed bool) {
				if confirmed {
					dp.doSlew(obj)
				}
			}, dp.window)
		return
	}

	dp.doSlew(obj)
}

func (dp *detailPanel) doSlew(obj *catalog.CelestialObject) {
	if err := dp.slewSvc.SlewToObject(
		displayName(obj), obj.CatalogID, obj.RAJ2000, obj.DecJ2000,
	); err != nil {
		dialog.ShowError(err, dp.window)
		return
	}
	dp.gotoBtn.SetText("Slewing...")
	dp.gotoBtn.Disable()
}

func (dp *detailPanel) handleShowOnChart() {
	if dp.current == nil {
		return
	}
	obj := &dp.current.Object
	if dp.onShowOnChart != nil {
		dp.onShowOnChart(obj.RAJ2000, obj.DecJ2000, obj.CatalogID)
	}
}

func formatTimeOrDashes(t *time.Time) string {
	if t == nil {
		return "--:--"
	}
	return t.Local().Format("15:04")
}
