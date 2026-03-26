package sys

import (
	"fmt"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// handleNotify implements NOTIFY syscall (28)
// Used for parent-child process notifications in Plan 9
func (k *Kernel) handleNotify(mu unicorn.Unicorn, rsp uint64) {
	arg0, _ := readArg(mu, rsp, 0)

	fmt.Printf("[sys] NOTIFY(pid=%d)\n", arg0)

	// Send notification to the specified process
	if k.processMgr != nil {
		k.processMgr.Notify(arg0)
	}

	setReturn(mu, 1) // Success
}

// handleNoted implements NOTED syscall (29)
// Checks if parent notification has been received
// This is what the polling loop is waiting for!
func (k *Kernel) handleNoted(mu unicorn.Unicorn, rsp uint64) {
	arg0, _ := readArg(mu, rsp, 0)

	if k.processMgr != nil {
		// Check if parent notification has been received
		notified := k.processMgr.WaitForParentNotification(mu)
		if notified {
			fmt.Printf("[sys] NOTED(tag=%d) -> Parent notification received!\n", arg0)
			setReturn(mu, 1) // Success - notification received
			return
		}
	}

	fmt.Printf("[sys] NOTED(tag=%d) -> Waiting for parent notification...\n", arg0)
	setReturn(mu, 0) // Not ready yet
}
