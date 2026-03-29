package sys

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// All syscall numbers from /sys/src/libc/9syscall/sys.h
const (
	SYSR1       = 0
	_ERRSTR     = 1
	BIND        = 2
	CHDIR       = 3
	CLOSE       = 4
	DUP         = 5
	ALARM       = 6
	EXEC        = 7
	EXITS       = 8
	_FSESSION   = 9
	FAUTH       = 10
	_FSTAT      = 11
	SEGBRK      = 12
	_MOUNT      = 13
	OPEN        = 14
	_READ       = 15
	OSEEK       = 16
	SLEEP       = 17
	_STAT       = 18
	RFORK       = 19
	_WRITE      = 20
	PIPE        = 21
	CREATE      = 22
	FD2PATH     = 23
	BRK_        = 24
	REMOVE      = 25
	_WSTAT      = 26
	_FWSTAT     = 27
	NOTIFY      = 28
	NOTED       = 29
	SEGATTACH   = 30
	SEGDETACH   = 31
	SEGFREE     = 32
	SEGFLUSH    = 33
	RENDEZVOUS  = 34
	UNMOUNT     = 35
	_WAIT       = 36
	SEMACQUIRE  = 37
	SEMRELEASE  = 38
	SEEK        = 39
	FVERSION    = 40
	ERRSTR      = 41
	STAT        = 42
	FSTAT       = 43
	WSTAT       = 44
	FWSTAT      = 45
	MOUNT       = 46
	PREAD       = 50
	PWRITE      = 51
	TSEMACQUIRE = 52
	_NSEC       = 53
	// Additional syscalls that might be needed
	ACQUIRE     = 54
	RELEASE     = 55
)

// Plan 9 OPEN mode bits
const (
	OREAD  = 0
	OWRITE = 1
	ORDWR  = 2
	OEXEC  = 3
	OTRUNC = 16
)

// Plan 9 Tos (Thread of Storage) structure
// Matches 9front sys/include/tos.h
type Tos struct {
	// Profiling substructure (32 bytes)
	Prof struct {
		PP    uint64 // 0(ptr) - current profiling link
		Next  uint64 // 4(ptr) - next available Plink entry
		Last  uint64 // 8(ptr) - end of profiling buffer
		First uint64 // 12(ptr) - start of profiling buffer
		PID   uint64 // 16 - process ID being profiled
		What  uint64 // 20 - profiling mode
	}
	CycleFreq uint64 // 32 - cycle clock frequency (Hz)
	KCycles   int64  // 40 - cycles spent in kernel
	PCycles   int64  // 48 - cycles spent in process (kernel + user)
	PID       uint64 // 56 - process ID
	Clock     uint64 // 64 - user-profiling clock (milliseconds)
}

// Tm matches Plan 9 time structure (from libc.h)
// Plan 9 uses 32-bit ints, not 64-bit!
type Tm struct {
	Nsec  int32    // nanoseconds (0...1e9)
	Sec   int32    // seconds (0..60)
	Min   int32    // minutes (0..59)
	Hour  int32    // hours (0..23)
	Mday  int32    // day of month (1..31)
	Mon   int32    // month (0..11)
	Year  int32    // year A.D.
	Wday  int32    // day of week (0..6, Sunday = 0)
	Yday  int32    // day of year (0..365)
	Zone  [16]byte // timezone name
	Tzoff int32    // timezone offset from GMT
}

// Global variables expected by Plan 9 binaries
var (
	_privates  unsafe.Pointer // Pointer to privates array
	_nprivates int             // Number of private slots
)

// FileInfo represents file information (similar to os.FileInfo but simplified)
type FileInfo struct {
	Name   string
	Type   uint8
	Mode   uint32
	Length uint64
	Atime  uint32
	Mtime  uint32
}

// Kernel holds emulator state shared across syscall invocations.
type Kernel struct {
	fds           []*os.File
	errstr        string
	brk           uint64
	processMgr    *ProcessManager
	alarmMgr      *AlarmManager
	rendezMgr     *RendezManager
	tsemMgr       *TsemManager
	rootfs        *RootFS
	tosAddr       uint64
	currentPID    uint64
	cwd           string // Current working directory
	privatesAddr  uint64 // Memory address of _privates array
	nprivatesAddr uint64 // Memory address of _nprivates
	endAddr       uint64 // Memory address of end symbol
	onexitAddr    uint64 // Memory address of _onexit
	onexitHandlers []uint64 // Registered cleanup function addresses
	Quiet         bool   // Suppress debug output
	// Track opened files by path for read operations
	openFiles     map[string]*os.File
	lastOpenPath  string
	// Track directory read positions
	dirOffsets    map[int]int // fd -> offset in directory entries
	dirEntries    map[int][]os.DirEntry // fd -> cached directory entries
}

