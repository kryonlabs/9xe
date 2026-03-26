package draw

import (
	"encoding/binary"
	"fmt"
)

// DrawClient handles draw protocol commands for a client connection
type DrawClient struct {
	images map[int]*Image
	screen *Screen
	nextID int
}

// NewDrawClient creates a new draw client
func NewDrawClient(screen *Screen) *DrawClient {
	return &DrawClient{
		images: make(map[int]*Image),
		screen: screen,
		nextID: 1,
	}
}

// HandleCommand parses and executes a draw command
func (c *DrawClient) HandleCommand(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, fmt.Errorf("empty command")
	}

	opcode := buf[0]
	var err error

	switch opcode {
	case AllocOpcode:
		err = c.allocateImage(buf[1:])
	case DrawOpcode:
		err = c.drawImage(buf[1:])
	case ClipOpcode:
		err = c.setClip(buf[1:])
	case FreeOpcode:
		err = c.freeImage(buf[1:])
	case FlushOpcode:
		err = c.flush(buf[1:])
	case WriteOpcode:
		err = c.writePixels(buf[1:])
	default:
		return 0, fmt.Errorf("unknown opcode: %c", opcode)
	}

	if err != nil {
		return 0, err
	}

	return len(buf), nil
}

// allocateImage handles 'b' command
// Format: b id[4] screenid[4] refresh[1] chan[4] repl[1] R[4*4] clipR[4*4] rrggbbaa[4]
func (c *DrawClient) allocateImage(buf []byte) error {
	const minLen = 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4
	if len(buf) < minLen {
		return fmt.Errorf("allocate: buffer too short")
	}

	id := int(binary.LittleEndian.Uint32(buf[0:4]))
	screenid := int(binary.LittleEndian.Uint32(buf[4:8]))
	// refresh := buf[8]
	chanfmt := binary.LittleEndian.Uint32(buf[9:13])
	repl := buf[13] != 0

	// Parse rectangle R
	rect := Rectangle{
		Min: Point{
			X: int32(binary.LittleEndian.Uint32(buf[14:18])),
			Y: int32(binary.LittleEndian.Uint32(buf[18:22])),
		},
		Max: Point{
			X: int32(binary.LittleEndian.Uint32(buf[22:26])),
			Y: int32(binary.LittleEndian.Uint32(buf[26:30])),
		},
	}

	// Parse clip rectangle
	clipr := Rectangle{
		Min: Point{
			X: int32(binary.LittleEndian.Uint32(buf[30:34])),
			Y: int32(binary.LittleEndian.Uint32(buf[34:38])),
		},
		Max: Point{
			X: int32(binary.LittleEndian.Uint32(buf[38:42])),
			Y: int32(binary.LittleEndian.Uint32(buf[42:46])),
		},
	}

	// color := binary.LittleEndian.Uint32(buf[46:50])

	// Create the image
	img := &Image{
		ID:    id,
		Rect:  rect,
		Chan:  chanfmt,
		ClipR: clipr,
		Repl:  repl,
		Ref:   1,
	}

	// Allocate pixel data
	pixelsPerRow := int(rect.Dx())
	bytesPerPixel := bytesPerPixel(chanfmt)
	img.Data = make([]byte, int(rect.Dy())*pixelsPerRow*bytesPerPixel)

	// If screenid != 0, this is a window/layer
	if screenid != 0 {
		// TODO: Implement layer/window support
	}

	c.images[id] = img
	return nil
}

// drawImage handles 'd' command
// Format: d dstid[4] srcid[4] maskid[4] R[4*4] P[2*4] P[2*4]
func (c *DrawClient) drawImage(buf []byte) error {
	const minLen = 4 + 4 + 4 + 4*4 + 2*4 + 2*4
	if len(buf) < minLen {
		return fmt.Errorf("draw: buffer too short")
	}

	dstid := int(binary.LittleEndian.Uint32(buf[0:4]))
	srcid := int(binary.LittleEndian.Uint32(buf[4:8]))
	maskid := int(binary.LittleEndian.Uint32(buf[8:12]))

	// Parse destination rectangle R
	r := Rectangle{
		Min: Point{
			X: int32(binary.LittleEndian.Uint32(buf[12:16])),
			Y: int32(binary.LittleEndian.Uint32(buf[16:20])),
		},
		Max: Point{
			X: int32(binary.LittleEndian.Uint32(buf[20:24])),
			Y: int32(binary.LittleEndian.Uint32(buf[24:28])),
		},
	}

	// Parse source point P
	p := Point{
		X: int32(binary.LittleEndian.Uint32(buf[28:32])),
		Y: int32(binary.LittleEndian.Uint32(buf[32:36])),
	}

	// Parse mask point P (we ignore this for now)
	// mp := Point{
	// 	X: int32(binary.LittleEndian.Uint32(buf[36:40])),
	// 	Y: int32(binary.LittleEndian.Uint32(buf[40:44])),
	// }

	dst := c.images[dstid]
	src := c.images[srcid]
	_ = maskid // TODO: handle mask

	if dst == nil || src == nil {
		return fmt.Errorf("draw: invalid image id")
	}

	// Perform the draw operation
	// TODO: Implement actual drawing with memdraw
	// For now, just copy pixel data
	c.blit(dst, r, src, p)

	return nil
}

