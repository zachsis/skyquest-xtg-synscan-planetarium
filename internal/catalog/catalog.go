package catalog

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
)

// Catalog is the in-memory star catalog loaded from the embedded HYG CSV.
type Catalog struct {
	stars    []Star
	byName   map[string][]*Star // lowercase name → matching stars
	byHipID  map[int]*Star
	astroSvc *astro.AstroService
}

// NewCatalog parses the embedded HYG CSV and returns a ready-to-query catalog.
func NewCatalog(astroSvc *astro.AstroService) (*Catalog, error) {
	start := time.Now()
	stars, err := parseCSV(hygCSVData)
	if err != nil {
		return nil, fmt.Errorf("parse star catalog: %w", err)
	}
	c := &Catalog{stars: stars, astroSvc: astroSvc}
	c.buildIndices()
	log.Printf("star catalog loaded: %d stars in %v", len(stars), time.Since(start))
	return c, nil
}

// Stars returns the full star slice (read-only by convention).
func (c *Catalog) Stars() []Star { return c.stars }

// Len returns the number of stars in the catalog.
func (c *Catalog) Len() int { return len(c.stars) }

// StarByHipID looks up a star by its Hipparcos catalog ID.
func (c *Catalog) StarByHipID(hipID int) (*Star, bool) {
	s, ok := c.byHipID[hipID]
	return s, ok
}

// StarsInRadius returns all stars within the given angular radius (degrees) of (ra, dec).
// RA is in hours, Dec in degrees. Handles RA wrap-around correctly.
func (c *Catalog) StarsInRadius(ra, dec, radiusDeg float64) []Star {
	var result []Star
	for i := range c.stars {
		d := angularDistance(ra, dec, c.stars[i].RA, c.stars[i].Dec)
		if d <= radiusDeg {
			result = append(result, c.stars[i])
		}
	}
	return result
}

// StarsAboveHorizon returns all stars with altitude > 0 at the given time.
func (c *Catalog) StarsAboveHorizon(t time.Time) []Star {
	var result []Star
	for i := range c.stars {
		alt, _ := c.astroSvc.AltAz(c.stars[i].RA, c.stars[i].Dec, t)
		if alt > 0 {
			result = append(result, c.stars[i])
		}
	}
	return result
}

// SearchByName performs case-insensitive substring matching on star proper names.
func (c *Catalog) SearchByName(query string) []Star {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var result []Star
	for key, stars := range c.byName {
		if strings.Contains(key, q) {
			for _, s := range stars {
				result = append(result, *s)
			}
		}
	}
	return result
}

// StarsBrighterThan returns all stars with apparent magnitude <= mag.
func (c *Catalog) StarsBrighterThan(mag float64) []Star {
	var result []Star
	for i := range c.stars {
		if c.stars[i].Mag <= mag {
			result = append(result, c.stars[i])
		}
	}
	return result
}

func (c *Catalog) buildIndices() {
	c.byName = make(map[string][]*Star)
	c.byHipID = make(map[int]*Star)
	for i := range c.stars {
		s := &c.stars[i]
		if s.HipID > 0 {
			c.byHipID[s.HipID] = s
		}
		if s.Name != "" {
			key := strings.ToLower(s.Name)
			c.byName[key] = append(c.byName[key], s)
		}
	}
}

// buildColumnMap creates a column-name-to-index map from the CSV header.
func buildColumnMap(header []string) (map[string]int, error) {
	required := []string{"hip", "proper", "ra", "dec", "mag", "spect", "con"}
	colMap := make(map[string]int, len(header))
	for i, name := range header {
		colMap[strings.TrimSpace(strings.ToLower(name))] = i
	}
	for _, req := range required {
		if _, ok := colMap[req]; !ok {
			return nil, fmt.Errorf("missing required column: %s", req)
		}
	}
	return colMap, nil
}

func parseCSV(data string) ([]Star, error) {
	r := csv.NewReader(strings.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	colMap, err := buildColumnMap(records[0])
	if err != nil {
		return nil, err
	}

	hipIdx := colMap["hip"]
	nameIdx := colMap["proper"]
	raIdx := colMap["ra"]
	decIdx := colMap["dec"]
	magIdx := colMap["mag"]
	spectIdx := colMap["spect"]
	conIdx := colMap["con"]

	stars := make([]Star, 0, len(records)-1)
	var skipped int
	for _, row := range records[1:] {
		ra, err := strconv.ParseFloat(row[raIdx], 64)
		if err != nil {
			skipped++
			continue
		}
		dec, err := strconv.ParseFloat(row[decIdx], 64)
		if err != nil {
			skipped++
			continue
		}
		mag, err := strconv.ParseFloat(row[magIdx], 64)
		if err != nil {
			skipped++
			continue
		}
		hip, _ := strconv.Atoi(row[hipIdx]) // optional; 0 if missing

		stars = append(stars, Star{
			HipID:         hip,
			Name:          row[nameIdx],
			RA:            ra,
			Dec:           dec,
			Mag:           mag,
			SpectralType:  row[spectIdx],
			Constellation: row[conIdx],
		})
	}
	if skipped > 0 {
		log.Printf("star catalog: skipped %d rows with parse errors", skipped)
	}
	return stars, nil
}

// angularDistance returns the angular separation in degrees between two positions.
// RA is in hours, Dec in degrees.
func angularDistance(ra1, dec1, ra2, dec2 float64) float64 {
	const deg2rad = math.Pi / 180
	// Convert RA from hours to radians.
	r1 := ra1 * 15 * deg2rad
	r2 := ra2 * 15 * deg2rad
	d1 := dec1 * deg2rad
	d2 := dec2 * deg2rad

	cosD := math.Sin(d1)*math.Sin(d2) + math.Cos(d1)*math.Cos(d2)*math.Cos(r1-r2)
	if cosD > 1 {
		cosD = 1
	}
	if cosD < -1 {
		cosD = -1
	}
	return math.Acos(cosD) * 180 / math.Pi
}