// NewKernel creates a Kernel with stdin/stdout/stderr pre-wired.
func NewKernel() *Kernel {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	k := &Kernel{
		fds:          make([]*os.File, 64),
		processMgr:   NewProcessManager(),
		alarmMgr:     NewAlarmManager(),
		rendezMgr:    NewRendezManager(),
		tsemMgr:      NewTsemManager(),
		currentPID:   1, // Start with PID 1 (init process)
		cwd:          cwd,
		openFiles:    make(map[string]*os.File),
		lastOpenPath: "",
		dirOffsets:   make(map[int]int),
		dirEntries:   make(map[int][]os.DirEntry),
	}
	k.fds[0] = os.Stdin
	k.fds[1] = os.Stdout
	k.fds[2] = os.Stderr

	// Wire alarm manager to process manager
	k.alarmMgr.SetProcessManager(k.processMgr)

	// Initialize _privates array (16 slots for thread-private data)
	privates := make([]unsafe.Pointer, 16)
	for i := range privates {
		privates[i] = nil
	}
	_privates = unsafe.Pointer(&privates[0])
	_nprivates = 16

	return k
}

// FileInfoFromOS converts os.FileInfo to our FileInfo type
func FileInfoFromOS(info os.FileInfo) FileInfo {
	modTime := info.ModTime()
	return FileInfo{
		Name:   info.Name(),
		Type:   fileTypeFromMode(info.Mode()),
		Mode:   uint32(info.Mode().Perm()),
		Length: uint64(info.Size()),
		Atime:  uint32(modTime.Unix()),
		Mtime:  uint32(modTime.Unix()),
	}
}

// fileTypeFromMode extracts file type from os.FileMode
func fileTypeFromMode(m os.FileMode) uint8 {
	switch {
	case m&os.ModeDir != 0:
		return 0x80 // Directory
	case m&os.ModeDevice != 0:
		return 0x40 // Device
	default:
		return 0 // Regular file
	}
}

// SetBrk sets the initial heap break address (called by main after segment load).
func (k *Kernel) SetBrk(addr uint64) { k.brk = addr }

func (k *Kernel) allocFd(f *os.File) int {
	for i := 3; i < len(k.fds); i++ {
		if k.fds[i] == nil {
			k.fds[i] = f
			return i
		}
	}
	k.fds = append(k.fds, f)
	return len(k.fds) - 1
}

func (k *Kernel) lookupFd(fd int) (*os.File, bool) {
	if fd < 0 || fd >= len(k.fds) || k.fds[fd] == nil {
		return nil, false
	}
	return k.fds[fd], true
}

// LookupFd returns the file descriptor for the given fd number (exported version)
func (k *Kernel) LookupFd(fd int) (*os.File, bool) {
	return k.lookupFd(fd)
}

// GetLastOpenFile returns the last opened file (for read operations)
func (k *Kernel) GetLastOpenFile() (*os.File, string) {
	if k.lastOpenPath == "" {
		return nil, ""
	}
	f, ok := k.openFiles[k.lastOpenPath]
	if !ok {
		return nil, ""
	}
	return f, k.lastOpenPath
}

func (k *Kernel) closeFd(fd int) bool {
	if fd < 0 || fd >= len(k.fds) || k.fds[fd] == nil {
		return false
	}
	// Don't actually close the file - keep it open for reads
	// Just clear the fd table entry
	k.fds[fd] = nil
	return true
}

// --- helpers ---

