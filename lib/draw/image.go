package draw

import (
	"fmt"
	"sync"
)

// ImageManager manages Plan 9 images
type ImageManager struct {
	images map[int]*Image
	nextID int
	mu     sync.Mutex
}

// NewImageManager creates a new image manager
func NewImageManager() *ImageManager {
	return &ImageManager{
		images: make(map[int]*Image),
		nextID: 1,
	}
}

// AllocImage allocates a new image with the given properties
func (im *ImageManager) AllocImage(r Rectangle, chanDesc uint32, repl bool, color uint32) (*Image, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Calculate size based on rectangle and channel format
	bytesPerPixel := BytesPerPixel(chanDesc)
	pitch := int(r.Dx()) * bytesPerPixel
 dataSize := pitch * int(r.Dy())

	// Allocate pixel data
	pixels := make([]byte, dataSize)

	// Fill with initial color if not using a screen buffer
	if color != 0 {
		fillColor := colorToBytes(color, chanDesc)
		for i := 0; i < len(pixels); i += bytesPerPixel {
			copy(pixels[i:i+bytesPerPixel], fillColor)
		}
	}

	image := &Image{
		ID:     im.nextID,
		Rect:   r,
		Chan:   chanDesc,
		Data:   pixels,
		ClipR:  r, // Initial clip is the full image
		Repl:   repl,
		Ref:    1,
		Screen: nil,
	}

	im.images[im.nextID] = image
	im.nextID++

	return image, nil
}

// LookupImage finds an image by ID
func (im *ImageManager) LookupImage(id int) (*Image, bool) {
	im.mu.Lock()
	defer im.mu.Unlock()

	img, ok := im.images[id]
	return img, ok
}

// FreeImage decrements reference count and frees if zero
func (im *ImageManager) FreeImage(id int) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	img, ok := im.images[id]
	if !ok {
		return fmt.Errorf("image %d not found", id)
	}

	img.Ref--
	if img.Ref <= 0 {
		delete(im.images, id)
	}

	return nil
}

// BytesPerPixel returns the number of bytes per pixel for a channel descriptor
func BytesPerPixel(ch uint32) int {
	// Extract the bytes per pixel from channel descriptor
	// Format: ABC... where each nibble represents a channel
	bpp := 0

	// Check each channel (4 nibbles max, 16 bits max)
	for i := 0; i < 4; i++ {
		nibble := (ch >> (i * 4)) & 0xF
		if nibble != 0 {
			bpp++
		}
	}

	if bpp == 0 {
		return 1 // At least 1 byte per pixel
	}

	return bpp
}

// colorToBytes converts a color value to byte array based on channel format
func colorToBytes(color uint32, chanDesc uint32) []byte {
	bpp := BytesPerPixel(chanDesc)
	bytes := make([]byte, bpp)

	// Extract color components based on channel descriptor
	// For now, handle common formats
	switch chanDesc {
	case RGB24, RGBA32, ARGB32, XRGB32:
		// Extract RGB components
		r := uint8((color >> 16) & 0xFF)
		g := uint8((color >> 8) & 0xFF)
		b := uint8(color & 0xFF)
		a := uint8((color >> 24) & 0xFF)

		switch chanDesc {
		case RGB24:
			bytes[0] = r
			bytes[1] = g
			bytes[2] = b
		case RGBA32:
			bytes[0] = r
			bytes[1] = g
			bytes[2] = b
			bytes[3] = a
		case ARGB32:
			bytes[0] = a
			bytes[1] = r
			bytes[2] = g
			bytes[3] = b
		case XRGB32:
			bytes[0] = 0xFF // X = opaque
			bytes[1] = r
			bytes[2] = g
			bytes[3] = b
		}
	default:
		// For other formats, just fill with the color value
		for i := range bytes {
			bytes[i] = uint8(color >> (i * 8))
		}
	}

	return bytes
}

