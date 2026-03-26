package sys

import (
	"fmt"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// RFORK flags
const (
	RFNAMEG  = 1 << 0 // detach name space
	RFENVG   = 1 << 1 // detach environment
	RFFDG    = 1 << 2 // detach file descriptor table
	RFNOTEG  = 1 << 3 // detach process namespace
	RFPROC   = 1 << 4 // fork a new process
	RFMEM    = 1 << 5 // fork memory
	RFNOWAIT = 1 << 6 // parent does not wait
)

// handleRfork implements RFORK syscall (19)
func (k *Kernel) handleRfork(mu unicorn.Unicorn, rsp uint64) {
	flags, _ := readArg(mu, rsp, 0)

	fmt.Printf("[sys] RFORK(flags=0x%x)\n", flags)

	// For now, we only handle simple fork (RFPROC)
	// In a real implementation, we'd handle all the flag combinations

	if flags&RFPROC != 0 {
		// Fork a new process - NOT YET IMPLEMENTED
		// For now, just return 0 (child) to simulate single-process behavior
		fmt.Printf("[sys] RFORK: Process forking not yet implemented, returning 0\n")
		setReturn(mu, 0) // Return 0 (child process)
	} else {
		// Same process - just manipulate current process flags
		setReturn(mu, 0)
	}
}
