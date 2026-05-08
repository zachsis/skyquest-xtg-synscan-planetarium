package catalog

import (
	"math"
	"strings"
	"sync"
)

// DynamicObjectProvider allows external systems to inject objects into the registry.
type DynamicObjectProvider interface {
	Objects() []CelestialObject
	Source() string
}

// FilterCriteria specifies how to filter catalog objects.
type FilterCriteria struct {
	NameSubstring string
	Types         []ObjectType
	Constellation string
	MaxMagnitude  float64 // only brighter than this (0 = no filter)
	CatalogSource string  // "" = all
}

// CatalogRegistry holds all static catalog objects and dynamic providers.
type CatalogRegistry struct {
	mu       sync.RWMutex
	objects  []CelestialObject
	dynamic  []DynamicObjectProvider
	byID     map[string]*CelestialObject
	byAltID  map[string]*CelestialObject
}

// NewCatalogRegistry creates an empty registry.
func NewCatalogRegistry() *CatalogRegistry {
	return &CatalogRegistry{
		byID:    make(map[string]*CelestialObject),
		byAltID: make(map[string]*CelestialObject),
	}
}

// LoadAll loads Messier, NGC/IC, and named stars catalogs.
func (r *CatalogRegistry) LoadAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load Messier first (priority for duplicate handling).
	for i := range messierCatalog {
		r.addLocked(&messierCatalog[i])
	}

	// Load NGC/IC, skipping duplicates already covered by Messier.
	ngcObjects, err := loadNGCData()
	if err != nil {
		return err
	}
	for i := range ngcObjects {
		if _, exists := r.byID[ngcObjects[i].CatalogID]; exists {
			continue
		}
		if _, exists := r.byAltID[ngcObjects[i].CatalogID]; exists {
			continue
		}
		r.addLocked(&ngcObjects[i])
	}

	// Load named stars.
	for i := range namedStarsCatalog {
		r.addLocked(&namedStarsCatalog[i])
	}

	return nil
}

func (r *CatalogRegistry) addLocked(obj *CelestialObject) {
	r.objects = append(r.objects, *obj)
	ptr := &r.objects[len(r.objects)-1]
	r.byID[ptr.CatalogID] = ptr
	for _, altID := range ptr.AlternateIDs {
		r.byAltID[altID] = ptr
	}
}

// RegisterDynamic adds a dynamic object provider.
func (r *CatalogRegistry) RegisterDynamic(provider DynamicObjectProvider) {
	r.mu.Lock()
	r.dynamic = append(r.dynamic, provider)
	r.mu.Unlock()
}

// Len returns the number of static objects.
func (r *CatalogRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.objects)
}

// GetByID looks up an object by its primary or alternate catalog ID.
func (r *CatalogRegistry) GetByID(catalogID string) (*CelestialObject, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if obj, ok := r.byID[catalogID]; ok {
		return obj, true
	}
	if obj, ok := r.byAltID[catalogID]; ok {
		return obj, true
	}

	// Check dynamic providers.
	for _, dp := range r.dynamic {
		for _, obj := range dp.Objects() {
			if obj.CatalogID == catalogID {
				cp := obj
				return &cp, true
			}
			for _, altID := range obj.AlternateIDs {
				if altID == catalogID {
					cp := obj
					return &cp, true
				}
			}
		}
	}

	return nil, false
}

// SearchByName searches objects by name or catalog ID substring (case-insensitive).
func (r *CatalogRegistry) SearchByName(substring string) []*CelestialObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lower := strings.ToLower(substring)
	var results []*CelestialObject

	for i := range r.objects {
		if matchesSearch(&r.objects[i], lower) {
			results = append(results, &r.objects[i])
		}
	}

	for _, dp := range r.dynamic {
		for _, obj := range dp.Objects() {
			if matchesSearch(&obj, lower) {
				cp := obj
				results = append(results, &cp)
			}
		}
	}

	return results
}

