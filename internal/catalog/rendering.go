package catalog

// RenderingHint provides drawing hints for each ObjectType on the sky chart.
type RenderingHint struct {
	Shape       string  // "ellipse", "square", "dotted_circle", "circle_cross", "star_dot", "filled_circle", "crescent", "sun"
	DefaultSize float64 // default icon size in logical pixels
	Label       bool    // whether to show a label by default
}

// RenderingHints maps each ObjectType to its rendering hint.
var RenderingHints = map[ObjectType]RenderingHint{
	ObjectTypeGalaxy:        {Shape: "ellipse", DefaultSize: 10, Label: true},
	ObjectTypeNebula:        {Shape: "square", DefaultSize: 8, Label: true},
	ObjectTypePlanetaryNeb:  {Shape: "square", DefaultSize: 8, Label: true},
	ObjectTypeOpenCluster:   {Shape: "dotted_circle", DefaultSize: 10, Label: true},
	ObjectTypeGlobularClust: {Shape: "circle_cross", DefaultSize: 10, Label: true},
	ObjectTypeSupernovaRem:  {Shape: "square", DefaultSize: 8, Label: true},
	ObjectTypeStar:          {Shape: "star_dot", DefaultSize: 4, Label: false},
	ObjectTypeDoubleStar:    {Shape: "star_dot", DefaultSize: 4, Label: false},
	ObjectTypePlanet:        {Shape: "filled_circle", DefaultSize: 12, Label: true},
	ObjectTypeMoon:          {Shape: "crescent", DefaultSize: 14, Label: true},
	ObjectTypeSun:           {Shape: "sun", DefaultSize: 14, Label: true},
}
