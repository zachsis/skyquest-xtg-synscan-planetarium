// Package data embeds the VSOP87B planetary theory data files.
// These files are used by the planetposition package to compute
// high-accuracy heliocentric ecliptic coordinates.
package data

import "embed"

// FS holds all embedded VSOP87B data files.
//
//go:embed VSOP87B.*
var FS embed.FS