// SetPixel sets a pixel at the given coordinates
func (img *Image) SetPixel(x, y int32, color uint32) error {
	if x < img.Rect.Min.X || x >= img.Rect.Max.X ||
		y < img.Rect.Min.Y || y >= img.Rect.Max.Y {
		return fmt.Errorf("coordinate (%d,%d) out of bounds", x, y)
	}

	// Check clipping
	if !img.ClipR.In(Point{x, y}) {
		return nil
	}

	// Calculate pixel index
	bpp := BytesPerPixel(img.Chan)
	offset := ((int(y-img.Rect.Min.Y)*int(img.Rect.Dx()) + int(x-img.Rect.Min.X))) * bpp

	if offset < 0 || offset+bpp > len(img.Data) {
		return fmt.Errorf("pixel offset out of bounds")
	}

	// Convert color to bytes
	colorBytes := colorToBytes(color, img.Chan)
	copy(img.Data[offset:offset+bpp], colorBytes)

	return nil
}

// GetPixel gets the pixel color at the given coordinates
func (img *Image) GetPixel(x, y int32) (uint32, error) {
	if x < img.Rect.Min.X || x >= img.Rect.Max.X ||
		y < img.Rect.Min.Y || y >= img.Rect.Max.Y {
		return 0, fmt.Errorf("coordinate (%d,%d) out of bounds", x, y)
	}

	// Check clipping
	if !img.ClipR.In(Point{x, y}) {
		return 0, nil // Out of clip region
	}

	// Calculate pixel index
	bpp := BytesPerPixel(img.Chan)
	offset := ((int(y-img.Rect.Min.Y)*int(img.Rect.Dx()) + int(x-img.Rect.Min.X))) * bpp

	if offset < 0 || offset+bpp > len(img.Data) {
		return 0, fmt.Errorf("pixel offset out of bounds")
	}

	// Convert bytes to color based on channel format
	colorBytes := img.Data[offset : offset+bpp]
	var color uint32

	switch img.Chan {
	case RGB24:
		color = uint32(colorBytes[0])<<16 | uint32(colorBytes[1])<<8 | uint32(colorBytes[2])
	case RGBA32:
		color = uint32(colorBytes[0])<<16 | uint32(colorBytes[1])<<8 | uint32(colorBytes[2]) | uint32(colorBytes[3])<<24
	case ARGB32:
		color = uint32(colorBytes[1])<<16 | uint32(colorBytes[2])<<8 | uint32(colorBytes[3]) | uint32(colorBytes[0])<<24
	case XRGB32:
		color = uint32(colorBytes[1])<<16 | uint32(colorBytes[2])<<8 | uint32(colorBytes[3]) | 0xFF000000
	default:
		// Generic conversion
		for i, b := range colorBytes {
			color |= uint32(b) << (i * 8)
		}
	}

	return color, nil
}

// LoadData loads pixel data into the image
func (img *Image) LoadData(data []byte, r Rectangle) error {
	if r.Min.X < img.Rect.Min.X || r.Max.X > img.Rect.Max.X ||
		r.Min.Y < img.Rect.Min.Y || r.Max.Y > img.Rect.Max.Y {
		return fmt.Errorf("rectangle out of image bounds")
	}

	// Calculate offset and size
	bpp := BytesPerPixel(img.Chan)
	width := int(r.Dx())
	height := int(r.Dy())
	rowSize := width * bpp

	// Copy data row by row
	for y := 0; y < height; y++ {
		dstY := int(r.Min.Y-img.Rect.Min.Y) + y
		dstX := int(r.Min.X-img.Rect.Min.X)
		offset := (dstY*int(img.Rect.Dx()) + dstX) * bpp

		srcOffset := y * rowSize
		if offset+rowSize > len(img.Data) || srcOffset+rowSize > len(data) {
			return fmt.Errorf("data size mismatch")
		}

		copy(img.Data[offset:offset+rowSize], data[srcOffset:srcOffset+rowSize])
	}

	return nil
}

