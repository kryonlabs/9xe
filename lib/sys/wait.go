package sys

import (
	"fmt"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// handleWait implements WAIT syscall (36)
// Format: wait(pid) - waits for child process
// Returns: pid of child that exited, or 0 if no children
func (k *Kernel) handleWait(mu unicorn.Unicorn, rsp uint64) {
	// For now, we'll implement simple wait that returns immediately
	// In Plan 9, WAIT takes a pid pointer and writes [childpid, status, msg]

	fmt.Printf("[sys] WAIT - No children to wait for, returning error\n")

	// No children in single-process emulation
	k.setError(mu, "no children")
	setReturn(mu, ^uint64(0))
}
