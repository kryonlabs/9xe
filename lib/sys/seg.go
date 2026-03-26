package sys

import (
	"fmt"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// Segment constants for Plan 9 memory model
const (
	SEG_TEXT   = 0  // Text segment
	SEG_DATA   = 1  // Data segment
	SEG_BSS    = 2  // BSS segment
	SEG_STACK  = 3  // Stack segment
	SEG_SHARED = 4  // Shared segment
)

// handleSegattach implements the SEGATTACH syscall (30)
// Attaches a new memory segment to the process
// Format: segattach(addr, type, va, len, flags) - stub
func (k *Kernel) handleSegattach(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 8)
	// typ, _ := readArg(mu, rsp, 16)
	// va, _ := readArg(mu, rsp, 24)
	// len, _ := readArg(mu, rsp, 32)
	// flags, _ := readArg(mu, rsp, 40)

	fmt.Printf("[sys] SEGATTACH(addr=0x%x) - stub implementation\n", addr)

	// For now, just return a fake address
	// In a full implementation, this would:
	// 1. Validate the segment type
	// 2. Allocate virtual memory
	// 3. Map the segment into the process address space
	// 4. Return the base address

	setReturn(mu, 0x40000000)  // Return fake segment address
}

// handleSegdetach implements the SEGDETACH syscall (31)
// Detaches a memory segment from the process
// Format: segdetach(addr) - stub
func (k *Kernel) handleSegdetach(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 8)

	fmt.Printf("[sys] SEGDETACH(addr=0x%x) - stub implementation\n", addr)

	// For now, just return success
	// In a full implementation, this would:
	// 1. Find the segment at addr
	// 2. Unmap it from the process address space
	// 3. Free the resources

	setReturn(mu, 0)
}

// handleSegfree implements the SEGFREE syscall (32)
// Frees a memory segment
// Format: segfree(addr) - stub
func (k *Kernel) handleSegfree(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 8)

	fmt.Printf("[sys] SEGFREE(addr=0x%x) - stub implementation\n", addr)

	// For now, just return success
	// In a full implementation, this would:
	// 1. Find the segment at addr
	// 2. Free the memory
	// 3. Remove it from the process

	setReturn(mu, 0)
}

// handleSegflush implements the SEGFLUSH syscall (33)
// Flushes a memory segment (ensures consistency)
// Format: segflush(addr, len) - stub
func (k *Kernel) handleSegflush(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 8)
	// len, _ := readArg(mu, rsp, 16)

	fmt.Printf("[sys] SEGFLUSH(addr=0x%x) - stub implementation\n", addr)

	// For now, just return success
	// In a full implementation, this would:
	// 1. Find the segment at addr
	// 2. Flush any cached data
	// 3. Ensure memory consistency

	setReturn(mu, 0)
}
