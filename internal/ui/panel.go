package ui

import "fyne.io/fyne/v2"

// Panel defines the contract for all navigation panels.
type Panel interface {
	Content() fyne.CanvasObject
	Title() string
	Icon() fyne.Resource
}