func matchesSearch(obj *CelestialObject, lower string) bool {
	if strings.Contains(strings.ToLower(obj.Name), lower) {
		return true
	}
	if strings.Contains(strings.ToLower(obj.CatalogID), lower) {
		return true
	}
	for _, altID := range obj.AlternateIDs {
		if strings.Contains(strings.ToLower(altID), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(obj.Constellation), lower) {
		return true
	}
	return false
}

// FilterByType returns all objects of the given type.
func (r *CatalogRegistry) FilterByType(objType ObjectType) []*CelestialObject {
	return r.Filter(FilterCriteria{Types: []ObjectType{objType}})
}

// FilterByConstellation returns all objects in the given constellation.
func (r *CatalogRegistry) FilterByConstellation(iauCode string) []*CelestialObject {
	return r.Filter(FilterCriteria{Constellation: iauCode})
}

// FilterByMagnitude returns all objects brighter than the given magnitude.
func (r *CatalogRegistry) FilterByMagnitude(brighterThan float64) []*CelestialObject {
	return r.Filter(FilterCriteria{MaxMagnitude: brighterThan})
}

// Filter applies multiple criteria (AND logic) and returns matching objects.
func (r *CatalogRegistry) Filter(criteria FilterCriteria) []*CelestialObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*CelestialObject
	lower := strings.ToLower(criteria.NameSubstring)

	for i := range r.objects {
		if matchesCriteria(&r.objects[i], criteria, lower) {
			results = append(results, &r.objects[i])
		}
	}

	for _, dp := range r.dynamic {
		for _, obj := range dp.Objects() {
			if matchesCriteria(&obj, criteria, lower) {
				cp := obj
				results = append(results, &cp)
			}
		}
	}

	return results
}

func matchesCriteria(obj *CelestialObject, c FilterCriteria, lowerName string) bool {
	if lowerName != "" && !matchesSearch(obj, lowerName) {
		return false
	}
	if len(c.Types) > 0 {
		found := false
		for _, t := range c.Types {
			if obj.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if c.Constellation != "" && obj.Constellation != c.Constellation {
		return false
	}
	if c.MaxMagnitude != 0 {
		if math.IsNaN(obj.Magnitude) || obj.Magnitude > c.MaxMagnitude {
			return false
		}
	}
	if c.CatalogSource != "" && obj.CatalogSource != c.CatalogSource {
		return false
	}
	return true
}

// ObjectsInRegion returns objects within the given RA/Dec bounding box.
// Handles RA wrap-around: when raMin > raMax, the region spans 0h.
func (r *CatalogRegistry) ObjectsInRegion(raMin, raMax, decMin, decMax float64, maxMag float64) []*CelestialObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*CelestialObject

	for i := range r.objects {
		if inRegion(&r.objects[i], raMin, raMax, decMin, decMax, maxMag) {
			results = append(results, &r.objects[i])
		}
	}

	for _, dp := range r.dynamic {
		for _, obj := range dp.Objects() {
			if inRegion(&obj, raMin, raMax, decMin, decMax, maxMag) {
				cp := obj
				results = append(results, &cp)
			}
		}
	}

	return results
}

func inRegion(obj *CelestialObject, raMin, raMax, decMin, decMax, maxMag float64) bool {
	if maxMag != 0 && (math.IsNaN(obj.Magnitude) || obj.Magnitude > maxMag) {
		return false
	}
	if obj.DecJ2000 < decMin || obj.DecJ2000 > decMax {
		return false
	}
	// Handle RA wrap-around.
	if raMin <= raMax {
		return obj.RAJ2000 >= raMin && obj.RAJ2000 <= raMax
	}
	// Wraps around 0h: region is [raMin, 24) ∪ [0, raMax].
	return obj.RAJ2000 >= raMin || obj.RAJ2000 <= raMax
}

// AllObjects returns all static objects (for iteration by the browser).
func (r *CatalogRegistry) AllObjects() []CelestialObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]CelestialObject, len(r.objects))
	copy(all, r.objects)

	for _, dp := range r.dynamic {
		all = append(all, dp.Objects()...)
	}

	return all
}
