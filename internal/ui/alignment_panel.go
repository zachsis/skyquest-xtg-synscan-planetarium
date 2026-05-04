package ui

import (
	"fmt"
	"sort"
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

// starWithAltAz pairs an alignment star with its current alt/az for display.
type starWithAltAz struct {
	star astro.AlignmentStar
	alt  float64
	az   float64
}

// AlignmentPanel is the multi-step alignment wizard panel.
type AlignmentPanel struct {
	pos      telescope.PositionProvider
	svc      slew.GoToService
	astroSvc *astro.AstroService
	cfg      *config.Config

	mu           sync.Mutex
	connected    bool
	mode         astro.AlignmentMode
	starIdx      int           // how many stars have been selected so far
	selected     []astro.AlignmentStar
	visibleStars []starWithAltAz
	pickedIdx    int // index into visibleStars, -1 if none
	slewUnsub    func()

	// Step containers (stored as fields so callbacks can cross-reference them).
	steps       *fyne.Container // container.NewMax — holds the active step
	step0       *fyne.Container
	step1       *fyne.Container
	step2       *fyne.Container
	step3       *fyne.Container
	step4       *fyne.Container

	// Shared widgets.
	alignedStatusLabel *widget.Label

	// Step 1.
	modeRadio    *widget.RadioGroup
	step1NextBtn *widget.Button

	// Step 2.
	step2Header  *widget.Label
	sepHintLabel *widget.Label
	starList     *widget.List
	step2NextBtn *widget.Button

	// Step 3.
	step3Header      *widget.Label
	step3RAVal       *widget.Label
	step3DecVal      *widget.Label
	step3AltVal      *widget.Label
	step3AzVal       *widget.Label
	step3SlewBtn     *widget.Button
	step3ProgressBar *widget.ProgressBarInfinite
	step3ProgressMsg *widget.Label
	step3CenterBtn   *widget.Button
	step3ErrLabel    *widget.Label

	// Step 4.
	completeSummary *widget.Label
	alignedConfirm  *widget.Label

	content fyne.CanvasObject
}

// NewAlignmentPanel creates the alignment wizard panel.
func NewAlignmentPanel(
	pos telescope.PositionProvider,
	svc slew.GoToService,
	astroSvc *astro.AstroService,
	cfg *config.Config,
) *AlignmentPanel {
	p := &AlignmentPanel{
		pos:       pos,
		svc:       svc,
		astroSvc:  astroSvc,
		cfg:       cfg,
		pickedIdx: -1,
	}
	p.buildWidgets()
	p.buildStepContainers()
	p.wireCallbacks()

	p.content = p.steps
	pos.Subscribe(p.onStateUpdate)
	p.onStateUpdate(pos.CurrentState())
	return p
}

// buildWidgets creates every widget with nil callbacks.
func (p *AlignmentPanel) buildWidgets() {
	p.alignedStatusLabel = widget.NewLabel("Not Aligned")

	// Step 1.
	p.modeRadio = widget.NewRadioGroup([]string{
		"1-Star — basic, good for rough GoTo accuracy",
		"2-Star — standard, good GoTo accuracy",
		"3-Star — best GoTo accuracy",
	}, nil)
	p.step1NextBtn = widget.NewButton("Next", nil)
	p.step1NextBtn.Disable()

	// Step 2.
	p.step2Header = widget.NewLabel("")
	p.step2Header.TextStyle = fyne.TextStyle{Bold: true}
	p.sepHintLabel = widget.NewLabel("")
	p.step2NextBtn = widget.NewButton("Next", nil)
	p.step2NextBtn.Disable()

	p.starList = widget.NewList(
		func() int { return len(p.visibleStars) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			s := p.visibleStars[id]
			obj.(*widget.Label).SetText(fmt.Sprintf(
				"%-18s  %+.2f mag  Alt %.1f°  Az %.1f°",
				s.star.Name, s.star.Magnitude, s.alt, s.az,
			))
		},
	)
	p.starList.OnSelected = func(id widget.ListItemID) {
		p.mu.Lock()
		p.pickedIdx = id
		p.mu.Unlock()
		p.step2NextBtn.Enable()
	}
	p.starList.OnUnselected = func(_ widget.ListItemID) {
		p.mu.Lock()
		p.pickedIdx = -1
		p.mu.Unlock()
		p.step2NextBtn.Disable()
	}

	// Step 3.
	p.step3Header = widget.NewLabel("")
	p.step3Header.TextStyle = fyne.TextStyle{Bold: true}
	p.step3RAVal = widget.NewLabel("—")
	p.step3DecVal = widget.NewLabel("—")
	p.step3AltVal = widget.NewLabel("—")
	p.step3AzVal = widget.NewLabel("—")
	p.step3ProgressBar = widget.NewProgressBarInfinite()
	p.step3ProgressBar.Hide()
	p.step3ProgressMsg = widget.NewLabel("")
	p.step3SlewBtn = widget.NewButton("Auto-Slew", nil)
	p.step3CenterBtn = widget.NewButton("Star Centered", nil)
	p.step3CenterBtn.Importance = widget.HighImportance
	p.step3ErrLabel = widget.NewLabel("")

	// Step 4.
	p.completeSummary = widget.NewLabel("")
	p.alignedConfirm = widget.NewLabel("")
}

// buildStepContainers assembles the step layout containers.
func (p *AlignmentPanel) buildStepContainers() {
	p.step0 = container.NewVBox(
		widget.NewLabel("Alignment Status"),
		p.alignedStatusLabel,
		widget.NewSeparator(),
		widget.NewButton("Start Alignment", nil), // callback set in wireCallbacks
		widget.NewLabel("Note: alignment is cleared when the telescope is powered off."),
	)

	p.step1 = container.NewVBox(
		widget.NewLabel("Select Alignment Mode"),
		widget.NewSeparator(),
		p.modeRadio,
		container.NewHBox(
			widget.NewButton("Back", nil), // wired below
			p.step1NextBtn,
		),
	)

	p.step2 = container.NewVBox(
		p.step2Header,
		p.sepHintLabel,
		container.NewMax(container.NewVScroll(p.starList)),
		container.NewHBox(
			widget.NewButton("Back", nil), // wired below
			p.step2NextBtn,
		),
	)

	starForm := widget.NewForm(
		widget.NewFormItem("RA", p.step3RAVal),
		widget.NewFormItem("Dec", p.step3DecVal),
		widget.NewFormItem("Alt", p.step3AltVal),
		widget.NewFormItem("Az", p.step3AzVal),
	)
	p.step3 = container.NewVBox(
		p.step3Header,
		starForm,
		widget.NewSeparator(),
		p.step3SlewBtn,
		p.step3ProgressBar,
		p.step3ProgressMsg,
		widget.NewSeparator(),
		widget.NewLabel("Center the star using the hand controller, then press:"),
		p.step3CenterBtn,
		p.step3ErrLabel,
		widget.NewButton("Back", nil), // wired below
	)

	p.step4 = container.NewVBox(
		widget.NewLabel("Alignment Complete"),
		widget.NewSeparator(),
		p.completeSummary,
		p.alignedConfirm,
		widget.NewSeparator(),
		widget.NewButton("Done", nil), // wired below
	)

	p.steps = container.NewMax(p.step0)
}

// wireCallbacks sets all button OnTapped callbacks now that all containers exist.
func (p *AlignmentPanel) wireCallbacks() {
	// Helper to find a button by label in a VBox.
	findBtn := func(c *fyne.Container, label string) *widget.Button {
		for _, obj := range c.Objects {
			if b, ok := obj.(*widget.Button); ok && b.Text == label {
				return b
			}
			if inner, ok := obj.(*fyne.Container); ok {
				for _, sub := range inner.Objects {
					if b, ok := sub.(*widget.Button); ok && b.Text == label {
						return b
					}
				}
			}
		}
		return nil
	}

	// Step 0: Start Alignment.
	findBtn(p.step0, "Start Alignment").OnTapped = func() {
		if !p.pos.CurrentState().Connected {
			p.alignedStatusLabel.SetText("Not connected — connect via the Status panel.")
			return
		}
		p.cfg.RLock()
		locConfigured := p.cfg.LocationConfigured
		p.cfg.RUnlock()
		if !locConfigured {
			p.alignedStatusLabel.SetText("Configure observer location in Settings first.")
			return
		}
		p.resetWizard()
		p.modeRadio.SetSelected("")
		p.step1NextBtn.Disable()
		p.showStep(p.step1)
	}

	// Step 1: Back and Next.
	findBtn(p.step1, "Back").OnTapped = func() { p.showStep(p.step0) }
	p.modeRadio.OnChanged = func(s string) {
		if s != "" {
			p.step1NextBtn.Enable()
		} else {
			p.step1NextBtn.Disable()
		}
	}
	p.step1NextBtn.OnTapped = func() {
		switch p.modeRadio.Selected {
		case "1-Star — basic, good for rough GoTo accuracy":
			p.mode = astro.Align1Star
		case "2-Star — standard, good GoTo accuracy":
			p.mode = astro.Align2Star
		default:
			p.mode = astro.Align3Star
		}
		p.mu.Lock()
		p.starIdx = 0
		p.selected = nil
		p.mu.Unlock()
		p.enterStarSelection()
	}

	// Step 2: Back and Next.
	findBtn(p.step2, "Back").OnTapped = func() {
		p.resetWizard()
		p.showStep(p.step1)
	}
	p.step2NextBtn.OnTapped = func() {
		p.mu.Lock()
		idx := p.pickedIdx
		p.mu.Unlock()
		if idx < 0 || idx >= len(p.visibleStars) {
			return
		}
		star := p.visibleStars[idx].star
		altaz := p.visibleStars[idx]
		p.mu.Lock()
		p.selected = append(p.selected, star)
		p.starIdx = len(p.selected)
		p.mu.Unlock()
		p.enterCentering(star, altaz.alt, altaz.az)
	}

	// Step 3: Back.
	findBtn(p.step3, "Back").OnTapped = func() {
		p.stopSlewSubscription()
		p.resetWizard()
		p.showStep(p.step1)
	}

	// Step 4: Done.
	findBtn(p.step4, "Done").OnTapped = func() {
		p.resetWizard()
		p.showStep(p.step0)
	}
}

func (p *AlignmentPanel) showStep(c *fyne.Container) {
	p.steps.Objects = []fyne.CanvasObject{c}
	p.steps.Refresh()
}

func (p *AlignmentPanel) resetWizard() {
	p.mu.Lock()
	p.starIdx = 0
	p.selected = nil
	p.pickedIdx = -1
	p.mu.Unlock()
}

func (p *AlignmentPanel) enterStarSelection() {
	p.mu.Lock()
	idx := p.starIdx
	mode := p.mode
	selectedSoFar := make([]astro.AlignmentStar, len(p.selected))
	copy(selectedSoFar, p.selected)
	p.mu.Unlock()

	p.step2Header.SetText(fmt.Sprintf("Select Alignment Star %d of %d", idx+1, int(mode)))
	if int(mode) > 1 && idx > 0 {
		p.sepHintLabel.SetText("For best results, choose a star ≥90° in azimuth from previous selections.")
	} else {
		p.sepHintLabel.SetText("")
	}

	now := time.Now()
	excludeNames := make(map[string]bool)
	for _, s := range selectedSoFar {
		excludeNames[s.Name] = true
	}
	var visible []starWithAltAz
	for _, s := range astro.BrightAlignmentStars {
		if excludeNames[s.Name] {
			continue
		}
		alt, az := p.astroSvc.AltAz(s.RA, s.Dec, now)
		if alt >= 10 {
			visible = append(visible, starWithAltAz{star: s, alt: alt, az: az})
		}
	}
	sort.Slice(visible, func(i, j int) bool {
		return visible[i].star.Magnitude < visible[j].star.Magnitude
	})

	p.mu.Lock()
	p.visibleStars = visible
	p.pickedIdx = -1
	p.mu.Unlock()

	p.starList.Refresh()
	p.step2NextBtn.Disable()
	p.showStep(p.step2)
}

func (p *AlignmentPanel) enterCentering(star astro.AlignmentStar, alt, az float64) {
	p.mu.Lock()
	idx := p.starIdx
	mode := p.mode
	p.mu.Unlock()

	p.step3Header.SetText(fmt.Sprintf("Center %s in the eyepiece (%d of %d)", star.Name, idx, int(mode)))
	p.step3RAVal.SetText(FormatRA(star.RA))
	p.step3DecVal.SetText(FormatDec(star.Dec))
	p.step3AltVal.SetText(FormatDeg(alt))
	p.step3AzVal.SetText(FormatDeg(az))
	p.step3ErrLabel.SetText("")
	p.step3ProgressMsg.SetText("")
	p.step3ProgressBar.Hide()
	p.step3SlewBtn.Enable()

	// Subscribe to slew events for this centering step.
	p.stopSlewSubscription()
	unsub := p.svc.OnSlew(func(e slew.SlewEvent) {
		switch e.Type {
		case slew.SlewSlewing:
			p.step3ProgressBar.Show()
			p.step3ProgressMsg.SetText(fmt.Sprintf("Slewing to %s...", star.Name))
			p.step3SlewBtn.Disable()
		case slew.SlewComplete:
			p.step3ProgressBar.Hide()
			p.step3ProgressMsg.SetText("Slew complete — now center the star")
			p.step3SlewBtn.Enable()
		case slew.SlewCancelled, slew.SlewFailed:
			p.step3ProgressBar.Hide()
			p.step3ProgressMsg.SetText("")
			p.step3SlewBtn.Enable()
		}
	})
	p.mu.Lock()
	p.slewUnsub = unsub
	p.mu.Unlock()

	p.step3SlewBtn.OnTapped = func() {
		if err := p.svc.SlewToCoordinates(star.RA, star.Dec); err != nil {
			p.step3ErrLabel.SetText("Slew error: " + err.Error())
		}
	}

	p.step3CenterBtn.OnTapped = func() {
		p.stopSlewSubscription()
		if err := p.svc.SyncPosition(star.RA, star.Dec); err != nil {
			p.step3ErrLabel.SetText("Sync failed: " + err.Error())
			return
		}
		p.mu.Lock()
		nextIdx := p.starIdx
		curMode := p.mode
		p.mu.Unlock()
		if nextIdx < int(curMode) {
			// More stars to align.
			p.enterStarSelection()
		} else {
			p.showCompletionStep()
		}
	}

	p.showStep(p.step3)
}

func (p *AlignmentPanel) showCompletionStep() {
	p.mu.Lock()
	selected := make([]astro.AlignmentStar, len(p.selected))
	copy(selected, p.selected)
	p.mu.Unlock()

	summary := "Stars aligned:\n"
	for _, s := range selected {
		summary += fmt.Sprintf("  • %s  RA %s  Dec %s\n", s.Name, FormatRA(s.RA), FormatDec(s.Dec))
	}
	p.completeSummary.SetText(summary)

	state := p.pos.CurrentState()
	if state.IsAligned {
		p.alignedConfirm.SetText("Controller confirms alignment ✓")
	} else {
		p.alignedConfirm.SetText("Alignment status not yet confirmed by controller.")
	}
	p.showStep(p.step4)
}

func (p *AlignmentPanel) stopSlewSubscription() {
	p.mu.Lock()
	unsub := p.slewUnsub
	p.slewUnsub = nil
	p.mu.Unlock()
	if unsub != nil {
		unsub()
	}
}

func (p *AlignmentPanel) onStateUpdate(s telescope.TelescopeState) {
	p.mu.Lock()
	wasConnected := p.connected
	p.connected = s.Connected
	p.mu.Unlock()

	if s.IsAligned {
		p.alignedStatusLabel.SetText("Aligned ✓")
	} else if !s.Connected {
		p.alignedStatusLabel.SetText("Not Aligned (disconnected)")
	} else {
		p.alignedStatusLabel.SetText("Not Aligned")
	}

	// If connection dropped mid-wizard, reset to resting state.
	if wasConnected && !s.Connected {
		p.stopSlewSubscription()
		p.resetWizard()
		p.showStep(p.step0)
	}
}

// Content implements Panel.
func (p *AlignmentPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *AlignmentPanel) Title() string { return "Alignment" }

// Icon implements Panel.
func (p *AlignmentPanel) Icon() fyne.Resource { return theme.VisibilityIcon() }
