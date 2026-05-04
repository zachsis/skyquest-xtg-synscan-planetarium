package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
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

// ParseRA parses a right ascension string into decimal hours.
// Accepts "HH:MM:SS", "HH:MM:SS.s", "HH:MM", or a bare decimal (e.g. "5.5833").
// Returns an error if the value is malformed or outside [0, 24).
func ParseRA(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("RA is empty")
	}

	parts := strings.Split(s, ":")
	var hours, minutes, seconds float64
	var err error

	switch len(parts) {
	case 1:
		hours, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid RA %q", s)
		}
	case 2:
		hours, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours %q", parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
	case 3:
		hours, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours %q", parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
		seconds, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds %q", parts[2])
		}
	default:
		return 0, fmt.Errorf("invalid RA format %q", s)
	}

	if minutes < 0 || minutes >= 60 {
		return 0, fmt.Errorf("minutes %g out of range [0, 60)", minutes)
	}
	if seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("seconds %g out of range [0, 60)", seconds)
	}

	ra := hours + minutes/60 + seconds/3600
	if ra < 0 || ra >= 24 {
		return 0, fmt.Errorf("RA %g out of range [0, 24)", ra)
	}
	return ra, nil
}

// ParseDec parses a declination string into decimal degrees.
// Accepts "±DD:MM:SS", "±DD:MM:SS.s", "±DD:MM", or a bare decimal (e.g. "-1.2019").
// Returns an error if the value is malformed or outside [-90, +90].
func ParseDec(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("Dec is empty")
	}

	sign := 1.0
	if strings.HasPrefix(s, "-") {
		sign = -1.0
		s = strings.TrimSpace(s[1:])
	} else if strings.HasPrefix(s, "+") {
		s = strings.TrimSpace(s[1:])
	}

	parts := strings.Split(s, ":")
	var degrees, minutes, seconds float64
	var err error

	switch len(parts) {
	case 1:
		degrees, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid Dec %q", s)
		}
	case 2:
		degrees, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid degrees %q", parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
	case 3:
		degrees, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid degrees %q", parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
		seconds, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds %q", parts[2])
		}
	default:
		return 0, fmt.Errorf("invalid Dec format %q", s)
	}

	if minutes < 0 || minutes >= 60 {
		return 0, fmt.Errorf("minutes %g out of range [0, 60)", minutes)
	}
	if seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("seconds %g out of range [0, 60)", seconds)
	}

	dec := sign * (degrees + minutes/60 + seconds/3600)
	if dec < -90 || dec > 90 {
		return 0, fmt.Errorf("Dec %g out of range [-90, +90]", dec)
	}
	return dec, nil
}
