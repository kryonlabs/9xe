package sys

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/kryonlabs/9xe/lib/aout"
	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// handleExec implements EXEC syscall (7)
// Format: exec(argv0, argv, label) - execute new binary
func (k *Kernel) handleExec(mu unicorn.Unicorn, rsp uint64) {
	argv0Ptr, _ := readArg(mu, rsp, 0)
	argvPtr, _ := readArg(mu, rsp, 1)

	argv0, err := readString(mu, argv0Ptr, 1024)
	if err != nil {
		k.setError(mu, "bad argv0 ptr")
		return
	}

	// Read argv array from memory
	argv := k.readStringArray(mu, argvPtr)

	if len(argv) == 0 {
		k.setError(mu, "no argv")
		return
	}

	// Open the binary file
	f, err := os.Open(argv0)
	if err != nil {
		k.setError(mu, "cannot open binary: "+err.Error())
		return
	}
	defer f.Close()

	// Parse the a.out header
	hdr, err := aout.ReadHeader(f)
	if err != nil {
		k.setError(mu, "bad a.out format: "+err.Error())
		return
	}

	// Seek past the 32-byte header
	if _, err := f.Seek(32, 0); err != nil {
		k.setError(mu, "seek failed: "+err.Error())
		return
	}

	// Constants
	const segAlign = 0x200000
	const BaseAddr = 0x200000

	// Load text segment
	textSegment := make([]byte, hdr.Text)
	if _, err := io.ReadFull(f, textSegment); err != nil {
		k.setError(mu, "failed to read text: "+err.Error())
		return
	}

	// Calculate data segment offset (immediately after text, no alignment)
	dataOffset := hdr.Text

	// Load data segment
	dataSegment := make([]byte, hdr.Data)
	if _, err := io.ReadFull(f, dataSegment); err != nil {
		k.setError(mu, "failed to read data: "+err.Error())
		return
	}

	// Write to memory
	mu.MemWrite(BaseAddr, textSegment)
	mu.MemWrite(BaseAddr+uint64(dataOffset), dataSegment)

	// For now, use entry point as main address
	// TODO: Implement proper symbol table lookup
	mainAddr := uint64(hdr.Entry)

	// Set up simplified argv (just argc for now)
	rspVal, _ := mu.RegRead(unicorn.X86_REG_RSP)
	stackTop := rspVal

	// Write argc at RSP+0xb0
	argcBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(argcBuf, uint32(len(argv)))
	mu.MemWrite(stackTop+0xb0, argcBuf)

	// Set registers
	mu.RegWrite(unicorn.X86_REG_RIP, uint64(hdr.Entry)) // Entry point
	mu.RegWrite(unicorn.X86_REG_RCX, mainAddr)          // main() function

	// Success
	setReturn(mu, 0)
}

// readStringArray reads an array of string pointers from memory
func (k *Kernel) readStringArray(mu unicorn.Unicorn, addr uint64) []string {
	var argv []string

	for i := 0; ; i++ {
		ptrBytes, err := mu.MemRead(addr+uint64(i*8), 8)
		if err != nil {
			break
		}

		ptr := binary.LittleEndian.Uint64(ptrBytes)
		if ptr == 0 {
			break
		}

		str, err := readString(mu, ptr, 1024)
		if err != nil {
			break
		}

		argv = append(argv, str)
	}

	return argv
}
