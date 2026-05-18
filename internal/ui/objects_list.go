package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// column indices for the results table.
const (
	colName = iota
	colType
	colMag
	colConst
	colAlt
	colRiseSet
	numCols
)

// columnWidths specifies the pixel width of each results column.
var columnWidths = [numCols]float32{200, 50, 50, 50, 80, 110}

// columnHeaders holds the display label for each column.
var columnHeaders = [numCols]string{
	"Name/ID", "Type", "Mag", "Const", "Alt", "Rise/Set",
}

// objectsTable wraps widget.Table with sort state and a results slice.
type objectsTable struct {
	table    *widget.Table
	results  []BrowserResult
	onSelect func(r BrowserResult)

	// sort state
	sortField     string
	sortAscending bool
	onSortChange  func(field string, asc bool)
}

func newObjectsTable(onSelect func(r BrowserResult), onSortChange func(string, bool)) *objectsTable {
	ot := &objectsTable{
		sortField:     "name",
		sortAscending: true,
		onSelect:      onSelect,
		onSortChange:  onSortChange,
	}

	ot.table = widget.NewTable(
		func() (int, int) {
			// +1 for header row
			return len(ot.results) + 1, numCols
		},
		func() fyne.CanvasObject {
			// Create a cell template: a label that may be coloured.
			lbl := canvas.NewText("", theme.Color(theme.ColorNameForeground))
			lbl.TextSize = 13
			return lbl
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			lbl := cell.(*canvas.Text)
			if id.Row == 0 {
				// Header row.
				lbl.Text = ot.headerText(id.Col)
				lbl.Color = theme.Color(theme.ColorNameForeground)
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.Refresh()
				return
			}
			idx := id.Row - 1
			if idx < 0 || idx >= len(ot.results) {
				lbl.Text = ""
				lbl.Refresh()
				return
			}
			r := &ot.results[idx]
			text, col := ot.cellContent(id.Col, r)
			lbl.Text = text
			lbl.Color = col
			lbl.TextStyle = fyne.TextStyle{}
			lbl.Refresh()
		},
	)

	// Set column widths.
	for col, w := range columnWidths {
		ot.table.SetColumnWidth(col, w)
	}

	// Row selection.
	ot.table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			// Header click — toggle sort.
			ot.handleHeaderClick(id.Col)
			return
		}
		idx := id.Row - 1
		if idx >= 0 && idx < len(ot.results) && onSelect != nil {
			onSelect(ot.results[idx])
		}
	}

	return ot
}

func (ot *objectsTable) headerText(col int) string {
	label := columnHeaders[col]
	if col == ot.sortColIndex() {
		if ot.sortAscending {
			return label + " ▲"
		}
		return label + " ▼"
	}
	return label
}

func (ot *objectsTable) sortColIndex() int {
	switch ot.sortField {
	case "magnitude":
		return colMag
	case "altitude":
		return colAlt
	case "type":
		return colType
	default:
		return colName
	}
}

func (ot *objectsTable) handleHeaderClick(col int) {
	var field string
	switch col {
	case colName:
		field = "name"
	case colType:
		field = "type"
	case colMag:
		field = "magnitude"
	case colAlt:
		field = "altitude"
	default:
		return // unsortable column
	}
	if ot.sortField == field {
		ot.sortAscending = !ot.sortAscending
	} else {
		ot.sortField = field
		ot.sortAscending = true
	}
	if ot.onSortChange != nil {
		ot.onSortChange(ot.sortField, ot.sortAscending)
	}
}

func (ot *objectsTable) cellContent(col int, r *BrowserResult) (string, color.NRGBA) {
	fg := colorToNRGBA(theme.Color(theme.ColorNameForeground))
	switch col {
	case colName:
		return displayName(&r.Object), fg
	case colType:
		return typeAbbreviation(r.Object.Type), fg
	case colMag:
		if math.IsNaN(r.Object.Magnitude) {
			return "--", fg
		}
		return fmt.Sprintf("%.1f", r.Object.Magnitude), fg
	case colConst:
		return r.Object.Constellation, fg
	case colAlt:
		text, col := formatAlt(r.Alt)
		return text, col
	case colRiseSet:
		return formatRiseSet(r.RiseTime, r.SetTime), fg
	default:
		return "", fg
	}
}

func formatAlt(alt float64) (string, color.NRGBA) {
	var arrow string
	var col color.NRGBA
	if alt > 0 {
		arrow = "^"
	} else {
		arrow = "v"
	}
	switch {
	case alt > 30:
		col = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // green
	case alt > 10:
		col = color.NRGBA{R: 220, G: 180, B: 0, A: 255} // yellow
	case alt > 0:
		col = color.NRGBA{R: 220, G: 60, B: 60, A: 255} // red
	default:
		col = color.NRGBA{R: 120, G: 120, B: 120, A: 255} // grey
	}
	return fmt.Sprintf("%.1f° %s", alt, arrow), col
}

func formatRiseSet(rise, set *time.Time) string {
	if rise == nil || set == nil {
		return "--:--/--:--"
	}
	return fmt.Sprintf("%s/%s", rise.Format("15:04"), set.Format("15:04"))
}

// setResults replaces the table data and refreshes the table.
func (ot *objectsTable) setResults(results []BrowserResult) {
	ot.results = results
	ot.table.Refresh()
}

// widget returns the underlying fyne widget for embedding.
func (ot *objectsTable) widget2() fyne.CanvasObject {
	return ot.table
}

// colorToNRGBA converts a color.Color to color.NRGBA for canvas.Text usage.
func colorToNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// buildSortableHeader creates a horizontal row of clickable header buttons.
// This is an alternative to embedding sort arrows in cells when the table
// header approach is not available. We use it as a label row within the
// same table instead.
func buildSortableHeader(ot *objectsTable) fyne.CanvasObject {
	_ = ot // header is embedded as row 0 in the table itself
	return container.NewWithoutLayout()
}
