package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/logging"
)

// LogPanel shows the observation log. It implements Panel.
type LogPanel struct {
	content fyne.CanvasObject
	store   logging.LogStore

	// Active session id — highlighted in sessions view.
	activeSessionID int64

	// Registry may be nil if catalog failed to load.
	registry *catalog.CatalogRegistry

	// Window reference needed for dialogs.
	window fyne.Window

	// Views stacked inside the panel's border container.
	stack        *fyne.Container
	sessionsView fyne.CanvasObject
	obsView      fyne.CanvasObject

	// Currently selected session for observations view.
	selectedSession *logging.SessionSummary

	// Data caches.
	sessions []logging.SessionSummary
	obs      []logging.Observation

	// Table widgets — kept so we can call Refresh after data changes.
	sessTable *widget.Table
	obsTable  *widget.Table
}

// NewLogPanel creates the log panel.
func NewLogPanel(
	store logging.LogStore,
	activeSessionID int64,
	registry *catalog.CatalogRegistry,
	window fyne.Window,
) *LogPanel {
	p := &LogPanel{
		store:           store,
		activeSessionID: activeSessionID,
		registry:        registry,
		window:          window,
	}
	p.buildSessionsView()
	p.buildObsView()

	p.stack = container.NewStack(p.sessionsView)
	p.content = p.stack

	// Initial data load.
	p.refreshSessions()

	return p
}

func (p *LogPanel) Content() fyne.CanvasObject { return p.content }
func (p *LogPanel) Title() string              { return "Log" }
func (p *LogPanel) Icon() fyne.Resource        { return theme.ListIcon() }

// --- Sessions view ---

func (p *LogPanel) buildSessionsView() {
	cols := []struct {
		header string
		width  float32
	}{
		{"Date", 110},
		{"Time", 80},
		{"Location", 160},
		{"Obs.", 60},
		{"Duration", 90},
	}

	tbl := widget.NewTable(
		func() (int, int) { return len(p.sessions), len(cols) },
		func() fyne.CanvasObject {
			return widget.NewLabel("                    ")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row >= len(p.sessions) {
				lbl.SetText("")
				return
			}
			ss := p.sessions[id.Row]
			switch id.Col {
			case 0:
				lbl.SetText(ss.StartedAt.Local().Format("2006-01-02"))
			case 1:
				lbl.SetText(ss.StartedAt.Local().Format("15:04"))
			case 2:
				lbl.SetText(ss.LocationName)
			case 3:
				lbl.SetText(fmt.Sprintf("%d", ss.ObservationCount))
			case 4:
				lbl.SetText(sessionDuration(ss))
			}
			// Highlight the currently active session.
			if ss.ID == p.activeSessionID {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				lbl.TextStyle = fyne.TextStyle{}
			}
			lbl.Refresh()
		},
	)

	for i, c := range cols {
		tbl.SetColumnWidth(i, c.width)
	}

	tbl.OnSelected = func(id widget.TableCellID) {
		if id.Row >= len(p.sessions) {
			return
		}
		ss := p.sessions[id.Row]
		p.showObsView(&ss)
		tbl.UnselectAll()
	}

	p.sessTable = tbl

	exportAllBtn := widget.NewButton("Export All", func() {
		p.exportAll()
	})

	header := buildTableHeader([]string{
		"Date", "Time", "Location", "Obs.", "Duration",
	}, []float32{110, 80, 160, 60, 90})

	p.sessionsView = container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		container.NewHBox(widget.NewLabel(""), exportAllBtn),
		nil, nil,
		tbl,
	)
}

// --- Observations view ---

