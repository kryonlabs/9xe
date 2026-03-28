package sys

import (
	"sync"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// ProcessState represents the state of a Plan 9 process (matching 9front's Proc.state)
type ProcessState int

const (
	ProcessDead    ProcessState = iota // Process has exited
	ProcessMoribund                     // Process is dying
	ProcessNew                          // Newly created, not yet ready
	ProcessReady                        // Ready to run
	ProcessRunning                      // Currently running
	ProcessWaiting                      // Waiting for I/O or event
	ProcessZombie                       // Zombie process
	ProcessExited                       // Fully exited
)

// MemoryRegion represents a memory segment (text, data, bss, stack)
type MemoryRegion struct {
	Base    uint64 // Base address
	Size    uint64 // Size in bytes
	Perm    uint32 // Permissions (rwx)
	Ref     string // Reference identifier
}

// String returns a string representation of the process state
func (ps ProcessState) String() string {
	switch ps {
	case ProcessDead:
		return "Dead"
	case ProcessMoribund:
		return "Moribund"
	case ProcessNew:
		return "New"
	case ProcessReady:
		return "Ready"
	case ProcessRunning:
		return "Running"
	case ProcessWaiting:
		return "Waiting"
	case ProcessZombie:
		return "Zombie"
	case ProcessExited:
		return "Exited"
	default:
		return "Unknown"
	}
}

// Process represents a Plan 9 process (matching 9front's Proc structure)
type Process struct {
	PID       uint64
	ParentPID uint64
	State     ProcessState
	Notified  bool // Parent notification received

	// Notes/Signals (matching 9front's note mechanism)
	Notes      []string
	NoteLock   sync.Mutex

	// Alarms
	AlarmTime  uint64
	AlarmActive bool

	// Memory segments
	TextSeg    MemoryRegion
	DataSeg    MemoryRegion
	BssSeg     MemoryRegion
	StackSeg   MemoryRegion

	// Synchronization (for rendezvous)
	RendezTag  uint64
	RendezVal  uint64

	// Exit status
	ExitStatus int
}

// ProcessManager handles Plan 9 process lifecycle and notifications
type ProcessManager struct {
	mu           sync.Mutex
	processes    map[uint64]*Process
	nextPID      uint64
	currentPID   uint64
	initialized  bool
	noteQueue    chan Note // Channel for delivering notes
}

// Note represents a notification to be delivered to a process
type Note struct {
	PID  uint64
	Msg  string
}

// NewProcessManager creates a new process manager with init process (PID 1)
func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		processes:   make(map[uint64]*Process),
		nextPID:     2, // Next PID will be 2 (1 is init)
		currentPID:  1,
		initialized: false,
		noteQueue:   make(chan Note, 100), // Buffer for notes
	}

	// Create init process (PID 1)
	initProc := &Process{
		PID:       1,
		ParentPID: 0, // Init has no parent
		State:     ProcessRunning,
		Notified:  true, // Init is already "notified" since it has no parent
		Notes:     make([]string, 0),
	}
	pm.processes[1] = initProc

	return pm
}

// GetCurrentProcess returns the current running process
func (pm *ProcessManager) GetCurrentProcess() *Process {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.processes[pm.currentPID]
}

// CreateProcess creates a new process with the given parent
func (pm *ProcessManager) CreateProcess(parentPid uint64) *Process {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pid := pm.nextPID
	pm.nextPID++

	proc := &Process{
		PID:       pid,
		ParentPID: parentPid,
		State:     ProcessNew,
		Notified:  false,
		Notes:     make([]string, 0),
	}

	pm.processes[pid] = proc
	return proc
}

// SetProcessState changes the state of a process
func (pm *ProcessManager) SetProcessState(pid uint64, state ProcessState) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proc, ok := pm.processes[pid]; ok {
		proc.State = state
	}
}

// GetProcessState returns the state of a process
func (pm *ProcessManager) GetProcessState(pid uint64) ProcessState {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proc, ok := pm.processes[pid]; ok {
		return proc.State
	}
	return ProcessDead
}

// PostNote posts a note (notification) to a process
// This is the 9front equivalent of signal delivery
func (pm *ProcessManager) PostNote(pid uint64, note string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, ok := pm.processes[pid]
	if !ok {
		return false
	}

	proc.NoteLock.Lock()
	defer proc.NoteLock.Unlock()
	proc.Notes = append(proc.Notes, note)
	return true
}

// CheckNotes retrieves and clears pending notes for a process
// Returns the list of notes (may be empty)
func (pm *ProcessManager) CheckNotes(pid uint64) []string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, ok := pm.processes[pid]
	if !ok {
		return nil
	}

	proc.NoteLock.Lock()
	defer proc.NoteLock.Unlock()

	notes := proc.Notes
	proc.Notes = make([]string, 0)
	return notes
}

// HasPendingNotes checks if a process has pending notes
func (pm *ProcessManager) HasPendingNotes(pid uint64) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, ok := pm.processes[pid]
	if !ok {
		return false
	}

	proc.NoteLock.Lock()
	defer proc.NoteLock.Unlock()
	return len(proc.Notes) > 0
}

// WaitForParentNotification implements the Plan 9 parent notify mechanism
// The polling loop is waiting for this notification to be sent
func (pm *ProcessManager) WaitForParentNotification(mu unicorn.Unicorn) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc := pm.processes[pm.currentPID]
	if proc == nil {
		return false
	}

	// Check if already notified
	if proc.Notified {
		return true
	}

	// Mark as notified - simulating parent notification received
	// In a real Plan 9 system, this would block until notification arrives
	// For emulation, we mark it immediately to unblock the init loop
	proc.Notified = true

	// Write the notification value to RSI so the program sees it
	// The program is waiting for RSI to match [RSP+36]
	// After marking as notified, the program should exit the polling loop
	mu.RegWrite(unicorn.X86_REG_RSI, 1) // Non-zero value indicates notified

	return true
}

// Notify sends notification to a process (typically from parent to child)
func (pm *ProcessManager) Notify(pid uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc := pm.processes[pid]
	if proc != nil {
		proc.Notified = true
	}
}

// SendParentNotification sends parent notification to the current process
// This simulates the kernel notifying a process that its parent is ready
func (pm *ProcessManager) SendParentNotification() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc := pm.processes[pm.currentPID]
	if proc != nil {
		proc.Notified = true
	}
}

// SetAlarm sets an alarm for a process (in milliseconds)
func (pm *ProcessManager) SetAlarm(pid uint64, millis uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proc, ok := pm.processes[pid]; ok {
		proc.AlarmTime = millis
		proc.AlarmActive = true
	}
}

// ClearAlarm clears the alarm for a process
func (pm *ProcessManager) ClearAlarm(pid uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proc, ok := pm.processes[pid]; ok {
		proc.AlarmActive = false
	}
}

// IsInitialized returns whether the process manager has been initialized
func (pm *ProcessManager) IsInitialized() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.initialized
}

// MarkInitialized marks the process manager as initialized
func (pm *ProcessManager) MarkInitialized() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.initialized = true
}

// SetProcessExit sets the exit status for a process and marks it as exited
func (pm *ProcessManager) SetProcessExit(pid uint64, status int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proc, ok := pm.processes[pid]; ok {
		proc.ExitStatus = status
		proc.State = ProcessExited
	}
}

// handleFsession implements _FSESSION syscall (9) - File session
func (k *Kernel) handleFsession(mu unicorn.Unicorn, rsp uint64) {
	// For now, this is a stub
	// In Plan 9, this is used for session management with file servers
	// We just return success

	setReturn(mu, 0)
}
