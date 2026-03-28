package draw

import (
	"fmt"
)

// Draw draws from src to dst with mask in the specified rectangle
// This is the core drawing operation of Plan 9's draw library
func Draw(dst *Image, r Rectangle, src, mask *Image, p Point) error {
	if dst == nil || src == nil {
		return fmt.Errorf("nil image")
	}

	// Clip destination rectangle to image bounds
	dstR := r
	if dstR.Min.X < dst.Rect.Min.X {
		dstR.Min.X = dst.Rect.Min.X
	}
	if dstR.Min.Y < dst.Rect.Min.Y {
		dstR.Min.Y = dst.Rect.Min.Y
	}
	if dstR.Max.X > dst.Rect.Max.X {
		dstR.Max.X = dst.Rect.Max.X
	}
	if dstR.Max.Y > dst.Rect.Max.Y {
		dstR.Max.Y = dst.Rect.Max.Y
	}

	// Check if rectangle is valid after clipping
	if dstR.Min.X >= dstR.Max.X || dstR.Min.Y >= dstR.Max.Y {
		return nil // Empty rectangle
	}

	// Calculate corresponding source point
	srcX := p.X + (dstR.Min.X - r.Min.X)
	srcY := p.Y + (dstR.Min.Y - r.Min.Y)

	// Perform draw operation
	if mask == nil {
		return drawWithoutMask(dst, dstR, src, Point{srcX, srcY})
	}
	return drawWithMask(dst, dstR, src, Point{srcX, srcY}, mask)
}

// drawWithoutMask copies pixels from src to dst without masking
func drawWithoutMask(dst *Image, dstR Rectangle, src *Image, sp Point) error {
	bpp := BytesPerPixel(dst.Chan)
	if bpp != BytesPerPixel(src.Chan) {
		return fmt.Errorf("channel format mismatch")
	}

	// Copy pixels row by row
	for y := dstR.Min.Y; y < dstR.Max.Y; y++ {
		for x := dstR.Min.X; x < dstR.Max.X; x++ {
			// Check clipping
			if !dst.ClipR.In(Point{x, y}) {
				continue
			}

			// Calculate source position
			sx := sp.X + (x - dstR.Min.X)
			sy := sp.Y + (y - dstR.Min.Y)

			// Check source bounds
			if sx < src.Rect.Min.X || sx >= src.Rect.Max.X ||
				sy < src.Rect.Min.Y || sy >= src.Rect.Max.Y {
				// Handle replication if source has repl set
				if !src.Repl {
					continue
				}
				// Wrap coordinates for replication
				sx = src.Rect.Min.X + (sx-src.Rect.Min.X)%src.Rect.Dx()
				if sx < src.Rect.Min.X {
					sx += src.Rect.Dx()
				}
				sy = src.Rect.Min.Y + (sy-src.Rect.Min.Y)%src.Rect.Dy()
				if sy < src.Rect.Min.Y {
					sy += src.Rect.Dy()
				}
			}

			// Get source pixel
			srcColor, err := src.GetPixel(sx, sy)
			if err != nil {
				continue
			}

			// Set destination pixel
			if err := dst.SetPixel(x, y, srcColor); err != nil {
				continue
			}
		}
	}

	return nil
}

// drawWithMask copies pixels from src to dst with mask applied
func drawWithMask(dst *Image, dstR Rectangle, src *Image, sp Point, mask *Image) error {
	bpp := BytesPerPixel(dst.Chan)
	if bpp != BytesPerPixel(src.Chan) {
		return fmt.Errorf("channel format mismatch")
	}

	// Calculate mask position
	maskP := Point{
		X: sp.X - src.Rect.Min.X,
		Y: sp.Y - src.Rect.Min.Y,
	}

	// Copy pixels row by row with mask
	for y := dstR.Min.Y; y < dstR.Max.Y; y++ {
		for x := dstR.Min.X; x < dstR.Max.X; x++ {
			// Check clipping
			if !dst.ClipR.In(Point{x, y}) {
				continue
			}

			// Calculate source position
			sx := sp.X + (x - dstR.Min.X)
			sy := sp.Y + (y - dstR.Min.Y)

			// Check source bounds
			if sx < src.Rect.Min.X || sx >= src.Rect.Max.X ||
				sy < src.Rect.Min.Y || sy >= src.Rect.Max.Y {
				if !src.Repl {
					continue
				}
				// Wrap for replication
				sx = src.Rect.Min.X + (sx-src.Rect.Min.X)%src.Rect.Dx()
				if sx < src.Rect.Min.X {
					sx += src.Rect.Dx()
				}
				sy = src.Rect.Min.Y + (sy-src.Rect.Min.Y)%src.Rect.Dy()
				if sy < src.Rect.Min.Y {
					sy += src.Rect.Dy()
				}
			}

			// Check mask
			mx := maskP.X + (x - dstR.Min.X)
			my := maskP.Y + (y - dstR.Min.Y)
			maskAlpha := uint32(0xFFFFFFFF)

			if mx >= mask.Rect.Min.X && mx < mask.Rect.Max.X &&
				my >= mask.Rect.Min.Y && my < mask.Rect.Max.Y {
				maskColor, _ := mask.GetPixel(mx, my)
				maskAlpha = maskColor
			}

			// Skip if mask is fully transparent
			if maskAlpha == 0 {
				continue
			}

			// Get source pixel
			srcColor, err := src.GetPixel(sx, sy)
			if err != nil {
				continue
			}

			// TODO: Apply proper alpha blending with mask
			// For now, just copy if mask is non-zero
			if err := dst.SetPixel(x, y, srcColor); err != nil {
				continue
			}
		}
	}

	return nil
}

