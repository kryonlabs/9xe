package draw

// Refresh refreshes a window's content to the screen
func (w *Window) Refresh() error {
	if !w.Visible || w.Image == nil || w.Screen == nil {
		return nil
	}

	screenImg := w.Screen.Image
	if screenImg == nil {
		return nil
	}

	// Copy window content to screen
	// This is a simplified implementation
	// TODO: Implement proper layer composition

	srcY := w.Image.Rect.Min.Y
	for y := w.Rect.Min.Y; y < w.Rect.Max.Y; y++ {
		if y < screenImg.Rect.Min.Y || y >= screenImg.Rect.Max.Y {
			continue
		}

		srcX := w.Image.Rect.Min.X
		for x := w.Rect.Min.X; x < w.Rect.Max.X; x++ {
			if x < screenImg.Rect.Min.X || x >= screenImg.Rect.Max.X {
				continue
			}

			// Copy pixel
			srcOff := pixelOffset(w.Image, srcX, srcY)
			dstOff := pixelOffset(screenImg, x, y)

			if srcOff+4 <= len(w.Image.Data) && dstOff+4 <= len(screenImg.Data) {
				copy(screenImg.Data[dstOff:dstOff+4], w.Image.Data[srcOff:srcOff+4])
			}

			srcX++
		}
		srcY++
	}

	return nil
}

// Flush flushes the window to the screen
func (w *Window) Flush() error {
	if err := w.Refresh(); err != nil {
		return err
	}
	return w.Screen.Flush()
}

// pixelOffset calculates the byte offset for a pixel in an image
func pixelOffset(img *Image, x, y int32) int {
	width := int(img.Rect.Dx())
	return (int(y-img.Rect.Min.Y)*width + int(x-img.Rect.Min.X)) * 4
}

// Show makes the window visible
func (w *Window) Show() {
	w.Visible = true
}

// Hide hides the window
func (w *Window) Hide() {
	w.Visible = false
}

// Move moves the window to a new position
func (w *Window) Move(x, y int32) {
	width := w.Rect.Dx()
	height := w.Rect.Dy()
	w.Rect.Min.X = x
	w.Rect.Min.Y = y
	w.Rect.Max.X = x + width
	w.Rect.Max.Y = y + height
}

// Resize resizes the window
func (w *Window) Resize(width, height int32) {
	w.Rect.Max.X = w.Rect.Min.X + width
	w.Rect.Max.Y = w.Rect.Min.Y + height

	// Reallocate pixel data
	w.Image.Data = make([]byte, int(width*height)*4)
}

// GetImage returns the window's image
func (w *Window) GetImage() *Image {
	return w.Image
}
