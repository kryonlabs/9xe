package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	fmt.Println("Initializing SDL2...")
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		fmt.Printf("SDL2 Init failed: %v\n", err)
		return
	}
	defer sdl.Quit()

	fmt.Println("Creating window...")
	window, err := sdl.CreateWindow("TEST WINDOW", 100, 100, 400, 300, sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Printf("Window creation failed: %v\n", err)
		return
	}
	defer window.Destroy()

	fmt.Println("Window created successfully!")
	fmt.Println("Keeping it open for 5 seconds...")

	for i := 0; i < 50; i++ {
		sdl.PumpEvents()
		sdl.Delay(100)
		if i % 10 == 0 {
			fmt.Printf("Window open for %d seconds...\n", i/10)
		}
	}

	fmt.Println("Done!")
}
