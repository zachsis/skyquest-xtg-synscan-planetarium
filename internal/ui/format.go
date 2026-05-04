package ui

import (
	"fmt"
	"math"
)

// FormatRA formats decimal hours as HH:MM:SS.s (one decimal place on seconds).
// Example: 5+35/60 → "05:35:00.0"
func FormatRA(hours float64) string {
	totalTenths := int(math.Round(hours * 36000))
	tenth := totalTenths % 10
	totalSec := totalTenths / 10
	ss := totalSec % 60
	totalMin := totalSec / 60
	mm := totalMin % 60
	hh := totalMin / 60
	return fmt.Sprintf("%02d:%02d:%02d.%d", hh, mm, ss, tenth)
}

// FormatDec formats decimal degrees as ±DD:MM:SS.s (one decimal place on seconds).
// Example: -1.2019 → "-01:12:06.8"
func FormatDec(deg float64) string {
	sign := "+"
	if deg < 0 {
		sign = "-"
		deg = -deg
	}
	totalTenths := int(math.Round(deg * 36000))
	tenth := totalTenths % 10
	totalSec := totalTenths / 10
	ss := totalSec % 60
	totalMin := totalSec / 60
	mm := totalMin % 60
	dd := totalMin / 60
	return fmt.Sprintf("%s%02d:%02d:%02d.%d", sign, dd, mm, ss, tenth)
}

// FormatDeg formats decimal degrees with two decimal places and a degree symbol.
// Example: 34.57 → "34.57°"
func FormatDeg(deg float64) string {
	return fmt.Sprintf("%.2f°", deg)
}
