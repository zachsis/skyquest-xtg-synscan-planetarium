package ui

import "testing"

func TestFormatRA(t *testing.T) {
	tests := []struct {
		hours float64
		want  string
	}{
		{0, "00:00:00.0"},
		{5 + 35.0/60, "05:35:00.0"},                     // 5h35m0s (ticket example)
		{12, "12:00:00.0"},
		{23 + 59.0/60 + 59.9/3600, "23:59:59.9"},
		{6.75, "06:45:00.0"}, // 6h45m
	}
	for _, tc := range tests {
		got := FormatRA(tc.hours)
		if got != tc.want {
			t.Errorf("FormatRA(%v) = %q, want %q", tc.hours, got, tc.want)
		}
	}
}

func TestFormatDec(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "+00:00:00.0"},
		{-1.2019, "-01:12:06.8"}, // ticket example
		{90, "+90:00:00.0"},
		{-90, "-90:00:00.0"},
		{45.5, "+45:30:00.0"},
		{-(30 + 15.0/60 + 30.0/3600), "-30:15:30.0"},
	}
	for _, tc := range tests {
		got := FormatDec(tc.deg)
		if got != tc.want {
			t.Errorf("FormatDec(%v) = %q, want %q", tc.deg, got, tc.want)
		}
	}
}

func TestFormatDeg(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{34.57, "34.57°"},
		{215.83, "215.83°"},
		{0, "0.00°"},
		{360, "360.00°"},
		{-5.5, "-5.50°"},
	}
	for _, tc := range tests {
		got := FormatDeg(tc.deg)
		if got != tc.want {
			t.Errorf("FormatDeg(%v) = %q, want %q", tc.deg, got, tc.want)
		}
	}
}
