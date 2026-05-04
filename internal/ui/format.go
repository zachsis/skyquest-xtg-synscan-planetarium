package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// formatSexagesimal formats a value in tenths of arc-seconds as DD:MM:SS.s with an optional prefix.
func formatSexagesimal(totalTenths int, prefix string) string {
	tenth := totalTenths % 10
	totalSec := totalTenths / 10
	ss := totalSec % 60
	totalMin := totalSec / 60
	mm := totalMin % 60
	hh := totalMin / 60
	return fmt.Sprintf("%s%02d:%02d:%02d.%d", prefix, hh, mm, ss, tenth)
}

// FormatRA formats decimal hours as HH:MM:SS.s (one decimal place on seconds).
func FormatRA(hours float64) string {
	return formatSexagesimal(int(math.Round(hours*36000)), "")
}

// FormatDec formats decimal degrees as ±DD:MM:SS.s (one decimal place on seconds).
func FormatDec(deg float64) string {
	prefix := "+"
	if deg < 0 {
		prefix = "-"
		deg = -deg
	}
	return formatSexagesimal(int(math.Round(deg*36000)), prefix)
}

// FormatDeg formats decimal degrees with two decimal places and a degree symbol.
// Example: 34.57 → "34.57°"
func FormatDeg(deg float64) string {
	return fmt.Sprintf("%.2f°", deg)
}

// parseSexagesimal splits a colon-separated string into primary, minutes, seconds.
// Returns the parsed components or an error. The caller is responsible for
// sign handling and final range validation.
func parseSexagesimal(s, label string) (primary, minutes, seconds float64, err error) {
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		primary, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid %s %q", label, s)
		}
	case 2:
		primary, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid %s %q", label, parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
	case 3:
		primary, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid %s %q", label, parts[0])
		}
		minutes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid minutes %q", parts[1])
		}
		seconds, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid seconds %q", parts[2])
		}
	default:
		return 0, 0, 0, fmt.Errorf("invalid %s format %q", label, s)
	}
	if minutes < 0 || minutes >= 60 {
		return 0, 0, 0, fmt.Errorf("minutes %g out of range [0, 60)", minutes)
	}
	if seconds < 0 || seconds >= 60 {
		return 0, 0, 0, fmt.Errorf("seconds %g out of range [0, 60)", seconds)
	}
	return primary, minutes, seconds, nil
}

// ParseRA parses a right ascension string into decimal hours.
// Accepts "HH:MM:SS", "HH:MM:SS.s", "HH:MM", or a bare decimal (e.g. "5.5833").
// Returns an error if the value is malformed or outside [0, 24).
func ParseRA(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("RA is empty")
	}
	hours, minutes, seconds, err := parseSexagesimal(s, "RA")
	if err != nil {
		return 0, err
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
	degrees, minutes, seconds, err := parseSexagesimal(s, "Dec")
	if err != nil {
		return 0, err
	}
	dec := sign * (degrees + minutes/60 + seconds/3600)
	if dec < -90 || dec > 90 {
		return 0, fmt.Errorf("Dec %g out of range [-90, +90]", dec)
	}
	return dec, nil
}