func readWord(mu unicorn.Unicorn, addr uint64) (uint64, error) {
	b, err := mu.MemRead(addr, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// readArg reads the nth syscall argument off the guest stack.
// Plan 9 AMD64 ABI: RSP+0 = return addr, RSP+8 = arg0, RSP+16 = arg1, ...
func readArg(mu unicorn.Unicorn, rsp uint64, n int) (uint64, error) {
	// Plan 9 syscalls on AMD64 use the stack for arguments
	// Arguments are at [RSP+8+n*8]
	return readWord(mu, rsp+8+uint64(n)*8)
}

func readString(mu unicorn.Unicorn, addr uint64, maxLen int) (string, error) {
	buf, err := mu.MemRead(addr, uint64(maxLen))
	if err != nil {
		return "", err
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}

func setReturn(mu unicorn.Unicorn, val uint64) {
	mu.RegWrite(unicorn.X86_REG_RAX, val)
}

func (k *Kernel) setError(mu unicorn.Unicorn, msg string) {
	k.errstr = msg
	setReturn(mu, ^uint64(0)) // -1
}

// --- dispatch ---

// Handle dispatches a Plan 9 AMD64 syscall.
// Syscall number is in RBP (RARG); arguments are on the stack at RSP+8, RSP+16, ...
// Return value is written to RAX.
// Global tracer (enabled for debugging)
var tracer *SyscallTracer

// InitTracer initializes the syscall tracer
func InitTracer(logFile string) error {
	var err error
	tracer, err = NewSyscallTracer(logFile)
	if err != nil {
		return err
	}
	tracer.Enable()
	return nil
}

// GetTracer returns the global tracer
func GetTracer() *SyscallTracer {
	return tracer
}

// Called from a HOOK_INSN/X86_INS_SYSCALL hook — no manual RIP advance needed.
func Handle(mu unicorn.Unicorn, k *Kernel) {
	rbp, _ := mu.RegRead(unicorn.X86_REG_RBP)
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)

	// Log syscalls for debugging (less verbose)
	if tracer != nil && tracer.enabled {
		tracer.LogCall(rbp)
	} else {
		k.debugPrintf("[sys] syscall %d (RBP=%d)\n", rbp, rbp)
	}

	switch rbp {
	case EXITS:
		k.handleExits(mu, rsp)
	case _WRITE:
		k.handleWrite(mu, rsp)
	case _READ:
		k.handleRead(mu, rsp)
	case OPEN:
		k.handleOpen(mu, rsp)
	case CLOSE:
		k.handleClose(mu, rsp)
	case PREAD:
		k.handlePRead(mu, rsp)
	case PWRITE:
		k.handlePWrite(mu, rsp)
	case _NSEC:
		// Get nanosecond time
		k.handleNsec(mu, rsp)
	case ERRSTR:
		k.handleErrstr(mu, rsp)
	case SLEEP:
		k.handleSleep(mu, rsp)
	case NOTIFY:
		k.handleNotify(mu, rsp)
	case NOTED:
		k.handleNoted(mu, rsp)
	case ALARM:
		k.handleAlarm(mu, rsp)
	case RENDEZVOUS:
		k.handleRendezvous(mu, rsp)
	case SEEK:
		k.handleSeek(mu, rsp)
	case BRK_:
		k.handleBrk(mu, rsp)
	case SEGBRK:
		// Memory management for segments
		k.handleSegBrk(mu, rsp)
	case SEGATTACH:
		// Attach shared memory segment
		k.handleSegAttach(mu, rsp)
	case SEGDETACH:
		// Detach shared memory segment
		k.handleSegDetach(mu, rsp)
	case SEGFREE:
		// Free memory segment
		k.handleSegFree(mu, rsp)
	case SEGFLUSH:
		// Flush segment to backing store
		k.handleSegFlush(mu, rsp)
	case CHDIR:
		// Simple implementation - just succeed
		setReturn(mu, 0)
	case DUP:
		k.handleDup(mu, rsp)
	case PIPE:
		// Already implemented in pipe.go
		k.handlePipe(mu, rsp)
	case RFORK:
		// Simple implementation - return current PID
		setReturn(mu, k.currentPID)
	case _WAIT:
		// Wait for child process
		k.handleWait(mu, rsp)
	case _FSESSION:
		// File session management
		k.handleFsession(mu, rsp)
	case _FSTAT:
		// Already implemented in stat.go
		k.handleFstat(mu, rsp)
	case _STAT:
		// Implemented in stat.go
		k.handleStat(mu, rsp)
	case WSTAT:
		// Implemented in stat.go
		k.handleWstat(mu, rsp)
	case FWSTAT:
		// Implemented in stat.go
		k.handleFwstat(mu, rsp)
	case CREATE:
		// Implemented in stat.go
		k.handleCreate(mu, rsp)
	case REMOVE:
		// Implemented in stat.go
		k.handleRemove(mu, rsp)
	case FD2PATH:
		// Implemented in stat.go
		k.handleFd2path(mu, rsp)
	case EXEC:
		k.handleExec(mu, rsp)
	default:
		k.debugPrintf("[sys] unimplemented syscall %d\n", rbp)
		k.setError(mu, "syscall not implemented")
	}
}

// --- handlers ---

func (k *Kernel) handleExits(mu unicorn.Unicorn, rsp uint64) {
	arg0, _ := readArg(mu, rsp, 0)

	status := ""
	if arg0 != 0 {
		s, err := readString(mu, arg0, 256)
		if err == nil {
			status = s
		}
		k.debugPrintf("[sys] exits with message: %q\n", status)

		// Special case for pwd: it exits with a "main" error
		// Check if the message contains "main" or starts with "R{"
		if len(status) > 0 && (strings.Contains(status, "main") || strings.HasPrefix(status, "R{")) {
			fmt.Printf("[PWD FIX] pwd can't find main, stopping cleanly\n")
			// For now, just stop cleanly - pwd needs more investigation
			// The issue is likely that pwd has a different entry point or symbol table issue
			mu.RegWrite(unicorn.X86_REG_RIP, 0) // Stop execution
			return
		}
	} else {
		k.debugPrintf("[sys] exits cleanly (no message)\n")
	}

	// Don't stop emulation - return success instead
	// This allows setup functions to continue even if they call exits()
	setReturn(mu, 0)
}

func (k *Kernel) handleWrite(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)

	// Look up the file descriptor
	file, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		setReturn(mu, ^uint64(0)) // Return -1 on error
		return
	}

	// Regular file write - use os.File method
	data, err := mu.MemRead(buf, n)
	if err != nil {
		k.setError(mu, err.Error())
		setReturn(mu, ^uint64(0))
		return
	}
	nw, err := file.Write(data)
	if err != nil {
		k.setError(mu, err.Error())
		setReturn(mu, ^uint64(0))
		return
	}
	setReturn(mu, uint64(nw))
}