// setClip handles 'c' command
// Format: c dstid[4] repl[1] clipR[4*4]
func (c *DrawClient) setClip(buf []byte) error {
	const minLen = 4 + 1 + 4*4
	if len(buf) < minLen {
		return fmt.Errorf("setclip: buffer too short")
	}

	dstid := int(binary.LittleEndian.Uint32(buf[0:4]))
	repl := buf[4] != 0

	clipr := Rectangle{
		Min: Point{
			X: int32(binary.LittleEndian.Uint32(buf[5:9])),
			Y: int32(binary.LittleEndian.Uint32(buf[9:13])),
		},
		Max: Point{
			X: int32(binary.LittleEndian.Uint32(buf[13:17])),
			Y: int32(binary.LittleEndian.Uint32(buf[17:21])),
		},
	}

	img := c.images[dstid]
	if img == nil {
		return fmt.Errorf("setclip: invalid image id")
	}

	img.Repl = repl
	img.ClipR = clipr

	return nil
}

// freeImage handles 'f' command
// Format: f id[4]
func (c *DrawClient) freeImage(buf []byte) error {
	if len(buf) < 4 {
		return fmt.Errorf("free: buffer too short")
	}

	id := int(binary.LittleEndian.Uint32(buf[0:4]))
	delete(c.images, id)
	return nil
}

// flush handles 'v' command
// Format: v
func (c *DrawClient) flush(buf []byte) error {
	// Flush the screen to the display
	if c.screen != nil && c.screen.Backend != nil {
		return c.screen.Backend.Flush()
	}
	return nil
}

// writePixels handles 'y' command
// Format: y id[4] R[4*4] data[x*1]
func (c *DrawClient) writePixels(buf []byte) error {
	if len(buf) < 4+4*4 {
		return fmt.Errorf("write: buffer too short")
	}

	id := int(binary.LittleEndian.Uint32(buf[0:4]))

	// Parse rectangle R
	r := Rectangle{
		Min: Point{
			X: int32(binary.LittleEndian.Uint32(buf[4:8])),
			Y: int32(binary.LittleEndian.Uint32(buf[8:12])),
		},
		Max: Point{
			X: int32(binary.LittleEndian.Uint32(buf[12:16])),
			Y: int32(binary.LittleEndian.Uint32(buf[16:20])),
		},
	}

	data := buf[20:]

	img := c.images[id]
	if img == nil {
		return fmt.Errorf("write: invalid image id")
	}

	// Write pixel data to the image
	// TODO: Implement proper pixel writing
	c.writePixelsData(img, r, data)

	return nil
}

// blit performs a simple blit operation
func (c *DrawClient) blit(dst *Image, r Rectangle, src *Image, sp Point) {
	// Simple implementation - just copy pixels
	// TODO: Implement proper memdraw blitting

	srcY := sp.Y
	for y := r.Min.Y; y < r.Max.Y; y++ {
		srcX := sp.X
		for x := r.Min.X; x < r.Max.X; x++ {
			// Copy pixel (very basic implementation)
			if srcX >= src.Rect.Min.X && srcX < src.Rect.Max.X &&
				srcY >= src.Rect.Min.Y && srcY < src.Rect.Max.Y &&
				x >= dst.Rect.Min.X && x < dst.Rect.Max.X &&
				y >= dst.Rect.Min.Y && y < dst.Rect.Max.Y {

				srcOff := c.offset(src, srcX, srcY)
				dstOff := c.offset(dst, x, y)

				if srcOff+4 <= len(src.Data) && dstOff+4 <= len(dst.Data) {
					copy(dst.Data[dstOff:dstOff+4], src.Data[srcOff:srcOff+4])
				}
			}
			srcX++
		}
		srcY++
	}
}

// writePixelsData writes pixel data to an image
func (c *DrawClient) writePixelsData(img *Image, r Rectangle, data []byte) {
	// Simple implementation - just copy data
	// TODO: Implement proper pixel writing with format conversion

	bytesPerPixel := bytesPerPixel(img.Chan)
	i := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if i+bytesPerPixel <= len(data) {
				off := c.offset(img, x, y)
				if off+bytesPerPixel <= len(img.Data) {
					copy(img.Data[off:off+bytesPerPixel], data[i:i+bytesPerPixel])
				}
				i += bytesPerPixel
			}
		}
	}
}

// offset calculates the byte offset for a pixel
func (c *DrawClient) offset(img *Image, x, y int32) int {
	width := int(img.Rect.Dx())
	bytesPerPixel := bytesPerPixel(img.Chan)
	return (int(y-img.Rect.Min.Y)*width + int(x-img.Rect.Min.X)) * bytesPerPixel
}

// bytesPerPixel returns the number of bytes per pixel for a channel format
func bytesPerPixel(chanfmt uint32) int {
	switch chanfmt {
	case GREY1:
		return 1 // Packed
	case GREY2:
		return 1 // Packed
	case GREY4:
		return 1 // Packed
	case GREY8, CMAP8:
		return 1
	case RGB15:
		return 2
	case RGB16:
		return 2
	case RGB24:
		return 3
	case RGBA32, ARGB32, XRGB32:
		return 4
	default:
		return 4 // Default to 4 bytes
	}
}
