package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PlaceholderPanel is a stub panel that displays its name.
type PlaceholderPanel struct {
	title   string
	icon    fyne.Resource
	content fyne.CanvasObject
}

// NewPlaceholderPanel creates a placeholder panel with the given title and icon.
func NewPlaceholderPanel(title string, icon fyne.Resource) *PlaceholderPanel {
	label := widget.NewLabel(fmt.Sprintf("%s Panel — Coming Soon", title))
	label.Alignment = fyne.TextAlignCenter
	return &PlaceholderPanel{
		title:   title,
		icon:    icon,
		content: container.NewCenter(label),
	}
}

func (p *PlaceholderPanel) Content() fyne.CanvasObject { return p.content }
func (p *PlaceholderPanel) Title() string              { return p.title }
func (p *PlaceholderPanel) Icon() fyne.Resource        { return p.icon }
