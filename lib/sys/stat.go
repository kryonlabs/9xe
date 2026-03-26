package sys

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// Plan 9 Qid type
type Qid struct {
	Path uint64
	Vers uint32
	Type uint8
}

// Plan 9 Dir structure (for STAT/FSTAT)
type Dir struct {
	Name   string
	Type   uint8
	Dev    uint32
	Qid    Qid
	Mode   uint32
	Atime  uint32
	Mtime  uint32
	Length uint64
	Uid    string // User ID (for extended format)
	Gid    string // Group ID (for extended format)
	Muid   string // Modified user ID (for extended format)
}

// Sizeof returns the size of Dir in Plan 9 wire format
func (d *Dir) Sizeof() int {
	nameLen := len(d.Name)
	// Basic format: size[2] type[2] dev[4] qid[13] mode[4] atime[4] mtime[4] length[8] nameLen[2] name[s]
	totalSize := 2 + 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + nameLen
	return totalSize
}

// Marshal converts Dir to Plan 9 wire format
func (d *Dir) Marshal() []byte {
	totalSize := d.Sizeof()
	buf := make([]byte, totalSize)

	// Size (ushort)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(totalSize))

	// Type (ushort)
	binary.LittleEndian.PutUint16(buf[2:4], uint16(d.Type))

	// Dev (uint)
	binary.LittleEndian.PutUint32(buf[4:8], d.Dev)

	// Qid (13 bytes: path[8] vers[4] type[1])
	binary.LittleEndian.PutUint64(buf[8:16], d.Qid.Path)
	binary.LittleEndian.PutUint32(buf[16:20], d.Qid.Vers)
	buf[20] = d.Qid.Type

	// Mode (uint)
	binary.LittleEndian.PutUint32(buf[21:25], d.Mode)

	// Atime (uint)
	binary.LittleEndian.PutUint32(buf[25:29], d.Atime)

	// Mtime (uint)
	binary.LittleEndian.PutUint32(buf[29:33], d.Mtime)

	// Length (uvlong)
	binary.LittleEndian.PutUint64(buf[33:41], d.Length)

	// Name length (ushort)
	nameLen := len(d.Name)
	binary.LittleEndian.PutUint16(buf[41:43], uint16(nameLen))

	// Name string
	copy(buf[43:], d.Name)

	return buf
}

// UnmarshalDir converts Plan 9 wire format to Dir structure
func UnmarshalDir(data []byte) (*Dir, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short for Dir")
	}

	// Read total size
	size := binary.LittleEndian.Uint16(data[0:2])
	if len(data) < int(size) {
		return nil, fmt.Errorf("incomplete Dir: need %d bytes, got %d", size, len(data))
	}

	dir := &Dir{}

	// Type (ushort)
	dir.Type = uint8(binary.LittleEndian.Uint16(data[2:4]))

	// Dev (uint)
	dir.Dev = binary.LittleEndian.Uint32(data[4:8])

	// Qid (13 bytes: path[8] vers[4] type[1])
	dir.Qid.Path = binary.LittleEndian.Uint64(data[8:16])
	dir.Qid.Vers = binary.LittleEndian.Uint32(data[16:20])
	dir.Qid.Type = data[20]

	// Mode (uint)
	dir.Mode = binary.LittleEndian.Uint32(data[21:25])

	// Atime (uint)
	dir.Atime = binary.LittleEndian.Uint32(data[25:29])

	// Mtime (uint)
	dir.Mtime = binary.LittleEndian.Uint32(data[29:33])

	// Length (uvlong)
	dir.Length = binary.LittleEndian.Uint64(data[33:41])

	// Name length (ushort)
	nameLen := binary.LittleEndian.Uint16(data[41:43])

	// Validate name length fits in buffer
	if 43+int(nameLen) > len(data) {
		return nil, fmt.Errorf("name too long: %d bytes, buffer has %d", nameLen, len(data)-43)
	}

	// Name string
	dir.Name = string(data[43 : 43+nameLen])

	return dir, nil
}

// Plan 9 file types
const (
	MTYPEFILE   = 0
	MTYPEDIR    = 0x80
	MTYPEDevice = 0x40
)

// Plan 9 permission bits
const (
	DMREAD  = 0400
	DMWRITE = 0200
	DMEXEC  = 0100
)

