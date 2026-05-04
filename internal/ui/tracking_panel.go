package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/synscan"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

const (
	trackingModeOff      = "Off"
	trackingModeSidereal = "Sidereal"
)

var modeDescriptions = map[string]string{
	trackingModeOff:      "Tracking disabled. The telescope will not follow objects.",
	trackingModeSidereal: "Tracks at the sidereal rate. Use for stars, galaxies, and most deep-sky objects.",
}

// TrackingPanel provides tracking mode display and control.
type TrackingPanel struct {
	pos  telescope.PositionProvider
	ctrl *synscan.Controller

	mu          sync.Mutex
	currentMode synscan.TrackingMode
	connected   bool

	modeLabel  *widget.Label // bold "Off" / "Sidereal"
	radio      *widget.RadioGroup
	setBtn     *widget.Button
	descLabel  *widget.Label
	msgLabel   *widget.Label
	notConnMsg *widget.Label

	content fyne.CanvasObject
}

// NewTrackingPanel creates the tracking control panel.
// pos is the PositionProvider (StatusPanel); ctrl is the shared SynScan controller.
func NewTrackingPanel(pos telescope.PositionProvider, ctrl *synscan.Controller) *TrackingPanel {
	p := &TrackingPanel{
		pos:        pos,
		ctrl:       ctrl,
		modeLabel:  widget.NewLabel("—"),
		descLabel:  widget.NewLabel(""),
		msgLabel:   widget.NewLabel(""),
		notConnMsg: widget.NewLabel("Not connected — connect via the Status panel."),
	}
	p.modeLabel.TextStyle = fyne.TextStyle{Bold: true}
	p.notConnMsg.TextStyle = fyne.TextStyle{Italic: true}

	p.radio = widget.NewRadioGroup([]string{trackingModeOff, trackingModeSidereal}, p.onRadioChanged)

	p.setBtn = widget.NewButton("Set Tracking", p.onSetTracking)
	p.setBtn.Disable()

	// Seed from current state (may be zero/disconnected on startup).
	initial := pos.CurrentState()
	p.applyState(initial)

	// Subscribe to live updates.
	pos.Subscribe(p.applyState)

	statusBar := container.NewHBox(
		widget.NewLabel("Current Tracking Mode:"),
		p.modeLabel,
	)

	controls := container.NewVBox(
		p.radio,
		p.descLabel,
		p.setBtn,
		p.msgLabel,
	)

	p.content = container.NewVBox(
		statusBar,
		widget.NewSeparator(),
		controls,
		p.notConnMsg,
	)

	return p
}

// Content implements Panel.
func (p *TrackingPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *TrackingPanel) Title() string { return "Tracking" }

// Icon implements Panel.
func (p *TrackingPanel) Icon() fyne.Resource { return theme.MediaPlayIcon() }

// applyState is called on each PositionProvider update. Thread-safe.
func (p *TrackingPanel) applyState(s telescope.TelescopeState) {
	p.mu.Lock()
	prevMode := p.currentMode
	p.currentMode = s.TrackingMode
	p.connected = s.Connected
	p.mu.Unlock()

	modeStr := modeToString(s.TrackingMode)
	p.modeLabel.SetText(modeStr)

	// Sync radio only when the user has no pending selection
	// (i.e. radio currently reflects the previous controller state).
	if p.radio.Selected == modeToString(prevMode) {
		p.radio.SetSelected(modeStr)
	}

	if s.Connected {
		p.notConnMsg.Hide()
	} else {
		p.notConnMsg.Show()
	}

	p.updateButtonState()
}

func (p *TrackingPanel) onRadioChanged(selected string) {
	if desc, ok := modeDescriptions[selected]; ok {
		p.descLabel.SetText(desc)
	}
	p.updateButtonState()
}

func (p *TrackingPanel) updateButtonState() {
	p.mu.Lock()
	currentMode := p.currentMode
	connected := p.connected
	p.mu.Unlock()

	if !connected || p.radio.Selected == modeToString(currentMode) {
		p.setBtn.Disable()
	} else {
		p.setBtn.Enable()
	}
}

func (p *TrackingPanel) onSetTracking() {
	selected := p.radio.Selected
	mode := stringToMode(selected)

	p.setBtn.Disable()

	if err := p.ctrl.SetTrackingMode(mode); err != nil {
		p.msgLabel.SetText("Error: " + err.Error())
		time.AfterFunc(3*time.Second, func() { p.msgLabel.SetText("") })
		p.updateButtonState()
		return
	}

	// Optimistic update: treat the new mode as current immediately so the
	// next state tick doesn't reset the radio back to the old mode.
	p.mu.Lock()
	p.currentMode = mode
	p.mu.Unlock()

	p.modeLabel.SetText(selected)
	p.msgLabel.SetText("Mode set successfully")
	time.AfterFunc(3*time.Second, func() { p.msgLabel.SetText("") })
	p.updateButtonState()
}

func modeToString(m synscan.TrackingMode) string {
	switch m {
	case synscan.TrackingSidereal:
		return trackingModeSidereal
	default:
		return trackingModeOff
	}
}

func stringToMode(s string) synscan.TrackingMode {
	switch s {
	case trackingModeSidereal:
		return synscan.TrackingSidereal
	default:
		return synscan.TrackingOff
	}
}
