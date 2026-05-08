package catalog

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"

	_ "embed"
)

//go:embed ngc_filtered.csv
var ngcDataCSV string

// loadNGCData parses the embedded NGC/IC CSV into CelestialObjects.
func loadNGCData() ([]CelestialObject, error) {
	r := csv.NewReader(strings.NewReader(ngcDataCSV))
	r.Comment = '#'
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse NGC CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("NGC CSV has no data rows")
	}

	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	required := []string{"Name", "Type", "RA", "Dec", "Mag", "Constellation"}
	for _, name := range required {
		if _, ok := colMap[name]; !ok {
			return nil, fmt.Errorf("NGC CSV missing column: %s", name)
		}
	}

	var objects []CelestialObject
	for _, row := range records[1:] {
		obj, err := parseNGCRow(row, colMap)
		if err != nil {
			continue // skip unparseable rows
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

func parseNGCRow(row []string, colMap map[string]int) (CelestialObject, error) {
	get := func(name string) string {
		idx, ok := colMap[name]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	name := get("Name")
	if name == "" {
		return CelestialObject{}, fmt.Errorf("empty name")
	}

	raStr := get("RA")
	decStr := get("Dec")
	if raStr == "" || decStr == "" {
		return CelestialObject{}, fmt.Errorf("missing coordinates")
	}

	ra, err := strconv.ParseFloat(raStr, 64)
	if err != nil {
		return CelestialObject{}, fmt.Errorf("invalid RA: %w", err)
	}
	dec, err := strconv.ParseFloat(decStr, 64)
	if err != nil {
		return CelestialObject{}, fmt.Errorf("invalid Dec: %w", err)
	}

	mag := math.NaN()
	if s := get("Mag"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			mag = v
		}
	}

	angSize := 0.0
	if s := get("MajAx"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			angSize = v
		}
	}

	commonName := get("CommonName")
	objType := mapNGCType(get("Type"))

	obj := CelestialObject{
		Name:          commonName,
		CatalogID:     name,
		Type:          objType,
		RAJ2000:       ra,
		DecJ2000:      dec,
		Magnitude:     mag,
		AngularSize:   angSize,
		Constellation: get("Constellation"),
		Description:   get("Type"),
		CatalogSource: "ngc",
	}

	return obj, nil
}

func mapNGCType(t string) ObjectType {
	switch t {
	case "G", "GPair", "GTrpl", "GGroup":
		return ObjectTypeGalaxy
	case "GCl":
		return ObjectTypeGlobularClust
	case "OCl":
		return ObjectTypeOpenCluster
	case "Cl+N":
		return ObjectTypeOpenCluster
	case "PN":
		return ObjectTypePlanetaryNeb
	case "HII", "Neb", "RfN", "EmN", "SNR":
		return ObjectTypeNebula
	case "*", "**", "*Ass":
		return ObjectTypeStar
	default:
		return ObjectTypeNebula
	}
}
