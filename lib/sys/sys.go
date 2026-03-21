package sys

import (
	"fmt"
	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// IDs from your sys.h
const (
	_ERRSTR    = 1
	BIND       = 2
	CHDIR      = 3
	CLOSE      = 4
	EXITS      = 8
	OPEN       = 14
	_READ      = 15
	_WRITE     = 20
	SEEK       = 39
	PREAD      = 50
	PWRITE     = 51
)

// Handle handles the trapped syscall based on the RAX value
func Handle(mu unicorn.Unicorn) {
	rax, _ := mu.RegRead(unicorn.X86_REG_RAX)
	rip, _ := mu.RegRead(unicorn.X86_REG_RIP)

	fmt.Printf("\n[9xe Kernel] SYSCALL Trapped: ID %d at 0x%x\n", rax, rip)

	switch rax {
	case EXITS:
		handleExits(mu)
	case OPEN:
		handleOpen(mu)
	case _WRITE:
		handleWrite(mu)
	default:
		fmt.Printf(" !!! Warning: Unimplemented Syscall %d. Skipping...\n", rax)
	}

	// Always advance RIP by 2 bytes (the size of the SYSCALL instruction)
	mu.RegWrite(unicorn.X86_REG_RIP, rip+2)
}

func handleExits(mu unicorn.Unicorn) {
	// EXITS takes a status string
	fmt.Println(" -> Process exiting...")
	mu.Stop()
}

func handleOpen(mu unicorn.Unicorn) {
	// Plan 9 OPEN: path (string), mode (int)
	fmt.Println(" -> SYS_OPEN requested")
}

func handleWrite(mu unicorn.Unicorn) {
	// Plan 9 WRITE: fd (int), buf (void*), n (long)
	fmt.Println(" -> SYS_WRITE requested")
}
