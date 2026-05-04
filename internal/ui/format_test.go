package ui

import (
	"math"
	"testing"
)

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

func TestParseRA(t *testing.T) {
	const eps = 1e-9
	valid := []struct {
		s    string
		want float64
	}{
		{"0", 0},
		{"6", 6},
		{"5.5833", 5.5833},
		{"05:35:00", 5 + 35.0/60},
		{"05:35:00.0", 5 + 35.0/60},
		{"05:35", 5 + 35.0/60},
		{"00:00:00", 0},
		{"23:59:59", 23 + 59.0/60 + 59.0/3600},
		{"06:00:00.0", 6},
	}
	for _, tc := range valid {
		got, err := ParseRA(tc.s)
		if err != nil {
			t.Errorf("ParseRA(%q) unexpected error: %v", tc.s, err)
			continue
		}
		if math.Abs(got-tc.want) > eps {
			t.Errorf("ParseRA(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}

	invalid := []string{
		"",
		"24",
		"-1",
		"25:00:00",
		"12:60:00",
		"12:00:60",
		"abc",
		"1:2:3:4",
	}
	for _, s := range invalid {
		_, err := ParseRA(s)
		if err == nil {
			t.Errorf("ParseRA(%q) expected error, got nil", s)
		}
	}
}

func TestParseDec(t *testing.T) {
	const eps = 1e-9
	valid := []struct {
		s    string
		want float64
	}{
		{"0", 0},
		{"+0", 0},
		{"-0", 0},
		{"45.5", 45.5},
		{"-45.5", -45.5},
		{"+45:30:00", 45.5},
		{"-01:12:06.8", -(1 + 12.0/60 + 6.8/3600)},
		{"90", 90},
		{"-90", -90},
		{"01:12", 1 + 12.0/60},
		{"-30:15:30", -(30 + 15.0/60 + 30.0/3600)},
	}
	for _, tc := range valid {
		got, err := ParseDec(tc.s)
		if err != nil {
			t.Errorf("ParseDec(%q) unexpected error: %v", tc.s, err)
			continue
		}
		if math.Abs(got-tc.want) > eps {
			t.Errorf("ParseDec(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}

	invalid := []string{
		"",
		"91",
		"-91",
		"90:01:00",
		"12:60:00",
		"12:00:60",
		"abc",
		"1:2:3:4",
	}
	for _, s := range invalid {
		_, err := ParseDec(s)
		if err == nil {
			t.Errorf("ParseDec(%q) expected error, got nil", s)
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
