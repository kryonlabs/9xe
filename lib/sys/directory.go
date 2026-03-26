package sys

import (
	"io"
	"os"
	"path/filepath"
)

// DirFile represents an open directory
type DirFile struct {
	path     string
	entries  []Dir
	position int
}

// NewDirFile opens a directory and reads its entries
func NewDirFile(path string) (*DirFile, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	dirs := make([]Dir, len(entries))
	for i, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(path, entry.Name())
		dirs[i] = *NewDirFromFile(fullPath, info)
	}

	return &DirFile{
		path:     path,
		entries:  dirs,
		position: 0,
	}, nil
}

// Read returns the next directory entry in Plan 9 Dir format
func (d *DirFile) Read(p []byte) (int, error) {
	if d.position >= len(d.entries) {
		return 0, io.EOF
	}

	dir := d.entries[d.position]
	data := dir.Marshal()

	n := copy(p, data)
	if n < len(data) {
		return n, io.ErrShortBuffer
	}

	d.position++
	return n, nil
}

// Close closes the directory
func (d *DirFile) Close() error {
	return nil
}