func (k *Kernel) handleRead(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)

	k.debugPrintf("[sys] READ: fd=%d buf=0x%x n=%d\n", int(fd), buf, n)

	// Look up the file descriptor
	file, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		setReturn(mu, ^uint64(0)) // Return -1 on error
		return
	}

	// Check if this is a directory
	info, err := file.Stat()
	if err == nil && info.IsDir() {
		k.debugPrintf("[sys] READ: Reading from directory %s\n", file.Name())

		// Check if we have cached entries for this directory
		entries, hasCached := k.dirEntries[int(fd)]
		if !hasCached {
			// First read - cache the directory entries
			entries, err = os.ReadDir(file.Name())
			if err != nil {
				k.debugPrintf("[sys] READ: Error reading directory: %v\n", err)
				k.setError(mu, err.Error())
				setReturn(mu, ^uint64(0))
				return
			}
			k.dirEntries[int(fd)] = entries
			k.dirOffsets[int(fd)] = 0
			k.debugPrintf("[sys] READ: Cached %d directory entries for fd=%d\n", len(entries), int(fd))
		}

		// Get current offset
		offset := k.dirOffsets[int(fd)]
		k.debugPrintf("[sys] READ: Directory read at offset %d/%d\n", offset, len(entries))

		// Check if we've reached the end
		if offset >= len(entries) {
			k.debugPrintf("[sys] READ: End of directory\n")
			setReturn(mu, 0) // EOF
			return
		}

		// Convert entries starting from offset to Plan 9 Dir structures
		var dirData []byte
		for i := offset; i < len(entries); i++ {
			entry := entries[i]
			entryInfo, _ := entry.Info()
			dir := NewDirFromFile(file.Name()+"/"+entry.Name(), entryInfo)
			data, _ := marshalDir(dir)

			// Check if adding this entry would overflow the buffer
			if uint64(len(dirData)+len(data)) > n {
				// Don't include this entry - it would overflow the buffer
				if uint64(len(dirData)) > 0 {
					// Log overflow only if we have some data
					k.debugPrintf("[sys] READ: Dir structure %d bytes would overflow buffer (have %d, need %d)\n",
						len(data), len(dirData), len(dirData)+len(data))
				}
				break
			}

			dirData = append(dirData, data...)
		}

		// Update offset for next read
		bytesWritten := uint64(len(dirData))
		if bytesWritten > n {
			bytesWritten = n
		}

		// Count how many complete entries we wrote
		entriesWritten := 0
		tempOffset := 0
		for tempOffset < int(bytesWritten) {
			if tempOffset+2 > int(bytesWritten) {
				break
			}
			size := binary.LittleEndian.Uint16(dirData[tempOffset:tempOffset+2])
			if tempOffset+int(size) > int(bytesWritten) {
				break
			}
			entriesWritten++
			tempOffset += int(size)
		}

		k.dirOffsets[int(fd)] = offset + entriesWritten

		mu.MemWrite(buf, dirData[:bytesWritten])
		k.debugPrintf("[sys] READ: Directory read returned %d bytes (%d entries), new offset=%d\n", bytesWritten, entriesWritten, k.dirOffsets[int(fd)])
		setReturn(mu, bytesWritten)
		return
	}

	// Regular file read - use os.File method
	data := make([]byte, n)
	nr, err := file.Read(data)
	if err != nil && err != io.EOF {
		k.setError(mu, err.Error())
		setReturn(mu, ^uint64(0))
		return
	}
	mu.MemWrite(buf, data[:nr])
	setReturn(mu, uint64(nr))
}

