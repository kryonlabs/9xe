package draw

import (
	"fmt"
)

// DrawOp represents Porter-Duff compositing operators
type DrawOp int

const (
	DrawOpClear    DrawOp = 0  // Clear destination
	DrawOpSinD     DrawOp = 8  // Source in Destination
	DrawOpDinS     DrawOp = 4  // Destination in Source
	DrawOpSoutD    DrawOp = 2  // Source out Destination
	DrawOpDoutS    DrawOp = 1  // Destination out Source
	DrawOpS        DrawOp = DrawOpSinD | DrawOpSoutD
	DrawOpSoverD   DrawOp = DrawOpSinD | DrawOpSoutD | DrawOpDoutS
	DrawOpD        DrawOp = DrawOpDinS | DrawOpDoutS
	DrawOpDoverS   DrawOp = DrawOpDinS | DrawOpDoutS | DrawOpSoutD
	DrawOpSatOverD DrawOp = DrawOpDinS | DrawOpSoutD
)

// Composite performs Porter-Duff compositing of src onto dst
// op: compositing operation
// r: destination rectangle
// sp: source point (top-left of source region)
// mask: optional mask image (can be nil)
func Composite(dst *Image, r Rectangle, src *Image, sp Point, mask *Image, op DrawOp) error {
	if dst == nil || src == nil {
		return fmt.Errorf("nil image")
	}

	// Clip to destination bounds
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

	if dstR.Min.X >= dstR.Max.X || dstR.Min.Y >= dstR.Max.Y {
		return nil
	}

	// Perform compositing
	for y := dstR.Min.Y; y < dstR.Max.Y; y++ {
		for x := dstR.Min.X; x < dstR.Max.X; x++ {
			// Check clipping
			if !dst.ClipR.In(Point{x, y}) {
				continue
			}

			// Calculate source position
			sx := sp.X + (x - r.Min.X)
			sy := sp.Y + (y - r.Min.Y)

			// Handle source replication
			if sx < src.Rect.Min.X || sx >= src.Rect.Max.X ||
				sy < src.Rect.Min.Y || sy >= src.Rect.Max.Y {
				if !src.Repl {
					continue
				}
				sx = src.Rect.Min.X + (sx-src.Rect.Min.X)%src.Rect.Dx()
				if sx < src.Rect.Min.X {
					sx += src.Rect.Dx()
				}
				sy = src.Rect.Min.Y + (sy-src.Rect.Min.Y)%src.Rect.Dy()
				if sy < src.Rect.Min.Y {
					sy += src.Rect.Dy()
				}
			}

			// Get source and destination pixels
			srcColor, err := src.GetPixel(sx, sy)
			if err != nil {
				continue
			}

			dstColor, err := dst.GetPixel(x, y)
			if err != nil {
				continue
			}

			// Apply mask if present
			maskAlpha := uint32(0xFFFFFFFF)
			if mask != nil {
				mx := sp.X - src.Rect.Min.X + (x - r.Min.X)
				my := sp.Y - src.Rect.Min.Y + (y - r.Min.Y)
				if mx >= mask.Rect.Min.X && mx < mask.Rect.Max.X &&
					my >= mask.Rect.Min.Y && my < mask.Rect.Max.Y {
					maskColor, _ := mask.GetPixel(mx, my)
					maskAlpha = maskColor
				}
			}

			// Perform compositing operation
			result := composeOp(srcColor, dstColor, maskAlpha, op, dst.Chan)

			// Set result pixel
			dst.SetPixel(x, y, result)
		}
	}

	return nil
}