// handleStat implements STAT syscall (18)
func (k *Kernel) handleStat(mu unicorn.Unicorn, rsp uint64) {
	namePtr, _ := readArg(mu, rsp, 0)
	edir, _ := readArg(mu, rsp, 1)
	nedd, _ := readArg(mu, rsp, 2)

	path, err := readString(mu, namePtr, 1024)
	if err != nil {
		k.setError(mu, "bad path ptr")
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	dir := statToDir(info, path)
	data, n := marshalDir(dir)

	if uint64(n) > nedd {
		setReturn(mu, uint64(n))
		return
	}

	mu.MemWrite(edir, data)
	setReturn(mu, uint64(n))
}

// handleFstat implements FSTAT syscall (43)
func (k *Kernel) handleFstat(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	edir, _ := readArg(mu, rsp, 1)
	nedd, _ := readArg(mu, rsp, 2)

	// Check if it's a virtual file
	if _, ok := k.lookupVFile(int(fd)); ok {
		// For virtual files, return minimal stat
		dir := &Dir{
			Type: MTYPEFILE,
			Mode: DMREAD | DMWRITE,
		}
		data, n := marshalDir(dir)

		if uint64(n) > nedd {
			setReturn(mu, uint64(n))
			return
		}

		mu.MemWrite(edir, data)
		setReturn(mu, uint64(n))
		return
	}

	// Regular file
	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}

	info, err := f.Stat()
	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	path := f.Name()
	dir := statToDir(info, path)
	data, n := marshalDir(dir)

	if uint64(n) > nedd {
		setReturn(mu, uint64(n))
		return
	}

	mu.MemWrite(edir, data)
	setReturn(mu, uint64(n))
}

// handle_Fstat implements _FSTAT syscall (11)
func (k *Kernel) handle_Fstat(mu unicorn.Unicorn, rsp uint64) {
	// Same as FSTAT for now
	k.handleFstat(mu, rsp)
}

// NewDirFromFile creates a Plan 9 Dir from os.FileInfo
// This is the preferred method for creating Dir structures from host files
func NewDirFromFile(path string, info os.FileInfo) *Dir {
	dir := &Dir{
		Name:   info.Name(),
		Length: uint64(info.Size()),
		Atime:  uint32(info.ModTime().Unix()),
		Mtime:  uint32(info.ModTime().Unix()),
		Uid:    "glenda", // Default Plan 9 user
		Gid:    "glenda", // Default Plan 9 group
		Muid:   "",
	}

	// Set type
	if info.IsDir() {
		dir.Type = MTYPEDIR
		dir.Mode = DMREAD | DMEXEC | 0111 // Directory is read/execute
	} else {
		dir.Type = MTYPEFILE
		dir.Mode = DMREAD | DMWRITE // Default to read/write
	}

	// Set permissions from file mode
	mode := info.Mode()
	if mode&0400 != 0 {
		dir.Mode |= DMREAD
	}
	if mode&0200 != 0 {
		dir.Mode |= DMWRITE
	}
	if mode&0100 != 0 {
		dir.Mode |= DMEXEC
	}

	// Set Qid - use modification time as unique path
	dir.Qid.Path = uint64(info.ModTime().UnixNano())
	dir.Qid.Vers = 0
	dir.Qid.Type = uint8(dir.Type)

	return dir
}

// statToDir converts os.FileInfo to Plan 9 Dir (legacy function, uses NewDirFromFile)
func statToDir(info os.FileInfo, path string) *Dir {
	return NewDirFromFile(path, info)
}

// marshalDir converts Dir to Plan 9 wire format
// Format: size[2] type[2] dev[4] qid[13] mode[4] atime[4] mtime[4] length[8] name[s]
func marshalDir(dir *Dir) ([]byte, int) {
	// Calculate size
	nameLen := len(dir.Name)
	totalSize := 2 + 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + nameLen

	buf := make([]byte, totalSize)

	// Size (ushort)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(totalSize))

	// Type (ushort)
	binary.LittleEndian.PutUint16(buf[2:4], uint16(dir.Type))

	// Dev (uint)
	binary.LittleEndian.PutUint32(buf[4:8], dir.Dev)

	// Qid (13 bytes: path[8] vers[4] type[1])
	binary.LittleEndian.PutUint64(buf[8:16], dir.Qid.Path)
	binary.LittleEndian.PutUint32(buf[16:20], dir.Qid.Vers)
	buf[20] = dir.Qid.Type

	// Mode (uint)
	binary.LittleEndian.PutUint32(buf[21:25], dir.Mode)

	// Atime (uint)
	binary.LittleEndian.PutUint32(buf[25:29], dir.Atime)

	// Mtime (uint)
	binary.LittleEndian.PutUint32(buf[29:33], dir.Mtime)

	// Length (uvlong)
	binary.LittleEndian.PutUint64(buf[33:41], dir.Length)

	// Name length (ushort)
	binary.LittleEndian.PutUint16(buf[41:43], uint16(nameLen))

	// Name string
	copy(buf[43:], dir.Name)

	return buf, totalSize
}