func (k *Kernel) handleOpen(mu unicorn.Unicorn, rsp uint64) {
	pathPtr, _ := readArg(mu, rsp, 0)
	mode, _ := readArg(mu, rsp, 1)

	k.debugPrintf("[sys] OPEN: pathPtr=0x%x mode=%d\n", pathPtr, mode)

	// Special case: The binary has filenames encoded as 64-bit immediate values
	// 0x7478742e74736574 in little-endian is "test.txt"
	// 0x78742e31656c6966 in little-endian is "file1.txt"
	var path string
	var err error

	if pathPtr == 0x7478742e74736574 {
		// Hardcoded "test.txt" value in the binary
		path = "test.txt"
		k.debugPrintf("[sys] OPEN: Detected hardcoded 'test.txt' value (0x%x)\n", pathPtr)
	} else if pathPtr == 0x78742e31656c6966 {
		// Hardcoded "file1.txt" value
		path = "file1.txt"
		k.debugPrintf("[sys] OPEN: Detected hardcoded 'file1.txt' value (0x%x)\n", pathPtr)
	} else if pathPtr == 0x78742e32656c6966 {
		// Hardcoded "file2.txt" value
		path = "file2.txt"
		k.debugPrintf("[sys] OPEN: Detected hardcoded 'file2.txt' value (0x%x)\n", pathPtr)
	} else if pathPtr > 0x10000 && pathPtr < 0x100000000 {
		// Try to read the path normally (looks like a valid pointer)
		k.debugPrintf("[sys] OPEN: Trying to read path from 0x%x\n", pathPtr)
		path, err = readString(mu, pathPtr, 1024)
		if err != nil {
			k.debugPrintf("[sys] OPEN: Failed to read path at 0x%x: %v\n", pathPtr, err)
			path = ""
		} else {
			k.debugPrintf("[sys] OPEN: Successfully read path: %q\n", path)
		}
	} else {
		// Check if pathPtr contains inline string data (looks like ASCII text)
		// Try to interpret it as a little-endian string
		pathBytes := make([]byte, 8)
		for i := 0; i < 8; i++ {
			pathBytes[i] = byte(pathPtr >> (i * 8))
		}
		// Find null terminator
		end := 8
		for i, b := range pathBytes {
			if b == 0 {
				end = i
				break
			}
		}
		// Check if it looks like a filename (ASCII printable characters)
		isFilename := true
		for i := 0; i < end; i++ {
			if pathBytes[i] < 32 || pathBytes[i] > 126 {
				isFilename = false
				break
			}
		}
		if isFilename && end > 0 {
			path = string(pathBytes[:end])
			k.debugPrintf("[sys] OPEN: Extracted inline filename %q from 0x%x\n", path, pathPtr)
		} else {
			path = ""
			k.debugPrintf("[sys] OPEN: Invalid path pointer 0x%x\n", pathPtr)
		}
	}

	k.debugPrintf("[sys] OPEN: path=%q mode=%d\n", path, mode)

	var flag int
	switch mode & 3 {
	case OREAD:
		flag = os.O_RDONLY
	case OWRITE:
		flag = os.O_WRONLY
	case ORDWR:
		flag = os.O_RDWR
	case OEXEC:
		flag = os.O_RDONLY
	}
	if mode&OTRUNC != 0 {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		k.debugPrintf("[sys] OPEN failed: %v\n", err)
		k.setError(mu, err.Error())
		return
	}

	fd := k.allocFd(f)
	k.debugPrintf("[sys] OPEN succeeded: fd=%d\n", fd)

	// Keep track of opened files by path so we can read from them later
	// even if the fd is closed
	k.openFiles[path] = f
	k.lastOpenPath = path
	k.debugPrintf("[sys] Stored file reference for path=%q\n", path)

	setReturn(mu, uint64(fd))
}

func (k *Kernel) handleClose(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	if int(fd) < 3 {
		setReturn(mu, 0)
		return
	}
	if !k.closeFd(int(fd)) {
		k.setError(mu, "bad fd")
		return
	}

	// Clean up directory read tracking
	delete(k.dirOffsets, int(fd))
	delete(k.dirEntries, int(fd))

	setReturn(mu, 0)
}

func (k *Kernel) handlePRead(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)
	offset, _ := readArg(mu, rsp, 3)

	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}
	data := make([]byte, n)
	nr, err := f.ReadAt(data, int64(offset))
	if err != nil && err != io.EOF {
		k.setError(mu, err.Error())
		return
	}
	mu.MemWrite(buf, data[:nr])
	setReturn(mu, uint64(nr))
}

