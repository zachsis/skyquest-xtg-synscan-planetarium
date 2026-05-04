package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Config holds all persistent application configuration.
type Config struct {
	mu sync.RWMutex `json:"-"`

	// Observer location.
	LocationConfigured bool    `json:"location_configured"`
	LocationName       string  `json:"location_name"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	Elevation          float64 `json:"elevation"`

	// Time.
	TimezoneOverride string `json:"timezone_override"`

	// Serial connection.
	SerialPort string `json:"serial_port"`
	BaudRate   int    `json:"baud_rate"`

	// Filtering.
	MagnitudeCutoff float64 `json:"magnitude_cutoff"`
}

// GetLatitude implements astro.LocationProvider.
func (c *Config) GetLatitude() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Latitude
}

// GetLongitude implements astro.LocationProvider.
func (c *Config) GetLongitude() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Longitude
}

// GetElevation implements astro.LocationProvider.
func (c *Config) GetElevation() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Elevation
}

// Lock acquires the write lock.
func (c *Config) Lock() { c.mu.Lock() }

// Unlock releases the write lock.
func (c *Config) Unlock() { c.mu.Unlock() }

// RLock acquires the read lock.
func (c *Config) RLock() { c.mu.RLock() }

// RUnlock releases the read lock.
func (c *Config) RUnlock() { c.mu.RUnlock() }

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Latitude:        0,
		Longitude:       0,
		Elevation:       0,
		BaudRate:        9600,
		MagnitudeCutoff: 10.0,
	}
}

// ConfigDir returns the platform-appropriate configuration directory path.
func ConfigDir() (string, error) {
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "oriontelescope"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".oriontelescope"), nil
}

// ConfigPath returns the full path to the configuration file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads and parses the configuration file. If the file does not exist,
// a default configuration is returned (first-run behavior).
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	return &cfg, nil
}

// Save persists the configuration to disk as indented JSON.
func Save(cfg *Config) error {
	cfg.RLock()
	defer cfg.RUnlock()

	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("config: create dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

// marshalConfig is a helper for testing that marshals config without locking.
func marshalConfig(cfg *Config) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

// loadFrom reads and parses a config file at a specific path.
// Returns defaults if the file doesn't exist.
func loadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	return &cfg, nil
}

// GetTimezone returns the configured timezone, falling back to the system timezone.
func GetTimezone(cfg *Config) *time.Location {
	cfg.RLock()
	override := cfg.TimezoneOverride
	cfg.RUnlock()

	if override != "" {
		loc, err := time.LoadLocation(override)
		if err == nil {
			return loc
		}
	}
	return time.Now().Location()
}
