package ephemeris

// defaultMagnitudes maps solar system body names to approximate visual
// magnitudes at mean distance / typical conditions.
var defaultMagnitudes = map[string]float64{
	"Sun":     -26.74,
	"Moon":    -12.7,
	"Mercury": -0.2,
	"Venus":   -4.4,
	"Mars":    -1.0,
	"Jupiter": -2.5,
	"Saturn":  0.5,
	"Uranus":  5.7,
	"Neptune": 7.8,
}