// Line draws a line from p0 to p1 with the given end styles and thickness
// end0, end1: endpoint styles (0=square, >0=disc radius)
// thickness: line thickness in pixels
// src: source image for drawing
// sp: position in source image to start from
func Line(dst *Image, p0, p1 Point, end0, end1, thickness int32, src *Image, sp Point) error {
	if thickness < 0 {
		thickness = 1
	}

	// Simple line drawing using Bresenham's algorithm
	dx := abs(p1.X - p0.X)
	dy := -abs(p1.Y - p0.Y)
	err := dx + dy
	sx := int32(1)
	if p0.X > p1.X {
		sx = -1
	}
	sy := int32(1)
	if p0.Y > p1.Y {
		sy = -1
	}

	x, y := p0.X, p0.Y
	srcX, srcY := sp.X, sp.Y

	for {
		// Draw pixel at current position
		if thickness > 1 {
			// Draw thick line as rectangle
			r := Rect(x-thickness/2, y-thickness/2, x+thickness/2+1, y+thickness/2+1)
			Draw(dst, r, src, nil, Point{srcX, srcY})
		} else {
			color, _ := src.GetPixel(srcX, srcY)
			dst.SetPixel(x, y, color)
		}

		if x == p1.X && y == p1.Y {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
			srcX += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
			srcY += sy
		}
	}

	// Draw endpoints
	if end0 > 0 {
		drawEndpoint(dst, p0, end0, src, sp)
	}
	if end1 > 0 {
		dx := p1.X - p0.X
		dy := p1.Y - p0.Y
		sp.X += dx
		sp.Y += dy
		drawEndpoint(dst, p1, end1, src, sp)
	}

	return nil
}

// drawEndpoint draws a circular endpoint
func drawEndpoint(dst *Image, center Point, radius int32, src *Image, sp Point) error {
	r := Rect(center.X-radius, center.Y-radius, center.X+radius+1, center.Y+radius+1)
	return Draw(dst, r, src, nil, sp)
}

// abs returns absolute value of int32
func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// Polygon draws a polygon with the given vertices
// winding: 0 for even-odd, non-zero for non-zero winding rule
func Polygon(dst *Image, points []Point, winding int32, src *Image, sp Point) error {
	if len(points) < 2 {
		return fmt.Errorf("polygon needs at least 2 points")
	}

	// Draw edges
	for i := 0; i < len(points)-1; i++ {
		err := Line(dst, points[i], points[i+1], 0, 0, 1, src, sp)
		if err != nil {
			return err
		}
	}

	// Close the polygon
	return Line(dst, points[len(points)-1], points[0], 0, 0, 1, src, sp)
}

// FillPolygon fills a polygon with the source image
func FillPolygon(dst *Image, points []Point, winding int32, src *Image, sp Point) error {
	// TODO: Implement proper polygon fill algorithm
	// For now, just use scanline filling
	if len(points) < 3 {
		return fmt.Errorf("filled polygon needs at least 3 points")
	}

	// Find bounding box
	minX, minY := points[0].X, points[0].Y
	maxX, maxY := points[0].X, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Fill bounding box with clipping to polygon
	// TODO: Implement proper point-in-polygon test
	r := Rect(minX, minY, maxX, maxY)
	return Draw(dst, r, src, nil, sp)
}

// Ellipse draws an ellipse
// center: ellipse center
// a, b: semi-major and semi-minor axes
// thick: thickness (0 for filled)
func Ellipse(dst *Image, center Point, a, b, thick int32, src *Image, sp Point) error {
	if thick < 0 {
		thick = 0
	}

	if thick == 0 {
		// Filled ellipse
		return fillEllipse(dst, center, a, b, src, sp)
	}

	// Outline ellipse
	return drawEllipseOutline(dst, center, a, b, thick, src, sp)
}

// fillEllipse fills an ellipse
func fillEllipse(dst *Image, center Point, a, b int32, src *Image, sp Point) error {
	// Simple scanline fill
	for y := -b; y <= b; y++ {
		for x := -a; x <= a; x++ {
			// Check if point is inside ellipse
			xx := int64(x) * int64(x)
			aa := int64(a) * int64(a)
			yy := int64(y) * int64(y)
			bb := int64(b) * int64(b)

			if xx*aa+yy*bb <= aa*bb {
				color, _ := src.GetPixel(sp.X, sp.Y)
				dst.SetPixel(center.X+x, center.Y+y, color)
			}
		}
	}

	return nil
}

// drawEllipseOutline draws an ellipse outline
func drawEllipseOutline(dst *Image, center Point, a, b, thick int32, src *Image, sp Point) error {
	// Draw outline by checking distance from ellipse boundary
	for y := -b - thick; y <= b+thick; y++ {
		for x := -a - thick; x <= a+thick; x++ {
			// Calculate distance from ellipse
			xx := float64(x)
			yy := float64(y)
			af := float64(a)
			bf := float64(b)

			// Normalize to unit circle
			nx := xx / af
			ny := yy / bf
			dist := sqrt(nx*nx + ny*ny)

			// Check if within thickness of ellipse boundary
			if dist >= 1.0 && dist <= 1.0+float64(thick)/min(af, bf) {
				color, _ := src.GetPixel(sp.X, sp.Y)
				dst.SetPixel(center.X+int32(x), center.Y+int32(y), color)
			}
		}
	}

	return nil
}

// sqrt returns square root (simple implementation)
func sqrt(x float64) float64 {
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}

// min returns minimum of two floats
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Text draws text at the specified position
// TODO: Implement font support
func Text(dst *Image, p Point, src *Image, font string, text string) error {
	// TODO: Implement proper font rendering
	// For now, this is a stub
	return fmt.Errorf("text rendering not yet implemented")
}
