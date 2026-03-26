package draw

import (
	"fmt"
	"io"
)

// VirtualFile represents a virtual file in the draw filesystem
type VirtualFile struct {
	Name   string
	Qid    uint32
	Mode   uint32
	Data   []byte
	Server *DrawServer
	Client *DrawClient

	// For pipes and special files
	Reader io.Reader
	Writer io.Writer
	Closer io.Closer
}

// NewVirtualFile creates a new virtual file
func NewVirtualFile(name string, server *DrawServer) *VirtualFile {
	return &VirtualFile{
		Name:   name,
		Qid:    0,
		Mode:   0666,
		Server: server,
	}
}

// Read reads from the virtual file
func (f *VirtualFile) Read(p []byte) (int, error) {
	// If this file has a custom reader, use it
	if f.Reader != nil {
		return f.Reader.Read(p)
	}

	// For /dev/draw, reading returns the draw protocol initialization
	// TODO: Implement proper draw protocol initialization
	return 0, nil
}

// Write writes to the virtual file
func (f *VirtualFile) Write(p []byte) (int, error) {
	// If this file has a custom writer, use it
	if f.Writer != nil {
		return f.Writer.Write(p)
	}

	if f.Client == nil {
		if f.Server != nil {
			f.Client = f.Server.NewClient()
		} else {
			return 0, fmt.Errorf("no server attached")
		}
	}

	return f.Client.HandleCommand(p)
}

// Close closes the virtual file
func (f *VirtualFile) Close() error {
	// If this file has a custom closer, use it
	if f.Closer != nil {
		return f.Closer.Close()
	}

	if f.Client != nil && f.Server != nil {
		// Client will be cleaned up by server
		f.Client = nil
	}
	return nil
}

// Stat returns file information
func (f *VirtualFile) Stat() (uint32, uint32, error) {
	return f.Mode, f.Qid, nil
}
