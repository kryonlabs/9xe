package sys

import (
	"fmt"
	"sync"
	"time"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// AlarmManager handles process alarms and timer delivery
type AlarmManager struct {
	mu         sync.Mutex
	alarms     map[uint64]*alarmEntry  // pid -> alarm entry
	processMgr *ProcessManager         // For posting alarm notes
}

type alarmEntry struct {
	timer    *time.Timer
	deadline time.Time
	pid      uint64
	mgr      *AlarmManager
}

// NewAlarmManager creates a new alarm manager
func NewAlarmManager() *AlarmManager {
	return &AlarmManager{
		alarms: make(map[uint64]*alarmEntry),
	}
}

// SetProcessManager sets the process manager for alarm delivery
func (am *AlarmManager) SetProcessManager(pm *ProcessManager) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.processMgr = pm
}

// SetAlarm sets an alarm for the specified process
// millis: milliseconds until alarm (0 = cancel)
// Returns: previous alarm time in milliseconds, or 0 if none
func (am *AlarmManager) SetAlarm(pid uint64, millis uint64) uint64 {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Cancel existing alarm if any
	if entry, exists := am.alarms[pid]; exists {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(am.alarms, pid)
	}

	// Zero means cancel alarm
	if millis == 0 {
		return 0
	}

	// Create new alarm
	deadline := time.Now().Add(time.Duration(millis) * time.Millisecond)
	entry := &alarmEntry{
		deadline: deadline,
		pid:      pid,
		mgr:      am,
	}

	// Set up timer to deliver alarm note
	entry.timer = time.AfterFunc(time.Duration(millis)*time.Millisecond, func() {
		// Alarm fired - post note to process
		fmt.Printf("[alarm] Alarm fired for PID %d\n", pid)
		if am.processMgr != nil {
			am.processMgr.PostNote(pid, "alarm")
		}
	})

	am.alarms[pid] = entry
	return 0
}

// GetAlarmTime returns the deadline for a process's alarm
// Returns: deadline in milliseconds since epoch, 0 if no alarm
func (am *AlarmManager) GetAlarmTime(pid uint64) uint64 {
	am.mu.Lock()
	defer am.mu.Unlock()

	if entry, exists := am.alarms[pid]; exists {
		return uint64(entry.deadline.UnixMilli())
	}
	return 0
}

// CancelAlarm cancels a process's alarm
func (am *AlarmManager) CancelAlarm(pid uint64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if entry, exists := am.alarms[pid]; exists {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(am.alarms, pid)
	}
}

// handleAlarm implements the ALARM syscall (6)
// Format: alarm(millis) - sets alarm timer
// Returns: previous alarm time in milliseconds
func (k *Kernel) handleAlarm(mu unicorn.Unicorn, rsp uint64) {
	millis, _ := readArg(mu, rsp, 0)

	fmt.Printf("[sys] ALARM(%d) for PID %d\n", millis, k.currentPID)

	if k.alarmMgr != nil {
		k.alarmMgr.SetAlarm(k.currentPID, millis)
	}

	setReturn(mu, 0)  // Return 0 = no previous alarm
}
