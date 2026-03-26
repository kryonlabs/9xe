package sys

import (
	"fmt"
	"os"
)

// ConsoleDevice provides console I/O for Plan 9 programs
// In 9front, this is /dev/cons - the primary console device
// It handles keyboard input and screen output
type ConsoleDevice struct {
	input  *os.File
	output *os.File
	error  *os.File
}

// NewConsoleDevice creates a new console device
// In a real 9front system, this would be the actual hardware console
// For emulation, we use stdin/stdout/stderr
func NewConsoleDevice() *ConsoleDevice {
	return &ConsoleDevice{
		input:  os.Stdin,
		output: os.Stdout,
		error:  os.Stderr,
	}
}

// Read reads from the console (keyboard input)
func (cd *ConsoleDevice) Read(buf []byte) (int, error) {
	fmt.Printf("[dev/cons] Reading %d bytes from console\n", len(buf))
	return cd.input.Read(buf)
}

// Write writes to the console (screen output)
func (cd *ConsoleDevice) Write(buf []byte) (int, error) {
	// Don't log every write, it's too verbose
	// Just write to stdout
	return cd.output.Write(buf)
}

// Stat returns file information about the console device
func (cd *ConsoleDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "cons",
		Type:   1, // Device
		Mode:   0666, // Read/write for all
		Length: 0,
	}, nil
}

// Close closes the console device
func (cd *ConsoleDevice) Close() error {
	// Don't actually close stdin/stdout/stderr
	// In a real system, this would release the device
	return nil
}

// Control handles console control operations (for /dev/consctl)
// This would handle things like raw mode, screen size, etc.
func (cd *ConsoleDevice) Control(operation string, arg interface{}) error {
	fmt.Printf("[dev/cons] Control operation: %s\n", operation)
	return nil
}

// NewConsControlDevice creates the console control device (/dev/consctl)
type ConsControlDevice struct {
	console *ConsoleDevice
}

// NewConsControlDevice creates a new console control device
func NewConsControlDevice(cd *ConsoleDevice) *ConsControlDevice {
	return &ConsControlDevice{
		console: cd,
	}
}

func (ccd *ConsControlDevice) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("cannot read from control device")
}

func (ccd *ConsControlDevice) Write(buf []byte) (int, error) {
	// Console control commands could be processed here
	fmt.Printf("[dev/consctl] Control command: %s\n", string(buf))
	return len(buf), nil
}

func (ccd *ConsControlDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "consctl",
		Type:   1, // Device
		Mode:   0600, // Read/write for owner only
		Length: 0,
	}, nil
}

func (ccd *ConsControlDevice) Close() error {
	return nil
}
