package sys

// Chan represents a Plan 9 channel (file handle)
// Based on 9front's Chan structure
// Channels are the core abstraction for all I/O in Plan 9
type Chan struct {
	Path string   // Plan 9 path
	Mode int      // Open mode (OREAD, OWRITE, ORDWR)
	Fid  uint32   // FID for 9P protocol
	Dev  interface{}  // Underlying device (if any)
	File *File    // Underlying file (if any)
	Qid  Qid      // QID of the file
}

// File represents an open file in the system
type File struct {
	Path   string
	Mode   int
	Offset int64
}

// NewChan creates a new channel
func NewChan(path string, mode int) *Chan {
	return &Chan{
		Path: path,
		Mode: mode,
		Fid:  0, // Will be assigned by 9P protocol
	}
}

// Clone creates a copy of a channel
func (c *Chan) Clone() *Chan {
	return &Chan{
		Path: c.Path,
		Mode: c.Mode,
		Fid:  c.Fid,
		Dev:  c.Dev,
		File: c.File,
		Qid:  c.Qid,
	}
}