// composeOp applies the compositing operation
func composeOp(src, dst, mask uint32, op DrawOp, chanDesc uint32) uint32 {
	// Extract alpha channels
	srcA, srcR, srcG, srcB := unpackRGBA(src, chanDesc)
	dstA, dstR, dstG, dstB := unpackRGBA(dst, chanDesc)
	maskA, _, _, _ := unpackRGBA(mask, chanDesc)

	// Normalize alpha to 0-1 range
	fa := float32(srcA) / 255.0
	fb := float32(dstA) / 255.0
	fm := float32(maskA) / 255.0

	// Apply mask to source alpha
	fa = fa * fm

	// Calculate Porter-Duff factors
	var fs, fd float32

	switch op {
	case DrawOpClear:
		fs, fd = 0, 0
	case DrawOpSinD:
		fs, fd = fb, 0
	case DrawOpDinS:
		fs, fd = 1, 0
	case DrawOpSoutD:
		fs, fd = 1-fb, 0
	case DrawOpDoutS:
		fs, fd = 0, 1-fa
	case DrawOpS:
		fs, fd = 1, 0
	case DrawOpSoverD:
		fs, fd = 1, 1-fa
	case DrawOpD:
		fs, fd = 0, 1
	case DrawOpDoverS:
		fs, fd = 1-fb, 1
	case DrawOpSatOverD:
		fs, fd = minF(fb, 1-fa), 1-fa
	default:
		fs, fd = 1, 0
	}

	// Calculate resulting alpha
	outA := fs*fa + fd*fb
	if outA > 1.0 {
		outA = 1.0
	}
	if outA < 0.0 {
		outA = 0.0
	}

	// Calculate resulting RGB components
	// If alpha is 0, RGB is undefined (set to 0)
	var outR, outG, outB float32
	if outA > 0 {
		outR = (fs*fa*float32(srcR) + fd*fb*float32(dstR)) / outA
		outG = (fs*fa*float32(srcG) + fd*fb*float32(dstG)) / outA
		outB = (fs*fa*float32(srcB) + fd*fb*float32(dstB)) / outA
	}

	// Clamp and pack result
	return packRGBA(
		uint8(outA*255.0+0.5),
		uint8(outR*255.0+0.5),
		uint8(outG*255.0+0.5),
		uint8(outB*255.0+0.5),
		chanDesc,
	)
}

// unpackRGBA extracts RGBA components from a pixel value
func unpackRGBA(pixel uint32, chanDesc uint32) (a, r, g, b uint8) {
	switch chanDesc {
	case RGB24:
		a = 0xFF
		r = uint8((pixel >> 16) & 0xFF)
		g = uint8((pixel >> 8) & 0xFF)
		b = uint8(pixel & 0xFF)
	case RGBA32:
		a = uint8((pixel >> 24) & 0xFF)
		r = uint8((pixel >> 16) & 0xFF)
		g = uint8((pixel >> 8) & 0xFF)
		b = uint8(pixel & 0xFF)
	case ARGB32:
		a = uint8((pixel >> 24) & 0xFF)
		r = uint8((pixel >> 16) & 0xFF)
		g = uint8((pixel >> 8) & 0xFF)
		b = uint8(pixel & 0xFF)
	case XRGB32:
		a = 0xFF
		r = uint8((pixel >> 16) & 0xFF)
		g = uint8((pixel >> 8) & 0xFF)
		b = uint8(pixel & 0xFF)
	case GREY8:
		v := uint8(pixel & 0xFF)
		a = 0xFF
		r, g, b = v, v, v
	case CMAP8:
		// TODO: Implement color map lookup
		a = 0xFF
		r = uint8(pixel & 0xFF)
		g = uint8(pixel & 0xFF)
		b = uint8(pixel & 0xFF)
	default:
		// Generic extraction
		a = uint8((pixel >> 24) & 0xFF)
		r = uint8((pixel >> 16) & 0xFF)
		g = uint8((pixel >> 8) & 0xFF)
		b = uint8(pixel & 0xFF)
	}

	return
}

// packRGBA packs RGBA components into a pixel value
func packRGBA(a, r, g, b uint8, chanDesc uint32) uint32 {
	switch chanDesc {
	case RGB24:
		return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	case RGBA32:
		return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	case ARGB32:
		return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	case XRGB32:
		return 0xFF000000 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	case GREY8:
		// Use luminance formula
		gray := uint32((uint32(r)*299 + uint32(g)*587 + uint32(b)*114) / 1000)
		return gray
	case CMAP8:
		// TODO: Implement color map lookup
		return uint32(r)
	default:
		return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	}
}

// minF returns minimum of two float32 values
func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
