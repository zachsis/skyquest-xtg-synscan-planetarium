package panels

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/zhatsis/oriontelescope/internal/config"
)

// SettingsPanel provides the observer settings UI.
type SettingsPanel struct {
	content fyne.CanvasObject
}

// NewSettingsPanel creates a settings panel wired to the given config.
func NewSettingsPanel(cfg *config.Config, onSave func()) *SettingsPanel {
	cfg.RLock()
	nameEntry := widget.NewEntry()
	nameEntry.SetText(cfg.LocationName)
	nameEntry.SetPlaceHolder("e.g. Backyard, Dark Sky Site")

	latEntry := widget.NewEntry()
	latEntry.SetText(fmt.Sprintf("%.6f", cfg.Latitude))
	latEntry.Validator = validateLatitude

	lonEntry := widget.NewEntry()
	lonEntry.SetText(fmt.Sprintf("%.6f", cfg.Longitude))
	lonEntry.Validator = validateLongitude

	elevEntry := widget.NewEntry()
	elevEntry.SetText(fmt.Sprintf("%.1f", cfg.Elevation))
	elevEntry.Validator = validateElevation

	tzEntry := widget.NewEntry()
	tzEntry.SetText(cfg.TimezoneOverride)
	tzEntry.SetPlaceHolder("Auto-detect from system")

	portEntry := widget.NewEntry()
	portEntry.SetText(cfg.SerialPort)
	portEntry.SetPlaceHolder("/dev/cu.usbserial-*")

	baudEntry := widget.NewEntry()
	baudEntry.SetText(fmt.Sprintf("%d", cfg.BaudRate))

	magEntry := widget.NewEntry()
	magEntry.SetText(fmt.Sprintf("%.1f", cfg.MagnitudeCutoff))
	cfg.RUnlock()

	statusLabel := widget.NewLabel("")

	form := widget.NewForm(
		widget.NewFormItem("Location Name", nameEntry),
		widget.NewFormItem("Latitude (deg)", latEntry),
		widget.NewFormItem("Longitude (deg)", lonEntry),
		widget.NewFormItem("Elevation (m)", elevEntry),
		widget.NewFormItem("Timezone", tzEntry),
		widget.NewFormItem("Serial Port", portEntry),
		widget.NewFormItem("Baud Rate", baudEntry),
		widget.NewFormItem("Magnitude Cutoff", magEntry),
	)

	saveBtn := widget.NewButton("Save", func() {
		// Validate all fields.
		if err := validateLatitude(latEntry.Text); err != nil {
			statusLabel.SetText("Latitude: " + err.Error())
			return
		}
		if err := validateLongitude(lonEntry.Text); err != nil {
			statusLabel.SetText("Longitude: " + err.Error())
			return
		}
		if err := validateElevation(elevEntry.Text); err != nil {
			statusLabel.SetText("Elevation: " + err.Error())
			return
		}

		lat, _ := strconv.ParseFloat(latEntry.Text, 64)
		lon, _ := strconv.ParseFloat(lonEntry.Text, 64)
		elev, _ := strconv.ParseFloat(elevEntry.Text, 64)
		baud, _ := strconv.Atoi(baudEntry.Text)
		if baud == 0 {
			baud = 9600
		}
		mag, _ := strconv.ParseFloat(magEntry.Text, 64)
		if mag == 0 {
			mag = 10.0
		}

		cfg.Lock()
		cfg.LocationName = nameEntry.Text
		cfg.Latitude = lat
		cfg.Longitude = lon
		cfg.Elevation = elev
		cfg.TimezoneOverride = tzEntry.Text
		cfg.SerialPort = portEntry.Text
		cfg.BaudRate = baud
		cfg.MagnitudeCutoff = mag
		cfg.LocationConfigured = true
		cfg.Unlock()

		if err := config.Save(cfg); err != nil {
			statusLabel.SetText("Error saving: " + err.Error())
			return
		}
		statusLabel.SetText("Settings saved.")
		if onSave != nil {
			onSave()
		}
	})

	resetBtn := widget.NewButton("Reset to Defaults", func() {
		d := config.DefaultConfig()
		nameEntry.SetText(d.LocationName)
		latEntry.SetText(fmt.Sprintf("%.6f", d.Latitude))
		lonEntry.SetText(fmt.Sprintf("%.6f", d.Longitude))
		elevEntry.SetText(fmt.Sprintf("%.1f", d.Elevation))
		tzEntry.SetText(d.TimezoneOverride)
		portEntry.SetText(d.SerialPort)
		baudEntry.SetText(fmt.Sprintf("%d", d.BaudRate))
		magEntry.SetText(fmt.Sprintf("%.1f", d.MagnitudeCutoff))
		statusLabel.SetText("Reset to defaults. Click Save to persist.")
	})

	p := &SettingsPanel{
		content: container.NewVBox(
			widget.NewLabel("Observer Settings"),
			form,
			container.NewHBox(saveBtn, resetBtn),
			statusLabel,
		),
	}
	return p
}

func (p *SettingsPanel) Content() fyne.CanvasObject { return p.content }
func (p *SettingsPanel) Title() string              { return "Settings" }
func (p *SettingsPanel) Icon() fyne.Resource         { return nil }

// SettingsPanel satisfies ui.Panel via structural typing.

func validateLatitude(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < -90 || v > 90 {
		return fmt.Errorf("must be between -90 and 90")
	}
	return nil
}

func validateLongitude(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < -180 || v > 180 {
		return fmt.Errorf("must be between -180 and 180")
	}
	return nil
}

func validateElevation(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < 0 {
		return fmt.Errorf("must be non-negative")
	}
	return nil
}
