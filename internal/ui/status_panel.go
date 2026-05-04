package ui

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/serial"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/synscan"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

const pollInterval = 500 * time.Millisecond

// StatusPanel is the live telescope status panel.
// It implements the Panel interface and telescope.PositionProvider so other
// components (sky chart, GoTo panel, Stellarium server) can subscribe to
// live telescope state without polling independently.
type StatusPanel struct {
	cfg      *config.Config
	port     *serial.Port
	ctrl     *synscan.Controller
	astroSvc *astro.AstroService
	pub      *telescope.StatePublisher

	mu         sync.Mutex
	cancelPoll context.CancelFunc

	portSelect  *widget.Select
	connectBtn  *widget.Button
	statusLabel *widget.Label
	raVal       *widget.Label
	decVal      *widget.Label
	altVal      *widget.Label
	azVal       *widget.Label
	trackingVal *widget.Label
	gotoVal     *widget.Label
	alignedVal  *widget.Label

	content fyne.CanvasObject
}

// NewStatusPanel creates the live status panel bound to the given config.
func NewStatusPanel(cfg *config.Config) *StatusPanel {
	p := &StatusPanel{
		cfg:         cfg,
		port:        serial.NewPort(),
		astroSvc:    astro.NewAstroService(cfg),
		pub:         telescope.NewStatePublisher(),
		statusLabel: widget.NewLabel("Disconnected"),
		raVal:       widget.NewLabel("—"),
		decVal:      widget.NewLabel("—"),
		altVal:      widget.NewLabel("—"),
		azVal:       widget.NewLabel("—"),
		trackingVal: widget.NewLabel("—"),
		gotoVal:     widget.NewLabel("—"),
		alignedVal:  widget.NewLabel("—"),
	}
	p.ctrl = synscan.NewController(p.port)

	// Handle unexpected port disconnects (e.g. USB unplugged).
	p.port.OnStatusChange(func(connected bool) {
		if !connected {
			p.handleDisconnected()
		}
	})

	ports, _ := serial.ListPorts()
	p.portSelect = widget.NewSelect(ports, nil)
	p.portSelect.PlaceHolder = "Select port..."

	// Pre-select last-used port from config.
	cfg.RLock()
	lastPort := cfg.SerialPort
	cfg.RUnlock()
	if lastPort != "" {
		p.portSelect.SetSelected(lastPort)
	}

	// Refresh button updates the port list (Fyne Select has no OnOpen callback).
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		pts, _ := serial.ListPorts()
		p.portSelect.Options = pts
		p.portSelect.Refresh()
	})

	p.connectBtn = widget.NewButton("Connect", p.onConnectToggle)

	// Connection bar: port dropdown expands, buttons pinned to the right.
	connBar := container.NewBorder(nil, nil, nil,
		container.NewHBox(refreshBtn, p.connectBtn),
		p.portSelect,
	)

	posForm := widget.NewForm(
		widget.NewFormItem("RA", p.raVal),
		widget.NewFormItem("Dec", p.decVal),
		widget.NewFormItem("Alt", p.altVal),
		widget.NewFormItem("Az", p.azVal),
	)

	stateForm := widget.NewForm(
		widget.NewFormItem("Tracking Mode", p.trackingVal),
		widget.NewFormItem("GoTo Active", p.gotoVal),
		widget.NewFormItem("Aligned", p.alignedVal),
	)

	p.content = container.NewVBox(
		connBar,
		p.statusLabel,
		widget.NewSeparator(),
		widget.NewLabel("Position"),
		posForm,
		widget.NewSeparator(),
		widget.NewLabel("Telescope State"),
		stateForm,
	)

	return p
}

// Content implements Panel.
func (p *StatusPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *StatusPanel) Title() string { return "Status" }

// Icon implements Panel.
func (p *StatusPanel) Icon() fyne.Resource { return theme.InfoIcon() }

// Controller returns the underlying SynScan controller.
// Other panels in the same package use this to share the serial connection.
func (p *StatusPanel) Controller() *synscan.Controller {
	return p.ctrl
}

// CurrentState implements telescope.PositionProvider.
func (p *StatusPanel) CurrentState() telescope.TelescopeState {
	return p.pub.CurrentState()
}

// Subscribe implements telescope.PositionProvider.
func (p *StatusPanel) Subscribe(cb func(telescope.TelescopeState)) func() {
	return p.pub.Subscribe(cb)
}

