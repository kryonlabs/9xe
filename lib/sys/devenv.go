package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// EnvDevice provides access to environment variables through a file interface
// In 9front, environment variables are files in /env
// Reading from /env/VARNAME returns the value
// Writing to /env/VARNAME sets the value
type EnvDevice struct {
	rootfs *RootFS // Root filesystem for path resolution
	mu     sync.RWMutex
}

// NewEnvDevice creates a new environment device
func NewEnvDevice() *EnvDevice {
	return &EnvDevice{
		mu: sync.RWMutex{},
	}
}

// SetRootFS sets the root filesystem for path resolution
func (ed *EnvDevice) SetRootFS(rootfs *RootFS) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.rootfs = rootfs
}

// Read reads an environment variable's value
// For /env/VARNAME, buf contains the variable name
// Returns the value of the environment variable
func (ed *EnvDevice) Read(buf []byte) (int, error) {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// Extract variable name from path
	varName := strings.TrimRight(string(buf), "\x00")

	if ed.rootfs == nil {
		return 0, fmt.Errorf("rootfs not initialized")
	}

	// Look up the environment variable
	value := ed.rootfs.GetEnv(varName)
	if value == "" {
		return 0, os.ErrNotExist
	}

	// Return the value
	n := copy(buf, value)
	if n < len(value) {
		// Buffer was too small, this is okay
	}

	return n, nil
}

// Write sets an environment variable's value
// For /env/VARNAME, buf contains the new value
func (ed *EnvDevice) Write(buf []byte) (int, error) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	if ed.rootfs == nil {
		return 0, fmt.Errorf("rootfs not initialized")
	}

	// Extract variable name and value
	// In 9front, the path includes the variable name
	// For simplicity, we'll parse it as "VARNAME=value"

	str := strings.TrimRight(string(buf), "\x00")

	parts := strings.SplitN(str, "=", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid env format, expected VAR=value")
	}

	varName := parts[0]
	value := parts[1]

	// Set the environment variable
	ed.rootfs.SetEnv(varName, value)

	fmt.Printf("[dev/env] Set %s=%s\n", varName, value)

	return len(buf), nil
}

// Stat returns file information about the env device
func (ed *EnvDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "env",
		Type:   2, // Directory (env is a directory containing files)
		Mode:   0555, // Read/execute
		Length: 0,
	}, nil
}

// Close closes the environment device
func (ed *EnvDevice) Close() error {
	return nil
}

// ListEnv lists all environment variables (for directory reading)
func (ed *EnvDevice) ListEnv() map[string]string {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	if ed.rootfs == nil {
		return make(map[string]string)
	}

	return ed.rootfs.GetAllEnv()
}

// RootDevice provides the root directory of the filesystem
type RootDevice struct {
	rootfs *RootFS
}

// NewRootDevice creates a new root device
func NewRootDevice() *RootDevice {
	return &RootDevice{}
}

func (rd *RootDevice) SetRootFS(rootfs *RootFS) {
	rd.rootfs = rootfs
}

func (rd *RootDevice) Read(buf []byte) (int, error) {
	// Root is a directory, can't directly read
	return 0, fmt.Errorf("is a directory")
}

func (rd *RootDevice) Write(buf []byte) (int, error) {
	return 0, fmt.Errorf("cannot write to root directory")
}

func (rd *RootDevice) Stat() (FileInfo, error) {
	return FileInfo{
		Name:   "root",
		Type:   2, // Directory
			Mode:   0555, // Read/execute
		Length: 0,
	}, nil
}

func (rd *RootDevice) Close() error {
	return nil
}

// ResolvePath resolves a path through the root filesystem
// This is used for path resolution in device operations
func (rd *RootDevice) ResolvePath(path string) string {
	if rd.rootfs == nil {
		return path
	}

	// Use RootFS to resolve to local path
	return rd.rootfs.LocalPath(path)
}

// Helper function to check if a path exists
func (rd *RootDevice) PathExists(path string) bool {
	if rd.rootfs == nil {
		return false
	}

	resolved := rd.rootfs.LocalPath(path)
	_, err := os.Stat(resolved)
	return err == nil
}

// ListDir lists the contents of a directory in the root filesystem
func (rd *RootDevice) ListDir(path string) ([]string, error) {
	if rd.rootfs == nil {
		return nil, fmt.Errorf("rootfs not initialized")
	}

	resolved := rd.rootfs.LocalPath(path)

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

// IsDir checks if a path is a directory
func (rd *RootDevice) IsDir(path string) bool {
	if rd.rootfs == nil {
		return false
	}

	resolved := rd.rootfs.LocalPath(path)
	info, err := os.Stat(resolved)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// GetFileInfo returns file info for a path
func (rd *RootDevice) GetFileInfo(path string) (FileInfo, error) {
	if rd.rootfs == nil {
		return FileInfo{}, fmt.Errorf("rootfs not initialized")
	}

	resolved := rd.rootfs.LocalPath(path)
	info, err := os.Stat(resolved)
	if err != nil {
		return FileInfo{}, err
	}

	// Convert os.FileInfo to our FileInfo
	var fileType uint8
	if info.IsDir() {
		fileType = 2 // Directory
	} else if info.Mode().IsRegular() {
		fileType = 0 // Regular file
	} else {
		fileType = 1 // Device or special file
	}

	return FileInfo{
		Name:   filepath.Base(path),
		Type:   fileType,
		Mode:   uint32(info.Mode().Perm()),
		Length: uint64(info.Size()),
	}, nil
}
