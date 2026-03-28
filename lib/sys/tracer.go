package sys

import (
	"fmt"
	"os"
)

// SyscallTracer tracks and logs syscalls
type SyscallTracer struct {
	enabled       bool
	logFile       *os.File
	syscallCounts map[uint64]int
	lastSyscall   uint64
	instructionCount int
}

// NewSyscallTracer creates a new syscall tracer
func NewSyscallTracer(logFile string) (*SyscallTracer, error) {
	var file *os.File
	var err error

	if logFile != "" {
		file, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
	}

	return &SyscallTracer{
		enabled:       true,
		logFile:       file,
		syscallCounts: make(map[uint64]int),
	}, nil
}

// Enable enables syscall tracing
func (st *SyscallTracer) Enable() {
	st.enabled = true
}

// Disable disables syscall tracing
func (st *SyscallTracer) Disable() {
	st.enabled = false
}

// LogCall logs a syscall invocation
func (st *SyscallTracer) LogCall(syscallNum uint64, args ...uint64) {
	if !st.enabled {
		return
	}

	st.syscallCounts[syscallNum]++
	st.lastSyscall = syscallNum

	// Get syscall name
	name := st.getSyscallName(syscallNum)

	// Log to file if available
	if st.logFile != nil {
		fmt.Fprintf(st.logFile, "[SYSCALL %d] %s args=%v\n", syscallNum, name, args)
	}

	// Also log to stdout for debugging
	fmt.Printf("[SYSCALL %d] %s\n", syscallNum, name)
}

// LogReturn logs a syscall return value
func (st *SyscallTracer) LogReturn(syscallNum uint64, ret uint64, err error) {
	if !st.enabled {
		return
	}

	name := st.getSyscallName(syscallNum)
	if err != nil {
		fmt.Printf("[SYSCALL %d] %s -> ERROR: %v\n", syscallNum, name, err)
		if st.logFile != nil {
			fmt.Fprintf(st.logFile, "[SYSCALL %d] %s -> ERROR: %v\n", syscallNum, name, err)
		}
	} else {
		fmt.Printf("[SYSCALL %d] %s -> %d\n", syscallNum, name, ret)
		if st.logFile != nil {
			fmt.Fprintf(st.logFile, "[SYSCALL %d] %s -> %d\n", syscallNum, name, ret)
		}
	}
}

// LogInstruction logs an instruction execution (for detailed tracing)
func (st *SyscallTracer) LogInstruction(addr uint64, instructionCount int) {
	st.instructionCount++

	// Log every 1000 instructions to avoid spam
	if st.instructionCount%1000 == 0 {
		fmt.Printf("[TRACE] Executed %d instructions, current PC: 0x%x\n", st.instructionCount, addr)
	}
}

// getSyscallName returns the name of a syscall
func (st *SyscallTracer) getSyscallName(syscallNum uint64) string {
	names := map[uint64]string{
		0:   "SYSR1",
		1:   "_ERRSTR",
		2:   "BIND",
		3:   "CHDIR",
		4:   "CLOSE",
		5:   "DUP",
		6:   "ALARM",
		7:   "EXEC",
		8:   "EXITS",
		9:   "_FSESSION",
		10:  "FAUTH",
		11:  "_FSTAT",
		12:  "SEGBRK",
		13:  "_MOUNT",
		14:  "OPEN",
		15:  "_READ",
		16:  "OSEEK",
		17:  "SLEEP",
		18:  "_STAT",
		19:  "RFORK",
		20:  "_WRITE",
		21:  "PIPE",
		22:  "CREATE",
		23:  "FD2PATH",
		24:  "BRK_",
		25:  "REMOVE",
		26:  "_WSTAT",
		27:  "_FWSTAT",
		28:  "NOTIFY",
		29:  "NOTED",
		30:  "SEGATTACH",
		31:  "SEGDETACH",
		32:  "SEGFREE",
		33:  "SEGFLUSH",
		34:  "RENDEZVOUS",
		35:  "UNMOUNT",
		36:  "_WAIT",
		37:  "SEMACQUIRE",
		38:  "SEMRELEASE",
		39:  "SEEK",
		40:  "FVERSION",
		41:  "ERRSTR",
		42:  "STAT",
		43:  "FSTAT",
		44:  "WSTAT",
		45:  "FWSTAT",
		46:  "MOUNT",
		50:  "PREAD",
		51:  "PWRITE",
		52:  "TSEMACQUIRE",
		53:  "_NSEC",
		54:  "ACQUIRE",
		55:  "RELEASE",
		56:  "ALLOCMEM",
		57:  "FREEMEM",
		58:  "ATTACH",
	}

	if name, ok := names[syscallNum]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_%d", syscallNum)
}

// GetStats returns statistics about syscalls called
func (st *SyscallTracer) GetStats() map[uint64]int {
	return st.syscallCounts
}

// PrintSummary prints a summary of syscalls called
func (st *SyscallTracer) PrintSummary() {
	fmt.Println("\n=== Syscall Summary ===")
	for num, count := range st.syscallCounts {
		name := st.getSyscallName(num)
		fmt.Printf("  %s: %d calls\n", name, count)
	}
	fmt.Printf("Total instructions executed: %d\n", st.instructionCount)
}

// Close closes the tracer and its log file
func (st *SyscallTracer) Close() {
	if st.logFile != nil {
		st.logFile.Close()
	}
}
