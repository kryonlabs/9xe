package draw

import (
	"fmt"
	"log"
)

// ninep integration helpers
// This file provides integration points for the ninep library
// The actual ninep server will be integrated into 9xe's syscall layer

// DrawFilesystem represents the draw device filesystem
type DrawFilesystem struct {
	server *DrawServer
	files  map[string]*VirtualFile
}

// NewDrawFilesystem creates a new draw filesystem
func NewDrawFilesystem(server *DrawServer) *DrawFilesystem {
	fs := &DrawFilesystem{
		server: server,
		files:  make(map[string]*VirtualFile),
	}

	// Create /dev/draw
	drawFile := NewVirtualFile("draw", server)
	fs.files["/dev/draw"] = drawFile

	return fs
}

// Open opens a file in the filesystem
func (fs *DrawFilesystem) Open(path string) (*VirtualFile, error) {
	file, ok := fs.files[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return file, nil
}

// Stat returns file information
func (fs *DrawFilesystem) Stat(path string) (uint32, uint32, error) {
	file, ok := fs.files[path]
	if !ok {
		return 0, 0, fmt.Errorf("file not found: %s", path)
	}
	return file.Stat()
}

// CreateVirtualFile creates a new virtual file
func (fs *DrawFilesystem) CreateVirtualFile(path string, file *VirtualFile) {
	fs.files[path] = file
	log.Printf("Created virtual file: %s", path)
}

// GetServer returns the filesystem's server
func (fs *DrawFilesystem) GetServer() *DrawServer {
	return fs.server
}
