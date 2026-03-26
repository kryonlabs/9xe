// +build !sdl2

package draw

import "fmt"

// SDL2Backend implements GraphicsBackend using SDL2
type SDL2Backend struct {
	width    int
	height   int
	initialized bool
}

// NewSDL2Backend creates a new SDL2 backend
func NewSDL2Backend() *SDL2Backend {
	return &SDL2Backend{
		initialized: false,
	}
}

// CreateWindow creates a window with the given dimensions
func (b *SDL2Backend) CreateWindow(width, height int) error {
	b.width = width
	b.height = height
	b.initialized = true
	fmt.Printf("[9xe] Graphics backend not available (build with -tags sdl2)\n")
	return nil
}

// Update updates a region of the screen
func (b *SDL2Backend) Update(rect Rectangle, data []byte) error {
	return nil
}

// Flush presents the rendered frame to the screen
func (b *SDL2Backend) Flush() error {
	return nil
}

// Close closes the SDL2 backend and cleans up resources
func (b *SDL2Backend) Close() error {
	b.initialized = false
	return nil
}
