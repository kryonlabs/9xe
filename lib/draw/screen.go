package draw

// NewScreen creates a new screen with the given backend
func NewScreen(width, height int, backend GraphicsBackend) (*Screen, error) {
	screen := &Screen{
		Windows: make([]*Window, 0),
		Backend: backend,
	}

	// Create the screen image
	rect := Rect(0, 0, int32(width), int32(height))
	screen.Image = &Image{
		ID:     0, // Screen is always ID 0
		Rect:   rect,
		Chan:   XRGB32,
		ClipR:  rect,
		Ref:    1,
		Screen: screen,
	}

	// Allocate pixel data
	screen.Image.Data = make([]byte, width*height*4)

	// Initialize the backend
	if backend != nil {
		if err := backend.CreateWindow(width, height); err != nil {
			return nil, err
		}
	}

	screen.ID = 1

	return screen, nil
}

// CreateWindow creates a new window on the screen
func (s *Screen) CreateWindow(rect Rectangle) (*Window, error) {
	window := &Window{
		ID:      len(s.Windows) + 1,
		Rect:    rect,
		Screen:  s,
		Visible: true,
	}

	// Create the window's image
	window.Image = &Image{
		ID:     window.ID,
		Rect:   rect,
		Chan:   XRGB32,
		ClipR:  rect,
		Ref:    1,
		Screen: s,
	}

	// Allocate pixel data
	width := int(rect.Dx())
	height := int(rect.Dy())
	window.Image.Data = make([]byte, width*height*4)

	s.Windows = append(s.Windows, window)

	return window, nil
}

// Flush flushes the screen to the display
func (s *Screen) Flush() error {
	if s.Backend != nil {
		// Update the backend with screen data
		if err := s.Backend.Update(s.Image.Rect, s.Image.Data); err != nil {
			return err
		}
		// Present to screen
		return s.Backend.Flush()
	}
	return nil
}

// GetImage returns the screen's image
func (s *Screen) GetImage() *Image {
	return s.Image
}

// AddImage adds an image to the screen's image list
func (s *Screen) AddImage(img *Image) {
	// Images are stored in the DrawClient, not the screen
	// This is a no-op for now
}

// RemoveImage removes an image from the screen
func (s *Screen) RemoveImage(id int) {
	// Images are managed by the DrawClient
	// This is a no-op for now
}