// GetData gets pixel data from the image
func (img *Image) GetData(r Rectangle) ([]byte, error) {
	if r.Min.X < img.Rect.Min.X || r.Max.X > img.Rect.Max.X ||
		r.Min.Y < img.Rect.Min.Y || r.Max.Y > img.Rect.Max.Y {
		return nil, fmt.Errorf("rectangle out of image bounds")
	}

	// Calculate size
	bpp := BytesPerPixel(img.Chan)
	width := int(r.Dx())
	height := int(r.Dy())
	dataSize := width * height * bpp

	// Allocate result buffer
	data := make([]byte, dataSize)

	// Copy data row by row
	for y := 0; y < height; y++ {
		srcY := int(r.Min.Y-img.Rect.Min.Y) + y
		srcX := int(r.Min.X-img.Rect.Min.X)
		offset := (srcY*int(img.Rect.Dx()) + srcX) * bpp

		dstOffset := y * width * bpp
		if offset+bpp > len(img.Data) {
			return nil, fmt.Errorf("source data out of bounds")
		}

		copy(data[dstOffset:dstOffset+width*bpp], img.Data[offset:offset+width*bpp])
	}

	return data, nil
}

// Clone creates a copy of the image
func (img *Image) Clone() (*Image, error) {
	img.Ref++

	// Create new image with same properties
	cloned := &Image{
		ID:     img.ID, // Same ID initially
		Rect:   img.Rect,
		Chan:   img.Chan,
		Data:   make([]byte, len(img.Data)),
		ClipR:  img.ClipR,
		Repl:   img.Repl,
		Ref:    1,
		Screen: img.Screen,
	}

	// Copy pixel data
	copy(cloned.Data, img.Data)

	return cloned, nil
}

// AddRef increments the reference count
func (img *Image) AddRef() {
	img.Ref++
}

// SubRef decrements the reference count
func (img *Image) SubRef() int {
	img.Ref--
	return img.Ref
}

// GetSize returns the size of the image in bytes
func (img *Image) GetSize() int {
	return len(img.Data)
}

// GetWidth returns the width of the image
func (img *Image) GetWidth() int {
	return int(img.Rect.Dx())
}

// GetHeight returns the height of the image
func (img *Image) GetHeight() int {
	return int(img.Rect.Dy())
}

// Clip sets the clipping rectangle
func (img *Image) Clip(r Rectangle) {
	img.ClipR = r
}

// Clear fills the image with a color
func (img *Image) Clear(color uint32) error {
	bpp := BytesPerPixel(img.Chan)
	colorBytes := colorToBytes(color, img.Chan)

	for i := 0; i < len(img.Data); i += bpp {
		copy(img.Data[i:i+bpp], colorBytes)
	}

	return nil
}

// FillRectangle fills a rectangle with a color
func (img *Image) FillRectangle(r Rectangle, color uint32) error {
	// Clip to image bounds
	if r.Min.X < img.Rect.Min.X {
		r.Min.X = img.Rect.Min.X
	}
	if r.Max.X > img.Rect.Max.X {
		r.Max.X = img.Rect.Max.X
	}
	if r.Min.Y < img.Rect.Min.Y {
		r.Min.Y = img.Rect.Min.Y
	}
	if r.Max.Y > img.Rect.Max.Y {
		r.Max.Y = img.Rect.Max.Y
	}

	// Check if rectangle is valid after clipping
	if r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y {
		return nil // Empty rectangle
	}

	bpp := BytesPerPixel(img.Chan)
	colorBytes := colorToBytes(color, img.Chan)

	// Fill rectangle row by row
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			offset := ((int(y-img.Rect.Min.Y)*int(img.Rect.Dx()) + int(x-img.Rect.Min.X))) * bpp
			if offset >= 0 && offset+bpp <= len(img.Data) {
				copy(img.Data[offset:offset+bpp], colorBytes)
			}
		}
	}

	return nil
}
