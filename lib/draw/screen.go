package draw

import (
	"fmt"
	"sync"
)

// ScreenManager manages Plan 9 screens
type ScreenManager struct {
	screens map[int]*Screen
	nextID  int
	mu      sync.Mutex
}

// NewScreenManager creates a new screen manager
func NewScreenManager() *ScreenManager {
	return &ScreenManager{
		screens: make(map[int]*Screen),
		nextID:  1,
	}
}

// AllocScreen allocates a new screen
func (sm *ScreenManager) AllocScreen(image, fill *Image, public bool) (*Screen, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if image == nil {
		return nil, fmt.Errorf("image cannot be nil")
	}

	screen := &Screen{
		ID:      sm.nextID,
		Image:   image,
		Fill:    fill,
		Public:  public,
		Windows: make([]*Window, 0),
	}

	// Mark the image as a screen image
	image.Screen = screen

	sm.screens[sm.nextID] = screen
	sm.nextID++

	return screen, nil
}

// LookupScreen finds a screen by ID
func (sm *ScreenManager) LookupScreen(id int) (*Screen, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	screen, ok := sm.screens[id]
	return screen, ok
}

// FreeScreen frees a screen
func (sm *ScreenManager) FreeScreen(id int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	screen, ok := sm.screens[id]
	if !ok {
		return fmt.Errorf("screen %d not found", id)
	}

	// Clear screen reference from image
	if screen.Image != nil {
		screen.Image.Screen = nil
	}

	delete(sm.screens, id)
	return nil
}

// GetScreens returns all screens
func (sm *ScreenManager) GetScreens() []*Screen {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	screens := make([]*Screen, 0, len(sm.screens))
	for _, screen := range sm.screens {
		screens = append(screens, screen)
	}

	return screens
}

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

// AllocWindow allocates a new window on a screen (for use by ScreenManager)
func (sm *ScreenManager) AllocWindow(screenID int, r Rectangle) (*Window, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	screen, ok := sm.screens[screenID]
	if !ok {
		return nil, fmt.Errorf("screen %d not found", screenID)
	}

	window, err := screen.CreateWindow(r)
	if err != nil {
		return nil, err
	}
	return window, nil
}

// FreeWindow frees a window
func (sm *ScreenManager) FreeWindow(screenID, windowID int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	screen, ok := sm.screens[screenID]
	if !ok {
		return fmt.Errorf("screen %d not found", screenID)
	}

	// Find and remove the window
	for i, win := range screen.Windows {
		if win.ID == windowID {
			screen.Windows = append(screen.Windows[:i], screen.Windows[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("window %d not found on screen %d", windowID, screenID)
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

// GetID returns the screen ID
func (s *Screen) GetID() int {
	return s.ID
}

// GetFill returns the fill image
func (s *Screen) GetFill() *Image {
	return s.Fill
}

// IsPublic returns whether the screen is public
func (s *Screen) IsPublic() bool {
	return s.Public
}

// GetWindows returns all windows on the screen
func (s *Screen) GetWindows() []*Window {
	return s.Windows
}

// TopToScreen brings a window to the top of the stack
func (s *Screen) TopToScreen(windowID int) error {
	for i, win := range s.Windows {
		if win.ID == windowID {
			// Move to end of slice (top of stack)
			s.Windows = append(s.Windows[:i], s.Windows[i+1:]...)
			s.Windows = append(s.Windows, win)
			return nil
		}
	}
	return fmt.Errorf("window %d not found", windowID)
}

// BottomToScreen sends a window to the bottom of the stack
func (s *Screen) BottomToScreen(windowID int) error {
	for i, win := range s.Windows {
		if win.ID == windowID {
			// Move to beginning of slice (bottom of stack)
			s.Windows = append(s.Windows[:i], s.Windows[i+1:]...)
			s.Windows = append([]*Window{win}, s.Windows...)
			return nil
		}
	}
	return fmt.Errorf("window %d not found", windowID)
}

// GetWindowRect returns the rectangle of a window
func (s *Screen) GetWindowRect(windowID int) (Rectangle, error) {
	for _, win := range s.Windows {
		if win.ID == windowID {
			return win.Rect, nil
		}
	}
	return Rectangle{}, fmt.Errorf("window %d not found", windowID)
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