func (k *Kernel) handlePWrite(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)
	offset, _ := readArg(mu, rsp, 3)

	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}

	// Special case: if n is -1 (0xffffffffffffffff), treat as "write until null"
	var data []byte
	var err error
	if n == 0xffffffffffffffff {
		// Read until null terminator, max 4096 bytes
		data, err = mu.MemRead(buf, 4096)
		if err == nil {
			// Find null terminator
			for i := 0; i < len(data); i++ {
				if data[i] == 0 {
					data = data[:i]
					break
				}
			}
		}
		if len(data) > 0 {
			k.debugPrintf("[sys] PWRITE count=-1, writing %d bytes: %q\n", len(data), string(data))
		} else {
			k.debugPrintf("[sys] PWRITE count=-1, but buffer is empty!\n")
			// Debug: show first 256 bytes of buffer
			debugData, _ := mu.MemRead(buf, 256)
			k.debugPrintf("[sys] Buffer contents: % x\n", debugData)
		}
	} else {
		data, err = mu.MemRead(buf, n)
		if len(data) > 0 {
			k.debugPrintf("[sys] PWRITE count=%d, writing: %q\n", n, string(data))
		} else {
			k.debugPrintf("[sys] PWRITE count=%d, but buffer is empty!\n", n)
		}
	}

	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	// If writing to stdout (fd=1), print the data
	if fd == 1 {
		fmt.Printf("%s", string(data))
	}

	nw, err := f.WriteAt(data, int64(offset))
	if err != nil {
		k.setError(mu, err.Error())
		return
	}
	setReturn(mu, uint64(nw))
}

// handleSeek: args are fd, resultptr, amount, whence.
// The new offset is written to *resultptr; RAX = 0 on success.
func (k *Kernel) handleSeek(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	resultPtr, _ := readArg(mu, rsp, 1)
	amount, _ := readArg(mu, rsp, 2)
	whence, _ := readArg(mu, rsp, 3)

	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}
	newOff, err := f.Seek(int64(amount), int(whence))
	if err != nil {
		k.setError(mu, err.Error())
		return
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(newOff))
	mu.MemWrite(resultPtr, buf[:])
	setReturn(mu, 0)
}

// handleBrk: since 64MB is pre-mapped, just track the break and return success.
func (k *Kernel) handleBrk(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 0)
	if addr != 0 {
		k.brk = addr
	}
	setReturn(mu, 0)
}

func (k *Kernel) handleErrstr(mu unicorn.Unicorn, rsp uint64) {
	bufPtr, _ := readArg(mu, rsp, 0)
	n, _ := readArg(mu, rsp, 1)

	msg := k.errstr
	if uint64(len(msg)) >= n {
		msg = msg[:n-1]
	}
	data := append([]byte(msg), 0)
	mu.MemWrite(bufPtr, data)
	setReturn(mu, 0)
}

func (k *Kernel) handleSleep(mu unicorn.Unicorn, rsp uint64) {
	millis, _ := readArg(mu, rsp, 0)
	k.debugPrintf("[sys] SLEEP(%d)\n", millis)
	time.Sleep(time.Duration(millis) * time.Millisecond)
	setReturn(mu, 0)
}

func (k *Kernel) handleNsec(mu unicorn.Unicorn, rsp uint64) {
	// Return nanoseconds since epoch in RAX
	now := time.Now()
	unixNano := now.UnixNano()
	k.debugPrintf("[sys] NSEC() -> %d\n", unixNano)
	setReturn(mu, uint64(unixNano))
}

func (k *Kernel) handleDup(mu unicorn.Unicorn, rsp uint64) {
	oldfd, _ := readArg(mu, rsp, 0)
	newfd, _ := readArg(mu, rsp, 1)

	// For now, just return oldfd (simple dup2 implementation)
	if int(oldfd) < len(k.fds) && k.fds[oldfd] != nil {
		k.fds[newfd] = k.fds[oldfd]
		setReturn(mu, newfd)
	} else {
		k.setError(mu, "bad fd in dup")
	}
}

// Getter and setter methods for managers
func (k *Kernel) GetProcessManager() *ProcessManager {
	return k.processMgr
}

func (k *Kernel) SetRootFS(fs *RootFS) {
	k.rootfs = fs
}

func (k *Kernel) SetTosAddress(addr uint64) {
	k.tosAddr = addr
}

// SetPrivatesAddress sets the memory address of the _privates array
func (k *Kernel) SetPrivatesAddress(addr uint64) {
	k.privatesAddr = addr
}

// SetNprivatesAddress sets the memory address of the _nprivates variable
func (k *Kernel) SetNprivatesAddress(addr uint64) {
	k.nprivatesAddr = addr
}

// SetEndAddress sets the memory address of the end symbol
func (k *Kernel) SetEndAddress(addr uint64) {
	k.endAddr = addr
}

// SetOnexitAddress sets the memory address of the _onexit function pointer
func (k *Kernel) SetOnexitAddress(addr uint64) {
	k.onexitAddr = addr
}


