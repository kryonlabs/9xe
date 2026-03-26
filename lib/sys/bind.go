package sys

import (
	"fmt"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// handleBind implements BIND syscall (2)
// Bind a new device over an existing file
func (k *Kernel) handleBind(mu unicorn.Unicorn, rsp uint64) {
	// Plan 9 BIND creates a new name for a file
	// bind(newname oldname flag) -> bind oldname onto newname
	// For now, we return success but don't actually implement binding
	// since this is typically used for mounting filesystems

	_, _ = readArg(mu, rsp, 0) // newname
	_, _ = readArg(mu, rsp, 1) // oldname
	_, _ = readArg(mu, rsp, 2) // flags

	// TODO: Implement proper binding
	// For now, return success
	fmt.Printf("[sys] bind: not fully implemented, returning success\n")
	setReturn(mu, 0)
}

// handleMount implements MOUNT syscall (46)
// Mount a file server on a directory
func (k *Kernel) handleMount(mu unicorn.Unicorn, rsp uint64) {
	// Plan 9 MOUNT connects to a file server
	// mount(fd, addr, flags, aname) -> mount file server at addr on fd
	// For now, we just return success

	_, _ = readArg(mu, rsp, 0) // fd
	_, _ = readArg(mu, rsp, 1) // addr
	_, _ = readArg(mu, rsp, 2) // flags
	_, _ = readArg(mu, rsp, 3) // aname

	// TODO: Implement proper mounting
	// For now, return success
	fmt.Printf("[sys] mount: not fully implemented, returning success\n")
	setReturn(mu, 0)
}

// handleFauth implements FAUTH syscall (10)
// Authenticate an open file
func (k *Kernel) handleFauth(mu unicorn.Unicorn, rsp uint64) {
	// Plan 9 FAUTH authenticates a file for subsequent operations
	// fauth(fd, aname, uid) -> returns authentication ticket
	// For now, we just return success (no authentication in emulator)

	_, _ = readArg(mu, rsp, 0) // fd
	_, _ = readArg(mu, rsp, 1) // aname
	_, _ = readArg(mu, rsp, 2) // uid

	// TODO: Implement proper authentication
	// For now, return success with no ticket
	setReturn(mu, 0)
}

// handleFversion implements FVERSION syscall (40)
// Negotiate protocol version with a file server
func (k *Kernel) handleFversion(mu unicorn.Unicorn, rsp uint64) {
	// Plan 9 FVERSION negotiates protocol version
	// fversion(fd, msize, version) -> returns version string
	// For now, we return "9P2000" (default Plan 9 protocol version)

	_, _ = readArg(mu, rsp, 0) // fd
	msize, _ := readArg(mu, rsp, 1)
	versionPtr, _ := readArg(mu, rsp, 2)

	// Return "9P2000" as the version
	version := "9P2000"
	versionBytes := []byte(version + "\x00")

	if uint64(len(versionBytes)) > msize {
		k.setError(mu, "version string too long")
		setReturn(mu, ^uint64(0))
		return
	}

	mu.MemWrite(versionPtr, versionBytes)
	setReturn(mu, 0)
}

// handleUnmount implements UNMOUNT syscall (44)
// Unmount a file server from a directory
func (k *Kernel) handleUnmount(mu unicorn.Unicorn, rsp uint64) {
	// Plan 9 UNMOUNT disconnects a file server
	// unmount(addr) -> unmount file server at addr

	addr, _ := readArg(mu, rsp, 0) // addr

	// Read path string from memory at addr
	path, err := readString(mu, addr, 256)
	if err != nil {
		fmt.Printf("[sys] UNMOUNT: failed to read path from 0x%x: %v\n", addr, err)
		setReturn(mu, ^uint64(0)) // error
		return
	}

	fmt.Printf("[sys] UNMOUNT(path=%s)\n", path)

	// For now, just return success
	// In a full implementation, this would remove from mount table
	setReturn(mu, 0)
}
