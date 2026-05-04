package skychart

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

var (
	bgColor       = color.RGBA{10, 10, 30, 255}
	horizonColor  = color.RGBA{40, 50, 70, 255}
	gridColor     = color.RGBA{40, 45, 60, 128}
	cardinalColor = color.RGBA{200, 200, 200, 255}
)

// SpectralColor returns the star display color based on spectral type prefix.
func SpectralColor(spType string) color.RGBA {
	if len(spType) == 0 {
		return color.RGBA{255, 255, 255, 255}
	}
	switch spType[0] {
	case 'O':
		return color.RGBA{155, 176, 255, 255}
	case 'B':
		return color.RGBA{170, 191, 255, 255}
	case 'A':
		return color.RGBA{202, 215, 255, 255}
	case 'F':
		return color.RGBA{248, 247, 255, 255}
	case 'G':
		return color.RGBA{255, 244, 234, 255}
	case 'K':
		return color.RGBA{255, 210, 161, 255}
	case 'M':
		return color.RGBA{255, 204, 111, 255}
	default:
		return color.RGBA{255, 255, 255, 255}
	}
}

// StarRadius returns the display radius in pixels for a given magnitude.
// Brighter stars (lower magnitude) get larger dots.
func StarRadius(mag float64) float64 {
	const minMag, maxMag = -1.5, 7.0
	const maxR, minR = 6.0, 1.0
	t := (mag - minMag) / (maxMag - minMag)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return maxR - t*(maxR-minR)
}

// DrawBackground fills the image with the dark background.
func DrawBackground(img *image.RGBA) {
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
}

// DrawHorizonCircle draws the horizon edge as a circle.
func DrawHorizonCircle(img *image.RGBA, cx, cy, radius float64) {
	drawCircle(img, cx, cy, radius, horizonColor)
}

// DrawGrid draws altitude circles and azimuth lines.
func DrawGrid(img *image.RGBA, cx, cy, baseRadius, zoom float64) {
	effR := baseRadius * zoom

	// Altitude circles at 30° and 60°.
	for _, altDeg := range []float64{30, 60} {
		alt := altDeg * math.Pi / 180
		r := effR * math.Cos(alt) / (1.0 + math.Sin(alt))
		drawCircle(img, cx, cy, r, gridColor)
	}

	// Azimuth lines every 45°.
	for az := 0.0; az < 2*math.Pi; az += math.Pi / 4 {
		// Draw from center to horizon.
		x1, y1 := cx, cy
		hx, hy := StereoProject(0, az, effR)
		x2, y2 := cx+hx, cy+hy
		drawLine(img, x1, y1, x2, y2, gridColor)
	}
}

// DrawStar draws a single star dot at the given screen position.
func DrawStar(img *image.RGBA, sx, sy, radius float64, col color.RGBA) {
	fillCircle(img, sx, sy, radius, col)
}

// drawCircle draws an outline circle using the midpoint algorithm.
func drawCircle(img *image.RGBA, cx, cy, radius float64, col color.RGBA) {
	r := int(math.Round(radius))
	icx := int(math.Round(cx))
	icy := int(math.Round(cy))
	x, y := r, 0
	d := 1 - r
	for x >= y {
		setPixel(img, icx+x, icy+y, col)
		setPixel(img, icx-x, icy+y, col)
		setPixel(img, icx+x, icy-y, col)
		setPixel(img, icx-x, icy-y, col)
		setPixel(img, icx+y, icy+x, col)
		setPixel(img, icx-y, icy+x, col)
		setPixel(img, icx+y, icy-x, col)
		setPixel(img, icx-y, icy-x, col)
		y++
		if d < 0 {
			d += 2*y + 1
		} else {
			x--
			d += 2*(y-x) + 1
		}
	}
}

// fillCircle fills a circle at the given screen position.
func fillCircle(img *image.RGBA, cx, cy, radius float64, col color.RGBA) {
	r := int(math.Ceil(radius))
	icx := int(math.Round(cx))
	icy := int(math.Round(cy))
	r2 := radius * radius
	bounds := img.Bounds()
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if float64(dx*dx+dy*dy) <= r2 {
				px, py := icx+dx, icy+dy
				if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
					img.Set(px, py, col)
				}
			}
		}
	}
}

// drawLine draws a line using Bresenham's algorithm.
func drawLine(img *image.RGBA, x1, y1, x2, y2 float64, col color.RGBA) {
	ix1, iy1 := int(math.Round(x1)), int(math.Round(y1))
	ix2, iy2 := int(math.Round(x2)), int(math.Round(y2))

	dx := abs(ix2 - ix1)
	dy := abs(iy2 - iy1)
	sx, sy := 1, 1
	if ix1 > ix2 {
		sx = -1
	}
	if iy1 > iy2 {
		sy = -1
	}
	err := dx - dy

	for {
		setPixel(img, ix1, iy1, col)
		if ix1 == ix2 && iy1 == iy2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			ix1 += sx
		}
		if e2 < dx {
			err += dx
			iy1 += sy
		}
	}
}

func setPixel(img *image.RGBA, x, y int, col color.RGBA) {
	b := img.Bounds()
	if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
		img.SetRGBA(x, y, col)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
