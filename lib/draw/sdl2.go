// +build sdl2

package draw

import (
	"fmt"
	"os"
	"unsafe"

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
	// Set X11 video driver explicitly to avoid Wayland issues
	if err := os.Setenv("SDL_VIDEODRIVER", "x11"); err != nil {
		return fmt.Errorf("failed to set SDL_VIDEODRIVER: %v", err)
	}

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}

	window, err := sdl.CreateWindow("9xe - Plan 9 Display",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(width), int32(height),
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		sdl.Quit()
		return err
	}

	// Explicitly show the window
	window.Show()

	// Raise the window to the top
	window.Raise()

	// Pump events more aggressively to ensure window becomes visible
	for i := 0; i < 20; i++ {
		sdl.PumpEvents()
		sdl.Delay(10)
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

	// Debug output
	windowID, err := window.GetID()
	if err != nil {
		fmt.Printf("[SDL2] Warning: Could not get window ID: %v\n", err)
		windowID = 0
	}
	fmt.Printf("[SDL2] Window created: ID=%d, Size=%dx%d\n", windowID, width, height)
	windowFlags := window.GetFlags()
	fmt.Printf("[SDL2] Window flags: %d (SHOWN=%v, RESIZABLE=%v)\n",
		windowFlags,
		windowFlags&sdl.WINDOW_SHOWN != 0,
		windowFlags&sdl.WINDOW_RESIZABLE != 0)

	// Force window to gain focus
	window.SetTitle("9xe - Plan 9 Display (Active)")
	sdl.PumpEvents()

	return nil
}

// Update updates a region of the screen
func (b *SDL2Backend) Update(rect Rectangle, data []byte) error {
	if !b.initialized {
		return nil
	}

	if len(data) == 0 {
		return nil
	}

	// Update texture with new pixel data
	// Convert []byte to unsafe.Pointer for SDL2
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}

	// TODO: Handle partial rectangle updates
	return b.texture.Update(nil, dataPtr, int(b.width)*4)
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

	return nil
}

// PollEvents checks for pending SDL2 events and returns them
func (b *SDL2Backend) PollEvents() []Event {
	// Pump events to ensure window becomes visible
	sdl.PumpEvents()

	var events []Event

	if !b.initialized {
		return events
	}

	for {
		ev := sdl.PollEvent()
		if ev == nil {
			break
		}

		switch e := ev.(type) {
		case *sdl.QuitEvent:
			events = append(events, Event{Type: QuitEvent})

		case *sdl.MouseMotionEvent:
			events = append(events, Event{
				Type: MouseEvent,
				Mouse: MouseState{
					X:        e.X,
					Y:        e.Y,
					Buttons:  translateMouseButtons(e.State),
					Modifiers: translateModifiers(uint16(e.State)),
				},
			})

		case *sdl.MouseButtonEvent:
			button := uint32(1 << (e.Button - 1)) // SDL button 1=1, 2=2, 3=4
			if e.Type == sdl.MOUSEBUTTONUP {
				button = 0
			}
			events = append(events, Event{
				Type: MouseEvent,
				Mouse: MouseState{
					X:        e.X,
					Y:        e.Y,
					Buttons:  button,
					Modifiers: translateModifiers(uint16(e.State)),
				},
			})

		case *sdl.KeyboardEvent:
			events = append(events, Event{
				Type: KeyEvent,
				Key: KeyState{
					Code:  uint16(e.Keysym.Scancode),
					Rune:  0, // Unicode field removed in newer go-sdl2
					Down:  e.State == sdl.PRESSED,
					Ctrl:  (e.Keysym.Mod & sdl.KMOD_CTRL) != 0,
					Shift: (e.Keysym.Mod & sdl.KMOD_SHIFT) != 0,
					Alt:   (e.Keysym.Mod & sdl.KMOD_ALT) != 0,
				},
			})

		case *sdl.WindowEvent:
			if e.Event == sdl.WINDOWEVENT_RESIZED {
				events = append(events, Event{
					Type: ResizeEvent,
				})
			}
		}
	}

	return events
}

// WaitEvent blocks until an event is available
func (b *SDL2Backend) WaitEvent() Event {
	if !b.initialized {
		return Event{Type: QuitEvent}
	}

	for {
		events := b.PollEvents()
		if len(events) > 0 {
			return events[0]
		}
		// Small sleep to avoid busy waiting
		sdl.Delay(10)
	}
}

// translateMouseButtons converts SDL button state to Plan 9 format
func translateMouseButtons(state uint32) uint32 {
	var buttons uint32
	// SDL2 uses bit masks for button states
	if state&1 != 0 { // Left button
		buttons |= 1
	}
	if state&2 != 0 { // Middle button
		buttons |= 2
	}
	if state&4 != 0 { // Right button
		buttons |= 4
	}
	return buttons
}

// translateModifiers converts SDL modifier state to Plan 9 format
func translateModifiers(state uint16) uint32 {
	var mods uint32
	if state&sdl.KMOD_LSHIFT != 0 || state&sdl.KMOD_RSHIFT != 0 {
		mods |= 0x1 // Shift
	}
	if state&sdl.KMOD_LCTRL != 0 || state&sdl.KMOD_RCTRL != 0 {
		mods |= 0x2 // Ctrl
	}
	if state&sdl.KMOD_LALT != 0 || state&sdl.KMOD_RALT != 0 {
		mods |= 0x4 // Alt
	}
	return mods
}
