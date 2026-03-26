package memdraw

import (
	"github.com/kryonlabs/9xe/lib/draw"
)

// MemDraw draws from src to dst using mask
func MemDraw(dst, src, mask *MemImage, r draw.Rectangle, sp, mp draw.Point, op int) error {
	if dst == nil || src == nil {
		return nil
	}

	// Simple blit implementation
	// TODO: Implement proper compositing with mask and operation

	srcY := sp.Y
	for y := r.Min.Y; y < r.Max.Y; y++ {
		srcX := sp.X
		for x := r.Min.X; x < r.Max.X; x++ {
			if srcX >= src.Rect.Min.X && srcX < src.Rect.Max.X &&
				srcY >= src.Rect.Min.Y && srcY < src.Rect.Max.Y &&
				x >= dst.Rect.Min.X && x < dst.Rect.Max.X &&
				y >= dst.Rect.Min.Y && y < dst.Rect.Max.Y {

				// Copy pixel
				copyPixel(dst, x, y, src, srcX, srcY)
			}
			srcX++
		}
		srcY++
	}

	return nil
}

// copyPixel copies a pixel from src to dst
func copyPixel(dst *MemImage, dx, dy int32, src *MemImage, sx, sy int32) {
	dstBytes := (dst.Depth + 7) / 8
	srcBytes := (src.Depth + 7) / 8

	dstOffset := pixelOffset(dst, dx, dy)
	srcOffset := pixelOffset(src, sx, sy)

	if dstOffset+dstBytes <= len(dst.Data) && srcOffset+srcBytes <= len(src.Data) {
		// Simple copy - TODO: handle format conversion
		minBytes := dstBytes
		if srcBytes < minBytes {
			minBytes = srcBytes
		}
		copy(dst.Data[dstOffset:dstOffset+minBytes], src.Data[srcOffset:srcOffset+minBytes])
	}
}

// pixelOffset calculates the byte offset for a pixel
func pixelOffset(img *MemImage, x, y int32) int {
	bytesPerRow := bytesPerRow(img.Rect, img.Chan)
	bytesPerPixel := (img.Depth + 7) / 8
	return int(y-img.Rect.Min.Y)*bytesPerRow + int(x-img.Rect.Min.X)*bytesPerPixel
}
