package sys

import (
	"fmt"
	"os"
	"time"
)

// DeviceType enumerates the types of devices in the system
type DeviceType int

const (
	DevCons DeviceType = iota // Console device
	DevEnv                     // Environment device
	DevRoot                    // Root device
	DevNull                    // Null device
	DevZero                    // Zero device
	DevBintime                 // Time device
)

// Device is the interface that all 9front devices must implement
type Device interface {
	// Read reads data from the device into buf
	// Returns number of bytes read and any error
	Read(buf []byte) (int, error)

	// Write writes data from buf to the device
	// Returns number of bytes written and any error
	Write(buf []byte) (int, error)

	// Stat returns file information about the device
	Stat() (FileInfo, error)

	// Close closes the device (if applicable)
	Close() error
}

// FileInfo represents file metadata (similar to Plan 9's Dir structure)
type FileInfo struct {
	Name    string
	Type    uint8   // File type (0=regular, 1=device, 2=directory, etc.)
	Dev     uint32  // Device number
	QID     uint64  // Unique ID
	Mode    uint32  // Permissions
	Atime   uint32  // Access time
	Mtime   uint32  // Modify time
	Length  uint64  // File size
}

// DeviceSwitch manages all devices in the system
// Provides a unified interface for device lookup and access
type DeviceSwitch struct {
	devices map[string]Device
}

// NewDeviceSwitch creates a new device switch and initializes standard devices
func NewDeviceSwitch() *DeviceSwitch {
	ds := &DeviceSwitch{
		devices: make(map[string]Device),
	}

	// Register standard 9front devices
	ds.devices["cons"] = NewConsoleDevice()
	ds.devices["env"] = NewEnvDevice()
	ds.devices["root"] = NewRootDevice()
	ds.devices["null"] = NewNullDevice()
	ds.devices["zero"] = NewZeroDevice()
	ds.devices["bintim"] = NewBintimDevice()

	return ds
}

// Lookup finds a device by name
// Returns the device and true if found, nil and false otherwise
func (ds *DeviceSwitch) Lookup(name string) (Device, bool) {
	if ds == nil {
		return nil, false
	}

	dev, ok := ds.devices[name]
	return dev, ok
}

// Register adds a device to the device switch
func (ds *DeviceSwitch) Register(name string, dev Device) {
	if ds == nil {
		return
	}

	ds.devices[name] = dev
	fmt.Printf("[device] Registered device: %s\n", name)
}

// Unregister removes a device from the device switch
func (ds *DeviceSwitch) Unregister(name string) {
	if ds == nil {
		return
	}

	delete(ds.devices, name)
	fmt.Printf("[device] Unregistered device: %s\n", name)
}

// List returns all device names currently registered
func (ds *DeviceSwitch) List() []string {
	if ds == nil {
		return nil
	}

	names := make([]string, 0, len(ds.devices))
	for name := range ds.devices {
		names = append(names, name)
	}

	return names
}

// NullDevice discards all writes and returns EOF on reads
type NullDevice struct{}

func (nd *NullDevice) Read(buf []byte) (int, error) {
	return 0, os.ErrClosed
}

func (nd *NullDevice) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func (nd *NullDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "null",
		Type:   1, // Device
		Mode:   0666,
		Length: 0,
	}, nil
}

func (nd *NullDevice) Close() error {
	return nil
}

// ZeroDevice provides an infinite stream of zero bytes
type ZeroDevice struct{}

func (zd *ZeroDevice) Read(buf []byte) (int, error) {
	// Fill buffer with zeros
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func (zd *ZeroDevice) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func (zd *ZeroDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "zero",
		Type:   1, // Device
		Mode:   0444,
		Length: 0,
	}, nil
}

func (zd *ZeroDevice) Close() error {
	return nil
}

// BintimDevice implements /dev/bintim (Plan 9 time device)
// Returns the current time in Plan 9 format: seconds nanoseconds
type BintimDevice struct {
	location *time.Location
}

// NewBintimDevice creates a new time device
func NewBintimDevice() *BintimDevice {
	return &BintimDevice{
		location: time.Local,
	}
}

// Read returns the current time in Plan 9 format
// Plan 9 time format: "sec nsec\n" where sec is seconds since epoch and nsec is nanoseconds
func (bd *BintimDevice) Read(buf []byte) (int, error) {
	now := time.Now().In(bd.location)
	sec := now.Unix()
	nsec := now.UnixNano() % 1e9
	data := fmt.Sprintf("%d %d\n", sec, nsec)
	n := copy(buf, data)
	if n < len(data) {
		return n, nil // Return what we can, don't error
	}
	return n, nil
}

func (bd *BintimDevice) Write(buf []byte) (int, error) {
	// Time device is read-only
	return 0, fmt.Errorf("cannot write to time device")
}

func (bd *BintimDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "bintim",
		Type:   1, // Device
		Mode:   0444,
		Length: 0,
	}, nil
}

func (bd *BintimDevice) Close() error {
	return nil
}

// NewNullDevice creates a new null device
func NewNullDevice() *NullDevice {
	return &NullDevice{}
}

// NewZeroDevice creates a new zero device
func NewZeroDevice() *ZeroDevice {
	return &ZeroDevice{}
}
