// +build sdl2

package draw

import (
	"github.com/veandco/go-sdl2/sdl"
)

// SDL2Backend implements GraphicsBackend using SDL2
type SDL2Backend struct {
	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
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
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}

	window, err := sdl.CreateWindow("9xe - Plan 9 Display",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(width), int32(height),
		sdl.WINDOW_SHOWN)
	if err != nil {
		sdl.Quit()
		return err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		window.Destroy()
		sdl.Quit()
		return err
	}

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_STREAMING,
		int32(width), int32(height))
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		sdl.Quit()
		return err
	}

	b.window = window
	b.renderer = renderer
	b.texture = texture
	b.width = width
	b.height = height
	b.initialized = true

	return nil
}

// Update updates a region of the screen
func (b *SDL2Backend) Update(rect Rectangle, data []byte) error {
	if !b.initialized {
		return nil
	}

	// Update texture with new pixel data
	// TODO: Handle partial rectangle updates
	return b.texture.Update(nil, data, int(b.width)*4)
}

// Flush presents the rendered frame to the screen
func (b *SDL2Backend) Flush() error {
	if !b.initialized {
		return nil
	}

	// Clear the renderer
	b.renderer.Clear()

	// Copy the texture to the renderer
	b.renderer.Copy(b.texture, nil, nil)

	// Present to screen
	b.renderer.Present()

	return nil
}

// Close closes the SDL2 backend and cleans up resources
func (b *SDL2Backend) Close() error {
	if !b.initialized {
		return nil
	}

	if b.texture != nil {
		b.texture.Destroy()
	}
	if b.renderer != nil {
		b.renderer.Destroy()
	}
	if b.window != nil {
		b.window.Destroy()
	}

	sdl.Quit()
	b.initialized = false

	return nil
}