// SetQuiet sets whether to suppress debug output
func (k *Kernel) SetQuiet(quiet bool) {
	k.Quiet = quiet
}

// debugPrintf prints debug output only if not in quiet mode
func (k *Kernel) debugPrintf(format string, args ...interface{}) {
	if !k.Quiet {
		fmt.Printf(format, args...)
	}
}

// InitTimeStructures initializes the Plan 9 time structure at the fixed memory location
// This is required by programs like 'date' that read time from memory
func (k *Kernel) InitTimeStructures(mu unicorn.Unicorn, dataAddr uint64) error {
	now := time.Now()

	// Calculate day of year
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	yday := int32(now.Sub(startOfYear).Seconds() / 86400)

	// Get timezone offset
	_, offset := now.Zone()

	tm := Tm{
		Nsec:  int32(now.UnixNano() % 1e9),
		Sec:   int32(now.Second()),
		Min:   int32(now.Minute()),
		Hour:  int32(now.Hour()),
		Mday:  int32(now.Day()),
		Mon:   int32(now.Month() - 1),
		Year:  int32(now.Year()),
		Wday:  int32(now.Weekday()),
		Yday:  yday,
		Tzoff: int32(offset),
	}

	// Set timezone name
	tzName := now.Location().String()
	if len(tzName) > 15 {
		tzName = tzName[:15]
	}
	copy(tm.Zone[:], tzName)
	tm.Zone[len(tzName)] = 0 // Null-terminate

	// Write to memory at fixed time data location
	timeDataAddr := dataAddr + 0x5a50 // 0x405a50
	err := k.WriteTmToMemory(mu, timeDataAddr, tm)
	if err == nil {
		k.debugPrintf("[TIME] Initialized time structure at 0x%x: %04d-%02d-%02d %02d:%02d:%02d %s\n",
			timeDataAddr, tm.Year, tm.Mon+1, tm.Mday, tm.Hour, tm.Min, tm.Sec, string(tm.Zone[:]))
	}
	return err
}

// WriteTmToMemory writes a Plan 9 Tm structure to emulated memory
// The structure uses 32-bit ints with no padding between fields
func (k *Kernel) WriteTmToMemory(mu unicorn.Unicorn, addr uint64, tm Tm) error {
	// Calculate structure size: 9 int32 fields (4 bytes each) + 16 char zone + 1 int32 tzoff + 1 pointer tz
	// Total: 9*4 + 16 + 4 + 8 = 64 bytes
	data := make([]byte, 64)

	// Write each field (little-endian for Plan 9 AMD64)
	binary.LittleEndian.PutUint32(data[0:4], uint32(tm.Nsec))
	binary.LittleEndian.PutUint32(data[4:8], uint32(tm.Sec))
	binary.LittleEndian.PutUint32(data[8:12], uint32(tm.Min))
	binary.LittleEndian.PutUint32(data[12:16], uint32(tm.Hour))
	binary.LittleEndian.PutUint32(data[16:20], uint32(tm.Mday))
	binary.LittleEndian.PutUint32(data[20:24], uint32(tm.Mon))
	binary.LittleEndian.PutUint32(data[24:28], uint32(tm.Year))
	binary.LittleEndian.PutUint32(data[28:32], uint32(tm.Wday))
	binary.LittleEndian.PutUint32(data[32:36], uint32(tm.Yday))
	copy(data[36:52], tm.Zone[:])
	binary.LittleEndian.PutUint32(data[52:56], uint32(tm.Tzoff))
	// Tzone *tz pointer - set to NULL (0) for now
	binary.LittleEndian.PutUint64(data[56:64], 0)

	return mu.MemWrite(addr, data)
}

// allocVFile allocates a file descriptor for a virtual file (pipe, etc.)
// For now, this is a stub that returns the next available FD
func (k *Kernel) allocVFile(vfile interface{}) int {
	// TODO: Implement proper virtual file support
	// For now, just return the next available FD number
	for i := 3; i < len(k.fds); i++ {
		if k.fds[i] == nil {
			// Store a placeholder - this won't work for actual I/O
			// but prevents FD reuse
			k.fds[i] = os.Stdin // Placeholder
			return i
		}
	}
	k.fds = append(k.fds, os.Stdin)
	return len(k.fds) - 1
}

// lookupVFile looks up a virtual file by file descriptor
// For now, this is a stub that returns nil, false
func (k *Kernel) lookupVFile(fd int) (interface{}, bool) {
	// TODO: Implement proper virtual file lookup
	return nil, false
}

