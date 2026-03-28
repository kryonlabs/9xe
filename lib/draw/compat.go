package draw

// +build sdl2

import (
	"fmt"
	"os"
)

// Plan 9 draw library compatibility functions
// These intercept calls from Plan 9 binaries and translate them to our SDL2 backend

// DrawlibState represents the state of the Plan 9 draw library
type DrawlibState struct {
	initialized bool
	display     string
	screen      *Screen
	imageID     int
	screenID    int
	drawFile    *os.File // Actual file for draw connection
}

var globalDrawlib DrawlibState

// _drawconnect is the Plan 9 function to connect to draw server
// We intercept this and initialize our SDL2 backend instead
func _drawconnect(display string) int {
	fmt.Printf("[DRAWLIB] _drawconnect(%q)\n", display)

	globalDrawlib.display = display
	globalDrawlib.initialized = true
	globalDrawlib.imageID = 1
	globalDrawlib.screenID = 1

	// Try to open the actual draw device
	drawFile, err := os.OpenFile("/dev/draw/new", os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("[DRAWLIB] Failed to open draw device: %v\n", err)
		fmt.Printf("[DRAWLIB] Creating dummy draw file\n")
		// Create a dummy file
		drawFile, _ = os.CreateTemp("", "draw-connection")
	}

	globalDrawlib.drawFile = drawFile

	// Return the actual file descriptor
	if drawFile != nil {
		return int(drawFile.Fd())
	}

	// Fallback to fd 3
	return 3
}

// _allocimage allocates an image in Plan 9 draw format
func _allocimage(id int, screenID int, refresh int, ch uint32, repl int, r Rectangle, clipr Rectangle, color uint32) int {
	fmt.Printf("[DRAWLIB] _allocimage(id=%d, screen=%d, ch=0x%x, r=%v)\n", id, screenID, ch, r)

	// Create image using our image manager
	img, err := globalImageManager.AllocImage(r, ch, repl != 0, color)
	if err != nil {
		fmt.Printf("[DRAWLIB] alloc failed: %v\n", err)
		return -1
	}

	// Store with the requested ID
	img.ID = id
	globalImageManager.images[id] = img

	return id
}

// _draw performs drawing operation
func _draw(dstID int, screenID int, srcID int, maskID int, r Rectangle, p Point) int {
	fmt.Printf("[DRAWLIB] _draw(dst=%d, src=%d, r=%v, p=%v)\n", dstID, srcID, r, p)

	// Look up images
	dst, ok := globalImageManager.LookupImage(dstID)
	if !ok {
		fmt.Printf("[DRAWLIB] dst image %d not found\n", dstID)
		return -1
	}

	src, ok := globalImageManager.LookupImage(srcID)
	if !ok {
		fmt.Printf("[DRAWLIB] src image %d not found\n", srcID)
		return -1
	}

	var mask *Image
	if maskID != 0 {
		mask, ok = globalImageManager.LookupImage(maskID)
		if !ok {
			fmt.Printf("[DRAWLIB] mask image %d not found\n", maskID)
			return -1
		}
	}

	// Perform draw operation
	err := Draw(dst, r, src, mask, p)
	if err != nil {
		fmt.Printf("[DRAWLIB] draw failed: %v\n", err)
		return -1
	}

	// Flush to screen
	if globalDrawlib.screen != nil && globalDrawlib.screen.Backend != nil {
		globalDrawlib.screen.Backend.Flush()
	}

	return 0
}

// _freeimage frees an image
func _freeimage(id int) int {
	fmt.Printf("[DRAWLIB] _freeimage(%d)\n", id)
	return 0
}

// _allocscreen allocates a screen
func _allocscreen(imageID int, fillID int, public int) int {
	fmt.Printf("[DRAWLIB] _allocscreen(image=%d, fill=%d, public=%d)\n", imageID, fillID, public)

	image, ok := globalImageManager.LookupImage(imageID)
	if !ok {
		fmt.Printf("[DRAWLIB] image %d not found\n", imageID)
		return -1
	}

	var fill *Image
	if fillID != 0 {
		fill, ok = globalImageManager.LookupImage(fillID)
		if !ok {
			fmt.Printf("[DRAWLIB] fill image %d not found\n", fillID)
			return -1
		}
	}

	screen, err := globalScreenManager.AllocScreen(image, fill, public != 0)
	if err != nil {
		fmt.Printf("[DRAWLIB] screen alloc failed: %v\n", err)
		return -1
	}

	globalDrawlib.screen = screen
	return screen.ID
}

// _allocwindow allocates a window on a screen
func _allocwindow(screenID int, r Rectangle) int {
	fmt.Printf("[DRAWLIB] _allocwindow(screen=%d, r=%v)\n", screenID, r)

	screen, ok := globalScreenManager.LookupScreen(screenID)
	if !ok {
		fmt.Printf("[DRAWLIB] screen %d not found\n", screenID)
		return -1
	}

	window, err := screen.CreateWindow(r)
	if err != nil {
		fmt.Printf("[DRAWLIB] window alloc failed: %v\n", err)
		return -1
	}

	return window.ID
}

// _readbitmap reads bitmap data
func _readbitmap(id int, r Rectangle, data []byte) int {
	fmt.Printf("[DRAWLIB] _readbitmap(id=%d, r=%v, len=%d)\n", id, r, len(data))

	image, ok := globalImageManager.LookupImage(id)
	if !ok {
		fmt.Printf("[DRAWLIB] image %d not found\n", id)
		return -1
	}

	err := image.LoadData(data, r)
	if err != nil {
		fmt.Printf("[DRAWLIB] load data failed: %v\n", err)
		return -1
	}

	return 0
}

// _writebitmap writes bitmap data
func _writebitmap(id int, r Rectangle) ([]byte, error) {
	fmt.Printf("[DRAWLIB] _writebitmap(id=%d, r=%v)\n", id, r)

	image, ok := globalImageManager.LookupImage(id)
	if !ok {
		return nil, fmt.Errorf("image %d not found", id)
	}

	return image.GetData(r)
}

// _drawsetdebug sets debug mode
func _drawsetdebug(debug int) int {
	fmt.Printf("[DRAWLIB] _drawsetdebug(%d)\n", debug)
	return 0
}

// _drawgetimage gets an image reference
func _drawgetimage(id int) *Image {
	image, ok := globalImageManager.LookupImage(id)
	if !ok {
		return nil
	}
	return image
}

// Export global instances for compatibility layer
var (
	globalImageManager  = NewImageManager()
	globalScreenManager  = NewScreenManager()
)

// Initialize the global managers
func init() {
	globalImageManager = NewImageManager()
	globalScreenManager = NewScreenManager()
}