func (p *StatusPanel) onConnectToggle() {
	if p.port.IsConnected() {
		p.disconnect()
	} else {
		// Refresh port list before presenting the selection to the user.
		pts, _ := serial.ListPorts()
		p.portSelect.Options = pts
		p.portSelect.Refresh()
		p.connect()
	}
}

func (p *StatusPanel) connect() {
	selected := p.portSelect.Selected
	if selected == "" {
		p.statusLabel.SetText("Select a port first")
		return
	}

	p.connectBtn.Disable()

	portCfg := serial.DefaultConfig()
	p.cfg.RLock()
	if p.cfg.BaudRate > 0 {
		portCfg.BaudRate = p.cfg.BaudRate
	}
	p.cfg.RUnlock()

	if err := p.port.Connect(selected, portCfg); err != nil {
		p.statusLabel.SetText("Connect failed: " + err.Error())
		p.connectBtn.Enable()
		return
	}

	// Persist the selected port so it is pre-selected on next launch.
	p.cfg.Lock()
	p.cfg.SerialPort = selected
	p.cfg.Unlock()
	_ = config.Save(p.cfg)

	p.connectBtn.SetText("Disconnect")
	p.connectBtn.Enable()
	p.portSelect.Disable()
	p.statusLabel.SetText("Connected to " + selected)

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.cancelPoll = cancel
	p.mu.Unlock()
	go p.pollLoop(ctx)
}

func (p *StatusPanel) stopPolling() {
	p.mu.Lock()
	cancel := p.cancelPoll
	p.cancelPoll = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (p *StatusPanel) disconnect() {
	p.stopPolling()
	// handleDisconnected will fire via OnStatusChange.
	p.port.Disconnect()
}

// handleDisconnected resets the UI and stops polling.
// Called from OnStatusChange on both user-initiated and unexpected disconnects.
func (p *StatusPanel) handleDisconnected() {
	p.stopPolling()

	p.connectBtn.SetText("Connect")
	p.connectBtn.Enable()
	p.portSelect.Enable()
	p.statusLabel.SetText("Disconnected")
	p.resetValues()
	p.pub.Update(telescope.TelescopeState{Connected: false, LastUpdate: time.Now()})
}

func (p *StatusPanel) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

func (p *StatusPanel) poll() {
	ra, dec, err := p.ctrl.GetRADec()
	if err != nil {
		p.raVal.SetText("Comm error")
		p.decVal.SetText("—")
		p.altVal.SetText("—")
		p.azVal.SetText("—")
		return
	}

	p.cfg.RLock()
	locConfigured := p.cfg.LocationConfigured
	p.cfg.RUnlock()

	var alt, az float64
	if locConfigured {
		alt, az = p.astroSvc.AltAz(ra, dec, time.Now())
	}

	mode, err := p.ctrl.GetTrackingMode()
	if err != nil {
		p.trackingVal.SetText("Comm error")
		return
	}

	inProgress, err := p.ctrl.IsGotoInProgress()
	if err != nil {
		p.gotoVal.SetText("Comm error")
		return
	}

	aligned, err := p.ctrl.IsAligned()
	if err != nil {
		p.alignedVal.SetText("Comm error")
		return
	}

	state := telescope.TelescopeState{
		RA:           ra,
		Dec:          dec,
		Alt:          alt,
		Az:           az,
		TrackingMode: mode,
		IsSlewing:    inProgress,
		IsAligned:    aligned,
		Connected:    true,
		PortName:     p.port.LastConnectedPort(),
		LastUpdate:   time.Now(),
	}
	p.pub.Update(state)

	p.raVal.SetText(FormatRA(ra))
	p.decVal.SetText(FormatDec(dec))

	if locConfigured {
		p.altVal.SetText(FormatDeg(alt))
		p.azVal.SetText(FormatDeg(az))
	} else {
		p.altVal.SetText("—")
		p.azVal.SetText("—")
	}

	switch mode {
	case synscan.TrackingSidereal:
		p.trackingVal.SetText("Sidereal")
	default:
		p.trackingVal.SetText("Off")
	}

	p.gotoVal.SetText(boolYesNo(inProgress))
	p.alignedVal.SetText(boolYesNo(aligned))
}

func boolYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func (p *StatusPanel) resetValues() {
	p.raVal.SetText("—")
	p.decVal.SetText("—")
	p.altVal.SetText("—")
	p.azVal.SetText("—")
	p.trackingVal.SetText("—")
	p.gotoVal.SetText("—")
	p.alignedVal.SetText("—")
}
