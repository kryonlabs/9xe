package draw

// Plan 9 draw protocol types and constants

// Point represents a 2D coordinate
type Point struct {
	X int32
	Y int32
}

// Rectangle represents a rectangular area
type Rectangle struct {
	Min Point
	Max Point
}

// Rect returns a rectangle with the given corners
func Rect(x0, y0, x1, y1 int32) Rectangle {
	return Rectangle{Point{x0, y0}, Point{x1, y1}}
}

// Inset returns the rectangle inset by inset units
func (r Rectangle) Inset(inset int32) Rectangle {
	return Rectangle{
		Point{r.Min.X + inset, r.Min.Y + inset},
		Point{r.Max.X - inset, r.Max.Y - inset},
	}
}

// Dx returns the width of the rectangle
func (r Rectangle) Dx() int32 {
	return r.Max.X - r.Min.X
}

// Dy returns the height of the rectangle
func (r Rectangle) Dy() int32 {
	return r.Max.Y - r.Min.Y
}

// Channel format constants from Plan 9
const (
	GREY1  = 0x01 // 1-bit greyscale
	GREY2  = 0x02 // 2-bit greyscale
	GREY4  = 0x04 // 4-bit greyscale
	GREY8  = 0x08 // 8-bit greyscale
	CMAP8  = 0x09 // 8-bit color-mapped
	RGB15  = 0x10 // 15-bit RGB
	RGB16  = 0x11 // 16-bit RGB
	RGB24  = 0x12 // 24-bit RGB
	RGBA32 = 0x13 // 32-bit RGBA
	ARGB32 = 0x14 // 32-bit ARGB (Plan 9 default)
	XRGB32 = 0x15 // 32-bit XRGB
)

// Draw protocol command opcodes
const (
	// Core commands
	AllocOpcode  = 'b' // Allocate image
	DrawOpcode   = 'd' // Draw operation
	ClipOpcode   = 'c' // Set clip/repl
	FreeOpcode   = 'f' // Free image
	FlushOpcode  = 'v' // Flush to screen
	WriteOpcode  = 'y' // Write pixel data

	// Extended commands (Phase 3)
	EllipseOpcode = 'e' // Draw ellipse
)

// Image represents a Plan 9 draw image
type Image struct {
	ID      int
	Rect    Rectangle
	Chan    uint32 // Pixel format
	Data    []byte // Pixel data
	ClipR   Rectangle
	Repl    bool
	Ref     int // Reference count
	Screen  *Screen // If this is a screen image
}

// Screen represents a virtual display
type Screen struct {
	ID      int
	Image   *Image
	Windows []*Window
	Backend GraphicsBackend
}

// Window represents a window on a screen
type Window struct {
	ID      int
	Image   *Image
	Rect    Rectangle
	Screen  *Screen
	Visible bool
}

// GraphicsBackend is the interface for rendering to the host OS
type GraphicsBackend interface {
	CreateWindow(width, height int) error
	Update(rect Rectangle, data []byte) error
	Flush() error
	Close() error
}
