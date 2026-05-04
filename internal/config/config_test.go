package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaudRate != 9600 {
		t.Errorf("expected baud 9600, got %d", cfg.BaudRate)
	}
	if cfg.MagnitudeCutoff != 10.0 {
		t.Errorf("expected mag cutoff 10.0, got %f", cfg.MagnitudeCutoff)
	}
	if cfg.LocationConfigured {
		t.Error("default should not be location configured")
	}
	if cfg.Latitude != 0 || cfg.Longitude != 0 {
		t.Error("default lat/lon should be 0")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.Latitude = 40.7128
	cfg.Longitude = -74.006
	cfg.Elevation = 10
	cfg.LocationName = "NYC"
	cfg.LocationConfigured = true
	cfg.SerialPort = "/dev/cu.usbserial-1110"

	// Save manually to our temp path.
	data, err := marshalConfig(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back.
	loaded, err := loadFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Latitude != 40.7128 {
		t.Errorf("lat: got %f", loaded.Latitude)
	}
	if loaded.Longitude != -74.006 {
		t.Errorf("lon: got %f", loaded.Longitude)
	}
	if loaded.LocationName != "NYC" {
		t.Errorf("name: got %s", loaded.LocationName)
	}
	if loaded.SerialPort != "/dev/cu.usbserial-1110" {
		t.Errorf("port: got %s", loaded.SerialPort)
	}
	if !loaded.LocationConfigured {
		t.Error("should be location configured")
	}
}

func TestLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	// Should return defaults.
	if cfg.BaudRate != 9600 {
		t.Errorf("expected defaults, got baud %d", cfg.BaudRate)
	}
}

func TestGetTimezoneDefault(t *testing.T) {
	cfg := DefaultConfig()
	loc := GetTimezone(cfg)
	if loc == nil {
		t.Fatal("timezone should not be nil")
	}
}

func TestGetTimezoneOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TimezoneOverride = "America/New_York"
	loc := GetTimezone(cfg)
	if loc.String() != "America/New_York" {
		t.Errorf("expected America/New_York, got %s", loc.String())
	}
}

func TestLocationProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Lock()
	cfg.Latitude = 51.5
	cfg.Longitude = -0.1
	cfg.Elevation = 15
	cfg.Unlock()

	if cfg.GetLatitude() != 51.5 {
		t.Errorf("lat: got %f", cfg.GetLatitude())
	}
	if cfg.GetLongitude() != -0.1 {
		t.Errorf("lon: got %f", cfg.GetLongitude())
	}
	if cfg.GetElevation() != 15 {
		t.Errorf("elev: got %f", cfg.GetElevation())
	}
}
