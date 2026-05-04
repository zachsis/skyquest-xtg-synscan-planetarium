package ui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

// GoToPanel provides coordinate entry and slew control.
// It subscribes to PositionProvider for live position display and drives
// slew operations through GoToService. It does not poll the serial port directly.
type GoToPanel struct {
	pos      telescope.PositionProvider
	svc      slew.GoToService
	astroSvc *astro.AstroService
	cfg      *config.Config

	mu        sync.Mutex
	connected bool
	inSlew    bool
	raValid   bool
	decValid  bool
	parsedRA  float64
	parsedDec float64

	// Current position (live).
	curRAVal  *widget.Label
	curDecVal *widget.Label
	curAltVal *widget.Label
	curAzVal  *widget.Label

	// Target entry.
	raEntry  *widget.Entry
	decEntry *widget.Entry
	raErr    *widget.Label
	decErr   *widget.Label

	// Target preview.
	tgtAltVal *widget.Label
	tgtAzVal  *widget.Label
	horizWarn *widget.Label

	// Action buttons.
	slewBtn   *widget.Button
	cancelBtn *widget.Button

	// Progress.
	progressBar *widget.ProgressBarInfinite
	progressMsg *widget.Label

	content fyne.CanvasObject
}

// NewGoToPanel creates the GoTo/slewing panel.
func NewGoToPanel(
	pos telescope.PositionProvider,
	svc slew.GoToService,
	astroSvc *astro.AstroService,
	cfg *config.Config,
) *GoToPanel {
	p := &GoToPanel{
		pos:         pos,
		svc:         svc,
		astroSvc:    astroSvc,
		cfg:         cfg,
		curRAVal:    widget.NewLabel("—"),
		curDecVal:   widget.NewLabel("—"),
		curAltVal:   widget.NewLabel("—"),
		curAzVal:    widget.NewLabel("—"),
		raErr:       widget.NewLabel(""),
		decErr:      widget.NewLabel(""),
		tgtAltVal:   widget.NewLabel("—"),
		tgtAzVal:    widget.NewLabel("—"),
		horizWarn:   widget.NewLabel(""),
		progressMsg: widget.NewLabel(""),
	}

	p.raEntry = widget.NewEntry()
	p.raEntry.PlaceHolder = "HH:MM:SS"
	p.raEntry.Validator = func(s string) error {
		_, err := ParseRA(s)
		return err
	}
	p.raEntry.OnChanged = func(s string) {
		ra, err := ParseRA(s)
		p.mu.Lock()
		if err != nil {
			p.raValid = false
			p.mu.Unlock()
			p.raErr.SetText(err.Error())
		} else {
			p.raValid = true
			p.parsedRA = ra
			p.mu.Unlock()
			p.raErr.SetText("")
		}
		p.updateAltAzPreview()
		p.updateButtonStates()
	}

	p.decEntry = widget.NewEntry()
	p.decEntry.PlaceHolder = "±DD:MM:SS"
	p.decEntry.Validator = func(s string) error {
		_, err := ParseDec(s)
		return err
	}
	p.decEntry.OnChanged = func(s string) {
		dec, err := ParseDec(s)
		p.mu.Lock()
		if err != nil {
			p.decValid = false
			p.mu.Unlock()
			p.decErr.SetText(err.Error())
		} else {
			p.decValid = true
			p.parsedDec = dec
			p.mu.Unlock()
			p.decErr.SetText("")
		}
		p.updateAltAzPreview()
		p.updateButtonStates()
	}

	p.slewBtn = widget.NewButton("Slew", p.onSlew)
	p.slewBtn.Disable()

	p.cancelBtn = widget.NewButton("Cancel Slew", p.onCancel)
	p.cancelBtn.Disable()

	p.progressBar = widget.NewProgressBarInfinite()
	p.progressBar.Hide()

	// Subscribe to position provider for live current position.
	pos.Subscribe(p.onPositionUpdate)
	// Seed with current state.
	p.onPositionUpdate(pos.CurrentState())

	// Subscribe to slew events.
	svc.OnSlew(p.onSlewEvent)

	// Layout.
	curPosForm := widget.NewForm(
		widget.NewFormItem("RA", p.curRAVal),
		widget.NewFormItem("Dec", p.curDecVal),
		widget.NewFormItem("Alt", p.curAltVal),
		widget.NewFormItem("Az", p.curAzVal),
	)

	tgtForm := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("RA", container.NewVBox(p.raEntry, p.raErr)),
			widget.NewFormItem("Dec", container.NewVBox(p.decEntry, p.decErr)),
		),
		widget.NewForm(
			widget.NewFormItem("Target Alt", p.tgtAltVal),
			widget.NewFormItem("Target Az", p.tgtAzVal),
		),
		p.horizWarn,
	)

	btns := container.NewHBox(p.slewBtn, p.cancelBtn)

	progress := container.NewVBox(p.progressBar, p.progressMsg)

	p.content = container.NewVBox(
		widget.NewLabel("Current Position"),
		curPosForm,
		widget.NewSeparator(),
		widget.NewLabel("Target Coordinates"),
		tgtForm,
		widget.NewSeparator(),
		btns,
		progress,
	)

	return p
}

