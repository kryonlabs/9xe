package memdraw

import (
	"github.com/kryonlabs/9xe/lib/draw"
)

// MemFillColor fills an image with a solid color
func MemFillColor(img *MemImage, color uint32) error {
	if img == nil {
		return nil
	}

	// Convert color to native format
	val := colorValue(img.Chan, color)

	bytesPerRow := bytesPerRow(img.Rect, img.Chan)
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		row := img.Data[int(y-img.Rect.Min.Y)*bytesPerRow:]
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			pixOffset := int(x-img.Rect.Min.X) * (img.Depth / 8)
			if pixOffset+4 <= len(row) {
				copy(row[pixOffset:pixOffset+4], val[:])
			}
		}
	}

	return nil
}

// colorValue converts a color value to the image's pixel format
func colorValue(chanfmt uint32, color uint32) []byte {
	val := make([]byte, 4)

	switch chanfmt {
	case draw.RGB24:
		val[0] = byte(color >> 16) // R
		val[1] = byte(color >> 8)  // G
		val[2] = byte(color)       // B
	case draw.RGBA32:
		val[0] = byte(color >> 24) // R
		val[1] = byte(color >> 16) // G
		val[2] = byte(color >> 8)  // B
		val[3] = byte(color)       // A
	case draw.ARGB32:
		val[0] = byte(color >> 24) // A
		val[1] = byte(color >> 16) // R
		val[2] = byte(color >> 8)  // G
		val[3] = byte(color)       // B
	case draw.XRGB32:
		val[0] = byte(color >> 16) // R
		val[1] = byte(color >> 8)  // G
		val[2] = byte(color)       // B
		val[3] = 0xFF              // X
	default:
		// Default to XRGB32
		val[0] = byte(color >> 16)
		val[1] = byte(color >> 8)
		val[2] = byte(color)
		val[3] = 0xFF
	}

	return val
}
