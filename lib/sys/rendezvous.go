package sys

import (
	"fmt"
	"sync"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// Rendezvous point for process synchronization
// Processes can rendezvous at a tag and exchange values
type Rendezvous struct {
	tag      uint64        // Unique identifier for this rendezvous
	value    uint64        // Value being exchanged
	waiting  map[uint64]bool // PIDs of processes waiting
	mu       sync.Mutex
}

// RendezManager manages all rendezvous points
type RendezManager struct {
	mu     sync.Mutex
	points map[uint64]*Rendezvous  // tag -> rendezvous point
}

// NewRendezManager creates a new rendezvous manager
func NewRendezManager() *RendezManager {
	return &RendezManager{
		points: make(map[uint64]*Rendezvous),
	}
}

// Rendezvous implements the RENDEZVOUS syscall (34)
// Processes rendezvous at a tag and exchange a value
// If two processes rendezvous with the same tag, they exchange values
// Returns: the value from the other process
func (rm *RendezManager) Rendezvous(tag uint64, value uint64) uint64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Find or create rendezvous point
	r, exists := rm.points[tag]
	if !exists {
		// First caller - create rendezvous point and wait
		r = &Rendezvous{
			tag:     tag,
			value:   value,
			waiting: make(map[uint64]bool),
		}
		rm.points[tag] = r

		// Mark that we're waiting
		// In a real implementation, this would block
		// For now, we'll return the value immediately (simple rendezvous)
		fmt.Printf("[rendezvous] First process at tag 0x%x, waiting with value 0x%x\n", tag, value)

		// Simple implementation: just return the value
		// This works for single-process emulation
		return value
	}

	// Second caller - exchange values
	otherValue := r.value
	r.value = value

	fmt.Printf("[rendezvous] Second process at tag 0x%x, exchanged 0x%x for 0x%x\n",
		tag, value, otherValue)

	// Clean up rendezvous point
	delete(rm.points, tag)

	return otherValue
}

// handleRendezvous implements the RENDEZVOUS syscall (34)
// Format: rendezvous(tag, value) - rendezvous at tag, exchange value
// Returns: value from other process
func (k *Kernel) handleRendezvous(mu unicorn.Unicorn, rsp uint64) {
	tag, _ := readArg(mu, rsp, 0)
	value, _ := readArg(mu, rsp, 8)

	fmt.Printf("[sys] RENDEZVOUS(tag=0x%x, value=0x%x) for PID %d\n",
		tag, value, k.currentPID)

	if k.rendezMgr == nil {
		fmt.Printf("[sys] ERROR: Rendezvous manager not initialized\n")
		setReturn(mu, ^uint64(0))  // Error
		return
	}

	result := k.rendezMgr.Rendezvous(tag, value)
	setReturn(mu, result)
}