// Content implements Panel.
func (p *GoToPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *GoToPanel) Title() string { return "GoTo" }

// Icon implements Panel.
func (p *GoToPanel) Icon() fyne.Resource { return theme.NavigateNextIcon() }

// SlewTo initiates a programmatic slew by populating the coordinate fields and
// triggering the slew command. Used by sky chart and object browser.
func (p *GoToPanel) SlewTo(ra, dec float64) {
	p.raEntry.SetText(FormatRA(ra))
	p.decEntry.SetText(FormatDec(dec))
	// Both entries will validate and update parsedRA/parsedDec via OnChanged.
	// Slew immediately if inputs are now valid and we're connected.
	p.mu.Lock()
	ok := p.raValid && p.decValid && p.connected && !p.inSlew
	pRA := p.parsedRA
	pDec := p.parsedDec
	p.mu.Unlock()
	if ok {
		p.doSlew(pRA, pDec)
	}
}

func (p *GoToPanel) onPositionUpdate(s telescope.TelescopeState) {
	p.mu.Lock()
	p.connected = s.Connected
	p.mu.Unlock()

	if s.Connected {
		p.curRAVal.SetText(FormatRA(s.RA))
		p.curDecVal.SetText(FormatDec(s.Dec))
		p.curAltVal.SetText(FormatDeg(s.Alt))
		p.curAzVal.SetText(FormatDeg(s.Az))
	} else {
		p.curRAVal.SetText("—")
		p.curDecVal.SetText("—")
		p.curAltVal.SetText("—")
		p.curAzVal.SetText("—")
	}
	p.updateButtonStates()
}

func (p *GoToPanel) updateAltAzPreview() {
	p.mu.Lock()
	raOK := p.raValid
	decOK := p.decValid
	ra := p.parsedRA
	dec := p.parsedDec
	p.mu.Unlock()

	if !raOK || !decOK {
		p.tgtAltVal.SetText("—")
		p.tgtAzVal.SetText("—")
		p.horizWarn.SetText("")
		return
	}

	p.cfg.RLock()
	locConfigured := p.cfg.LocationConfigured
	p.cfg.RUnlock()

	if !locConfigured {
		p.tgtAltVal.SetText("—")
		p.tgtAzVal.SetText("—")
		p.horizWarn.SetText("Observer location not configured")
		return
	}

	alt, az := p.astroSvc.AltAz(ra, dec, time.Now())
	p.tgtAltVal.SetText(FormatDeg(alt))
	p.tgtAzVal.SetText(FormatDeg(az))

	if alt < 0 {
		p.horizWarn.SetText("Warning: target is below the horizon")
	} else {
		p.horizWarn.SetText("")
	}
}

func (p *GoToPanel) updateButtonStates() {
	p.mu.Lock()
	conn := p.connected
	inSlew := p.inSlew
	raOK := p.raValid
	decOK := p.decValid
	p.mu.Unlock()

	if conn && !inSlew && raOK && decOK {
		p.slewBtn.Enable()
	} else {
		p.slewBtn.Disable()
	}

	if inSlew {
		p.cancelBtn.Enable()
	} else {
		p.cancelBtn.Disable()
	}
}

func (p *GoToPanel) onSlew() {
	p.mu.Lock()
	ra := p.parsedRA
	dec := p.parsedDec
	p.mu.Unlock()
	p.doSlew(ra, dec)
}

func (p *GoToPanel) doSlew(ra, dec float64) {
	if err := p.svc.SlewToCoordinates(ra, dec); err != nil {
		p.progressMsg.SetText("Error: " + err.Error())
		return
	}
	p.mu.Lock()
	p.inSlew = true
	p.mu.Unlock()
	p.updateButtonStates()
}

func (p *GoToPanel) onCancel() {
	if err := p.svc.CancelSlew(); err != nil {
		p.progressMsg.SetText("Cancel error: " + err.Error())
	}
}

func (p *GoToPanel) onSlewEvent(e slew.SlewEvent) {
	switch e.Type {
	case slew.SlewSlewing:
		p.mu.Lock()
		p.inSlew = true
		p.mu.Unlock()
		p.progressBar.Show()
		p.progressMsg.SetText(fmt.Sprintf(
			"Slewing to RA %s  Dec %s ...",
			FormatRA(e.TargetRA), FormatDec(e.TargetDec),
		))

	case slew.SlewComplete, slew.SlewCancelled, slew.SlewFailed:
		p.mu.Lock()
		p.inSlew = false
		p.mu.Unlock()
		p.progressBar.Hide()
		switch e.Type {
		case slew.SlewComplete:
			p.progressMsg.SetText("Slew complete")
			time.AfterFunc(5*time.Second, func() { p.progressMsg.SetText("") })
		case slew.SlewCancelled:
			p.progressMsg.SetText("Slew cancelled")
		case slew.SlewFailed:
			p.progressMsg.SetText("Slew failed")
		}
	}

	p.updateButtonStates()
}
