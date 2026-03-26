package memdraw

// MemImage represents an in-memory image
type MemImage struct {
	Rect   Rectangle
	Chan   uint32 // Pixel format
	Data   []byte // Pixel data
	Depth  int
	Flags  int
	Clipr  Rectangle
}

// Drawing operation constants
const (
	SoverD = 0 // Source over Destination
	SinD   = 1 // Source in Destination
	SoutD  = 2 // Source out Destination
)

// Image flags
const (
	Frepl = 1 << iota // Image replicates
)

// Channel format constants
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
