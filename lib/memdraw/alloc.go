package memdraw

// Rectangle represents a rectangular area
type Rectangle struct {
	Min Point
	Max Point
}

// Point represents a 2D coordinate
type Point struct {
	X int32
	Y int32
}

// Rect returns a rectangle with the given corners
func Rect(x0, y0, x1, y1 int32) Rectangle {
	return Rectangle{Point{x0, y0}, Point{x1, y1}}
}

// Dx returns the width of the rectangle
func (r Rectangle) Dx() int32 {
	return r.Max.X - r.Min.X
}

// Dy returns the height of the rectangle
func (r Rectangle) Dy() int32 {
	return r.Max.Y - r.Min.Y
}

// AllocMemImage allocates a new memory image
func AllocMemImage(r Rectangle, chanfmt uint32) *MemImage {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return nil
	}

	depth := chantodepth(chanfmt)
	if depth == 0 {
		return nil
	}

	bytesPerRow := bytesPerRow(r, chanfmt)
	data := make([]byte, int(r.Dy())*bytesPerRow)

	return &MemImage{
		Rect:  r,
		Chan:  chanfmt,
		Data:  data,
		Depth: depth,
		Clipr: r,
	}
}

// FreeMemImage frees a memory image
func FreeMemImage(img *MemImage) {
	if img != nil {
		img.Data = nil
	}
}

// chantodepth returns the depth (bits per pixel) for a channel format
func chantodepth(chanfmt uint32) int {
	switch chanfmt {
	case GREY1:
		return 1
	case GREY2:
		return 2
	case GREY4:
		return 4
	case GREY8, CMAP8:
		return 8
	case RGB15:
		return 15
	case RGB16:
		return 16
	case RGB24:
		return 24
	case RGBA32, ARGB32, XRGB32:
		return 32
	default:
		return 0
	}
}

// bytesPerRow calculates bytes per row for an image
func bytesPerRow(r Rectangle, chanfmt uint32) int {
	d := chantodepth(chanfmt)
	if d == 0 {
		return 0
	}

	w := int(r.Dx())
	if d < 8 {
		return (w*d + 7) / 8
	}
	return w * (d / 8)
}

// byteaddr returns a pointer to the byte containing pixel (x,y)
func byteaddr(img *MemImage, x, y int) []byte {
	if x < img.Rect.Min.X || y < img.Rect.Min.Y ||
		x >= img.Rect.Max.X || y >= img.Rect.Max.Y {
		return nil
	}

	bytesPerRow := bytesPerRow(img.Rect, img.Chan)
	offset := (int(y-img.Rect.Min.Y) * bytesPerRow) +
		((int(x-img.Rect.Min.X) * img.Depth) / 8)

	return img.Data[offset:]
}
