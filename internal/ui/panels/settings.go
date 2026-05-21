package panels

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/server"
)

// SettingsPanel provides the observer settings UI.
type SettingsPanel struct {
	content fyne.CanvasObject

	// Stellarium section state (set via SetStellariumServer).
	stelSrv       *server.StellariumServer
	stelStatusLbl *widget.Label
	stelClientsLbl *widget.Label
	stelPortEntry  *widget.Entry
	stelIntervalEntry *widget.Entry
	stelToggle    *widget.Button
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

	stelPort := cfg.StellariumPort
	stelInterval := cfg.StellariumIntervalMs
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

	// --- Stellarium Server section ---
	stelPortEntry := widget.NewEntry()
	stelPortEntry.SetText(fmt.Sprintf("%d", stelPort))
	stelPortEntry.SetPlaceHolder("10001")

	stelIntervalEntry := widget.NewEntry()
	stelIntervalEntry.SetText(fmt.Sprintf("%d", stelInterval))
	stelIntervalEntry.SetPlaceHolder("500")

	stelStatusLbl := widget.NewLabel("Stopped")
	stelClientsLbl := widget.NewLabel("No clients")

	// The toggle button label and behaviour are set up properly when the
	// server is wired via SetStellariumServer.
	stelToggle := widget.NewButton("Enable", nil)
	stelToggle.Disable() // disabled until a server is wired in

	stelForm := widget.NewForm(
		widget.NewFormItem("Port", stelPortEntry),
		widget.NewFormItem("Broadcast interval (ms)", stelIntervalEntry),
		widget.NewFormItem("Status", stelStatusLbl),
		widget.NewFormItem("Clients", stelClientsLbl),
	)

	p := &SettingsPanel{
		stelStatusLbl:     stelStatusLbl,
		stelClientsLbl:    stelClientsLbl,
		stelPortEntry:     stelPortEntry,
		stelIntervalEntry: stelIntervalEntry,
		stelToggle:        stelToggle,
	}

	p.content = container.NewVBox(
		widget.NewLabel("Observer Settings"),
		form,
		container.NewHBox(saveBtn, resetBtn),
		statusLabel,
		widget.NewSeparator(),
		widget.NewLabel("Stellarium Server"),
		stelForm,
		stelToggle,
	)
	return p
}

// SetStellariumServer wires the given server into the settings panel so the
// enable/disable toggle and status labels work. Safe to call after the panel
// is constructed. cfg is used to persist the enabled state.
func (p *SettingsPanel) SetStellariumServer(
	srv *server.StellariumServer,
	cfg *config.Config,
) {
	if srv == nil {
		return
	}
	p.stelSrv = srv

	// Register for client count changes so the label stays current.
	srv.SetOnClientChange(func(n int) {
		if n == 0 {
			p.stelClientsLbl.SetText("No clients")
		} else {
			p.stelClientsLbl.SetText(fmt.Sprintf("%d client(s) connected", n))
		}
	})

	// Configure the toggle button action.
	p.stelToggle.OnTapped = func() {
		if srv.IsRunning() {
			p.stopStellariumServer(srv, cfg)
		} else {
			p.startStellariumServer(srv, cfg)
		}
	}
	p.stelToggle.Enable()

	// Reflect the current server state.
	if srv.IsRunning() {
		p.setStellariumRunning(srv.Port())
	} else {
		p.setStellariumStopped()
	}
}

func (p *SettingsPanel) startStellariumServer(
	srv *server.StellariumServer,
	cfg *config.Config,
) {
	port, err := strconv.Atoi(p.stelPortEntry.Text)
	if err != nil || port < 1024 || port > 65535 {
		p.stelStatusLbl.SetText("Invalid port (1024-65535)")
		return
	}
	interval, err := strconv.Atoi(p.stelIntervalEntry.Text)
	if err != nil || interval < 100 || interval > 2000 {
		p.stelStatusLbl.SetText("Invalid interval (100-2000 ms)")
		return
	}

	// Apply the port/interval to the config before starting.
	cfg.Lock()
	cfg.StellariumPort = port
	cfg.StellariumIntervalMs = interval
	cfg.StellariumEnabled = true
	cfg.Unlock()
	_ = config.Save(cfg)

	if startErr := srv.Start(); startErr != nil {
		p.stelStatusLbl.SetText("Error: " + startErr.Error())
		return
	}
	p.setStellariumRunning(port)
}

func (p *SettingsPanel) stopStellariumServer(
	srv *server.StellariumServer,
	cfg *config.Config,
) {
	srv.Stop()

	cfg.Lock()
	cfg.StellariumEnabled = false
	cfg.Unlock()
	_ = config.Save(cfg)

	p.setStellariumStopped()
}

func (p *SettingsPanel) setStellariumRunning(port int) {
	p.stelStatusLbl.SetText(fmt.Sprintf("Listening on :%d", port))
	p.stelToggle.SetText("Disable")
	p.stelPortEntry.Disable()
	p.stelIntervalEntry.Disable()
}

func (p *SettingsPanel) setStellariumStopped() {
	p.stelStatusLbl.SetText("Stopped")
	p.stelClientsLbl.SetText("No clients")
	p.stelToggle.SetText("Enable")
	p.stelPortEntry.Enable()
	p.stelIntervalEntry.Enable()
}

func (p *SettingsPanel) Content() fyne.CanvasObject { return p.content }
func (p *SettingsPanel) Title() string              { return "Settings" }
func (p *SettingsPanel) Icon() fyne.Resource        { return nil }

// SettingsPanel satisfies ui.Panel via structural typing.

func validateRange(s string, min, max float64) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < min || v > max {
		return fmt.Errorf("must be between %.0f and %.0f", min, max)
	}
	return nil
}

func validateLatitude(s string) error  { return validateRange(s, -90, 90) }
func validateLongitude(s string) error { return validateRange(s, -180, 180) }
func validateElevation(s string) error { return validateRange(s, 0, 1e9) }
