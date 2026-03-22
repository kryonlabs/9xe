package sys

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

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
)

// Plan 9 OPEN mode bits
const (
	OREAD  = 0
	OWRITE = 1
	ORDWR  = 2
	OEXEC  = 3
	OTRUNC = 16
)

// Kernel holds emulator state shared across syscall invocations.
type Kernel struct {
	fds    []*os.File
	errstr string
	brk    uint64
}

// NewKernel creates a Kernel with stdin/stdout/stderr pre-wired.
func NewKernel() *Kernel {
	k := &Kernel{fds: make([]*os.File, 64)}
	k.fds[0] = os.Stdin
	k.fds[1] = os.Stdout
	k.fds[2] = os.Stderr
	return k
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

func (k *Kernel) closeFd(fd int) bool {
	if fd < 0 || fd >= len(k.fds) || k.fds[fd] == nil {
		return false
	}
	k.fds[fd].Close()
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
// Called from a HOOK_INSN/X86_INS_SYSCALL hook — no manual RIP advance needed.
func Handle(mu unicorn.Unicorn, k *Kernel) {
	rbp, _ := mu.RegRead(unicorn.X86_REG_RBP)
	rsp, _ := mu.RegRead(unicorn.X86_REG_RSP)
	rip, _ := mu.RegRead(unicorn.X86_REG_RIP)

	fmt.Printf("[sys] syscall %d at 0x%x\n", rbp, rip)

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
	case SEEK:
		k.handleSeek(mu, rsp)
	case BRK_:
		k.handleBrk(mu, rsp)
	case ERRSTR:
		k.handleErrstr(mu, rsp)
	default:
		fmt.Printf("[sys] unimplemented syscall %d\n", rbp)
		k.setError(mu, "syscall not implemented")
	}
}

// --- handlers ---

func (k *Kernel) handleExits(mu unicorn.Unicorn, rsp uint64) {
	arg0, _ := readArg(mu, rsp, 0)
	status := ""
	if arg0 != 0 {
		s, err := readString(mu, arg0, 64)
		if err == nil {
			status = s
		}
	}
	fmt.Printf("[sys] exits: %q\n", status)
	mu.Stop()
}

func (k *Kernel) handleWrite(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)

	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}
	data, err := mu.MemRead(buf, n)
	if err != nil {
		k.setError(mu, err.Error())
		return
	}
	nw, err := f.Write(data)
	if err != nil {
		k.setError(mu, err.Error())
		return
	}
	setReturn(mu, uint64(nw))
}

func (k *Kernel) handleRead(mu unicorn.Unicorn, rsp uint64) {
	fd, _ := readArg(mu, rsp, 0)
	buf, _ := readArg(mu, rsp, 1)
	n, _ := readArg(mu, rsp, 2)

	f, ok := k.lookupFd(int(fd))
	if !ok {
		k.setError(mu, "bad fd")
		return
	}
	data := make([]byte, n)
	nr, err := f.Read(data)
	if err != nil && err != io.EOF {
		k.setError(mu, err.Error())
		return
	}
	mu.MemWrite(buf, data[:nr])
	setReturn(mu, uint64(nr))
}

func (k *Kernel) handleOpen(mu unicorn.Unicorn, rsp uint64) {
	pathPtr, _ := readArg(mu, rsp, 0)
	mode, _ := readArg(mu, rsp, 1)

	path, err := readString(mu, pathPtr, 1024)
	if err != nil {
		k.setError(mu, "bad path ptr")
		return
	}

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
		k.setError(mu, err.Error())
		return
	}
	setReturn(mu, uint64(k.allocFd(f)))
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
	data, err := mu.MemRead(buf, n)
	if err != nil {
		k.setError(mu, err.Error())
		return
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