func (p *LogPanel) buildObsView() {
	cols := []struct {
		header string
		width  float32
	}{
		{"Time", 80},
		{"Target", 180},
		{"Seeing", 70},
		{"Trans.", 70},
		{"Notes", 220},
	}

	tbl := widget.NewTable(
		func() (int, int) { return len(p.obs), len(cols) },
		func() fyne.CanvasObject {
			return widget.NewLabel("                    ")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row >= len(p.obs) {
				lbl.SetText("")
				return
			}
			o := p.obs[id.Row]
			switch id.Col {
			case 0:
				lbl.SetText(o.Timestamp.Local().Format("15:04"))
			case 1:
				name := o.TargetName
				if o.CatalogID != "" && o.CatalogID != o.TargetName {
					name = fmt.Sprintf("%s (%s)", o.TargetName, o.CatalogID)
				}
				lbl.SetText(name)
			case 2:
				lbl.SetText(ratingStr(o.Seeing))
			case 3:
				lbl.SetText(ratingStr(o.Transparency))
			case 4:
				lbl.SetText(o.Notes)
			}
		},
	)

	for i, c := range cols {
		tbl.SetColumnWidth(i, c.width)
	}

	tbl.OnSelected = func(id widget.TableCellID) {
		if id.Row >= len(p.obs) {
			return
		}
		ob := p.obs[id.Row]
		p.openEditDialog(&ob)
		tbl.UnselectAll()
	}

	p.obsTable = tbl

	// Buttons — built once; the Add/Back/Export buttons do not need the
	// selected session at construction time because they capture *p.
	backBtn := widget.NewButton("Back", func() {
		p.refreshSessions()
		p.stack.Objects = []fyne.CanvasObject{p.sessionsView}
		p.stack.Refresh()
	})
	addBtn := widget.NewButton("Add Entry", func() {
		p.openManualEntryDialog()
	})
	exportBtn := widget.NewButton("Export Session", func() {
		if p.selectedSession != nil {
			p.exportSession(p.selectedSession.ID)
		}
	})

	header := buildTableHeader(
		[]string{"Time", "Target", "Seeing", "Trans.", "Notes"},
		[]float32{80, 180, 70, 70, 220},
	)

	p.obsView = container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		container.NewHBox(backBtn, addBtn, exportBtn),
		nil, nil,
		tbl,
	)
}

// --- Navigation helpers ---

func (p *LogPanel) showObsView(ss *logging.SessionSummary) {
	p.selectedSession = ss
	p.refreshObs(ss.ID)
	p.stack.Objects = []fyne.CanvasObject{p.obsView}
	p.stack.Refresh()
}

func (p *LogPanel) refreshSessions() {
	sessions, err := p.store.ListSessions()
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}
	p.sessions = sessions
	if p.sessTable != nil {
		p.sessTable.Refresh()
	}
}

func (p *LogPanel) refreshObs(sessionID int64) {
	obs, err := p.store.ListObservations(sessionID)
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}
	p.obs = obs
	if p.obsTable != nil {
		p.obsTable.Refresh()
	}
}

// --- Export ---

func (p *LogPanel) exportAll() {
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer w.Close()
		if err := p.store.ExportCSV(w, nil); err != nil {
			dialog.ShowError(err, p.window)
		}
	}, p.window)
}

func (p *LogPanel) exportSession(id int64) {
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer w.Close()
		if err := p.store.ExportCSV(w, &id); err != nil {
			dialog.ShowError(err, p.window)
		}
	}, p.window)
}

// --- Helpers ---

func sessionDuration(ss logging.SessionSummary) string {
	if ss.EndedAt == nil {
		return "active"
	}
	d := ss.EndedAt.Sub(ss.StartedAt).Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func ratingStr(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d/5", *v)
}

// buildTableHeader returns a horizontal container of fixed-width labels
// matching the column headers of a widget.Table.
func buildTableHeader(labels []string, widths []float32) fyne.CanvasObject {
	hbox := container.NewHBox()
	for i, lbl := range labels {
		l := widget.NewLabelWithStyle(
			lbl, fyne.TextAlignLeading, fyne.TextStyle{Bold: true},
		)
		w := float32(0)
		if i < len(widths) {
			w = widths[i]
		}
		hbox.Add(container.NewGridWrap(fyne.NewSize(w, 24), l))
	}
	return hbox
}
