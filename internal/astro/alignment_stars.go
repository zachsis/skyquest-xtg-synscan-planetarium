package astro

// AlignmentMode selects how many stars are used for alignment.
type AlignmentMode int

const (
	Align1Star AlignmentMode = 1
	Align2Star AlignmentMode = 2
	Align3Star AlignmentMode = 3
)

// AlignmentStar is one entry in the bright-star alignment catalog.
type AlignmentStar struct {
	Name      string  // Common name
	RA        float64 // Right ascension, decimal hours (J2000)
	Dec       float64 // Declination, decimal degrees (J2000)
	Magnitude float64 // Apparent visual magnitude
}

// BrightAlignmentStars is a ~50-star catalog of bright, well-distributed
// alignment stars suitable for use with GoTo mounts.
// Coordinates are J2000 epoch.
var BrightAlignmentStars = []AlignmentStar{
	// Brightest stars
	{"Sirius", 6.7525, -16.7161, -1.46},
	{"Canopus", 6.3992, -52.6957, -0.74},
	{"Arcturus", 14.2610, +19.1824, -0.05},
	{"Vega", 18.6157, +38.7837, +0.03},
	{"Capella", 5.2782, +45.9980, +0.08},
	{"Rigel", 5.2423, -8.2016, +0.12},
	{"Procyon", 7.6553, +5.2250, +0.34},
	{"Achernar", 1.6286, -57.2367, +0.46},
	{"Betelgeuse", 5.9195, +7.4071, +0.58},
	{"Acrux", 12.4433, -63.0991, +0.77},
	{"Altair", 19.8464, +8.8683, +0.76},
	{"Aldebaran", 4.5987, +16.5093, +0.85},
	{"Spica", 13.4199, -11.1613, +0.97},
	{"Antares", 16.4901, -26.4320, +1.09},
	{"Pollux", 7.7553, +28.0262, +1.14},
	{"Fomalhaut", 22.9608, -29.6222, +1.16},
	{"Deneb", 20.6905, +45.2803, +1.25},
	{"Mimosa", 12.7953, -59.6888, +1.25},
	{"Regulus", 10.1395, +11.9672, +1.35},
	{"Castor", 7.5767, +31.8883, +1.58},
	{"Gacrux", 12.5194, -57.1132, +1.59},
	{"Bellatrix", 5.4189, +6.3497, +1.64},
	{"Elnath", 5.4382, +28.6075, +1.65},
	{"Alnilam", 5.6036, -1.2019, +1.70},
	{"Dubhe", 11.0621, +61.7510, +1.79},
	{"Mirfak", 3.4054, +49.8612, +1.79},
	{"Polaris", 2.5302, +89.2641, +1.98},
	{"Hamal", 2.1196, +23.4624, +2.00},
	{"Mizar", 13.3988, +54.9253, +2.04},
	{"Diphda", 0.7265, -17.9866, +2.04},
	{"Alpheratz", 0.1398, +29.0904, +2.06},
	{"Kochab", 14.8451, +74.1555, +2.07},
	{"Menkent", 14.1114, -36.3699, +2.06},
	{"Mirach", 1.1622, +35.6207, +2.06},
	{"Rasalhague", 17.5823, +12.5600, +2.08},
	{"Alkaid", 13.7923, +49.3133, +1.85},
	{"Kaus Australis", 18.4029, -34.3846, +1.85},
	{"Nunki", 18.9211, -26.2967, +2.05},
	{"Almach", 2.0650, +42.3297, +2.17},
	{"Shaula", 17.5601, -37.1038, +1.62},
	{"Sargas", 17.6220, -42.9978, +1.87},
	{"Schedar", 0.6751, +56.5373, +2.23},
	{"Alphecca", 15.5781, +26.7147, +2.23},
	{"Eltanin", 17.9434, +51.4889, +2.23},
	{"Caph", 0.1530, +59.1498, +2.27},
	{"Enif", 21.7364, +9.8750, +2.38},
	{"Sabik", 17.1729, -15.7249, +2.43},
	{"Markab", 23.0793, +15.2053, +2.49},
	{"Unukalhai", 15.7378, +6.4256, +2.65},
	{"Algenib", 0.2206, +15.1836, +2.83},
	{"Zubeneschamali", 15.2834, -9.3829, +2.61},
}