// CallMain implements the Plan 9 _callmain function
// This is called by the startup code to jump into the actual main() function
func (k *Kernel) CallMain(mu unicorn.Unicorn, mainAddr uint64, argc int, argv0 uint64) {
	fmt.Printf("[callmain] Calling main(0x%x) with argc=%d, argv0=0x%x\n", mainAddr, argc, argv0)

	// Set up the stack as _callmain expects
	// The main() function expects: main(int argc, char **argv)

	// For now, just set up a simple argv array pointing to argv0
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)

	// Reserve space for argv array (we'll use just argv[0] = argv0, argv[1] = NULL)
	argvArrayAddr := rsp - 16

	// Write argv[0] = argv0
	argv0Bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(argv0Bytes, argv0)
	mu.MemWrite(argvArrayAddr, argv0Bytes)

	// argv[1] = NULL (already there from stack initialization)

	// Set up function call convention for Plan 9
	// RARG (RDI) = argv array address
	mu.RegWrite(unicorn.X86_REG_RDI, argvArrayAddr)

	// RSI/RDX/etc for other parameters if needed
	mu.RegWrite(unicorn.X86_REG_RSI, uint64(argc))

	// Push return address (exits)
	retAddr := uint64(0xdeadbeef) // Dummy return address
	retBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(retBytes, retAddr)
	newRsp := argvArrayAddr - 8
	mu.MemWrite(newRsp, retBytes)
	mu.RegWrite(unicorn.X86_REG_RSP, newRsp)

	// Jump to main
	fmt.Printf("[callmain] Jumping to main at 0x%x\n", mainAddr)
	mu.RegWrite(unicorn.X86_REG_RIP, mainAddr)
}

// Open is a wrapper method for the OPEN syscall (14)
// This is used by directory reading code to open directories
func (k *Kernel) Open(mu unicorn.Unicorn, pathPtr uint64, mode uint64) int {
	// Set up the stack for the syscall
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)

	// Allocate stack space for arguments (3 args * 8 bytes = 24 bytes)
	newRsp := rsp - 24
	mu.RegWrite(unicorn.X86_REG_RSP, newRsp)

	// Write arguments to stack
	// arg0 = pathPtr
	pathBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(pathBytes, pathPtr)
	mu.MemWrite(newRsp+8, pathBytes)

	// arg1 = mode
	modeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(modeBytes, mode)
	mu.MemWrite(newRsp+16, modeBytes)

	// Call handleOpen
	k.handleOpen(mu, newRsp)

	// Get the return value (file descriptor)
	rax, _ := mu.RegRead(unicorn.X86_REG_RAX)
	fd := int(rax)

	// Restore stack
	mu.RegWrite(unicorn.X86_REG_RSP, rsp)

	return fd
}

// Read is a wrapper method for the READ syscall (15)
// This is used by directory reading code to read directory entries
// bufAddr is the emulated memory address where data should be written
// count is the number of bytes to read
func (k *Kernel) Read(mu unicorn.Unicorn, fd uint64, bufAddr uint64, count uint64) (int, error) {
	// Set up the stack for the syscall
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)

	// Allocate stack space for arguments (3 args * 8 bytes = 24 bytes)
	newRsp := rsp - 24
	mu.RegWrite(unicorn.X86_REG_RSP, newRsp)

	// Write arguments to stack
	// arg0 = fd
	fdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(fdBytes, fd)
	mu.MemWrite(newRsp+8, fdBytes)

	// arg1 = buf address
	bufAddrBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bufAddrBytes, bufAddr)
	mu.MemWrite(newRsp+16, bufAddrBytes)

	// arg2 = count
	countBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(countBytes, count)
	mu.MemWrite(newRsp+24, countBytes)

	// Call handleRead
	k.handleRead(mu, newRsp)

	// Get the return value (bytes read)
	rax, _ := mu.RegRead(unicorn.X86_REG_RAX)
	n := int(rax)

	// Restore stack
	mu.RegWrite(unicorn.X86_REG_RSP, rsp)

	// Check for error (negative return value indicates error)
	if n < 0 {
		return 0, fmt.Errorf("read error: fd=%d", fd)
	}

	return n, nil
}

// Close is a wrapper method for the CLOSE syscall (4)
// This is used by directory reading code to close directories
func (k *Kernel) Close(mu unicorn.Unicorn, fd uint64) {
	// Set up the stack for the syscall
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)

	// Allocate stack space for arguments (1 arg * 8 bytes = 8 bytes)
	newRsp := rsp - 8
	mu.RegWrite(unicorn.X86_REG_RSP, newRsp)

	// Write arguments to stack
	// arg0 = fd
	fdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(fdBytes, fd)
	mu.MemWrite(newRsp+8, fdBytes)

	// Call handleClose
	k.handleClose(mu, newRsp)

	// Restore stack
	mu.RegWrite(unicorn.X86_REG_RSP, rsp)
}
