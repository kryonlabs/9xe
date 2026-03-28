package sys

import (
	"encoding/binary"
	"fmt"
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

	// Read symbol table to find main() function
	var mainAddr uint64 = uint64(hdr.Entry) // Default to entry point
	if hdr.Syms > 0 {
		symTableOffset := int64(32 + hdr.Text + hdr.Data)
		if _, err := f.Seek(symTableOffset, 0); err == nil {
			symbols, err := aout.ReadSymbolTable(f, hdr.Syms)
			if err == nil {
				// Try to find main function
				if foundMain := aout.FindMainSymbol(symbols, argv0); foundMain != 0 {
					mainAddr = foundMain
					fmt.Printf("[exec] Found main symbol at 0x%x\n", mainAddr)
				} else {
					fmt.Printf("[exec] Main symbol not found, using entry point 0x%x\n", uint64(hdr.Entry))
				}
			}
		}
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

	// Set up proper argv array for the new process
	rspVal, _ := mu.RegRead(unicorn.X86_REG_RSP)
	stackTop := rspVal

	// Reserve space for argv array on stack
	// Plan 9 expects: [argc][argv[0]][argv[1]]...[argv[n-1]][NULL]
	argvAddrs := make([]uint64, len(argv))
	currentStack := stackTop - 0x100 // Reserve stack space

	// First, write all argument strings to stack
	for i, arg := range argv {
		argBytes := []byte(arg + "\x00")
		currentStack -= uint64(len(argBytes))
		currentStack &= ^uint64(7) // 8-byte align
		mu.MemWrite(currentStack, argBytes)
		argvAddrs[i] = currentStack
	}

	// Now write argv array (pointers to strings)
	argvArrayStart := currentStack - uint64((len(argv)+1)*8)
	argvArrayStart &= ^uint64(7)

	for i, addr := range argvAddrs {
		addrBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(addrBytes, addr)
		mu.MemWrite(argvArrayStart+uint64(i*8), addrBytes)
	}

	// Write NULL terminator at end of argv array
	nullTerm := make([]byte, 8)
	mu.MemWrite(argvArrayStart+uint64(len(argv)*8), nullTerm)

	// Write argc at a known location (standard Plan 9 location)
	argcBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(argcBuf, uint64(len(argv)))
	mu.MemWrite(stackTop-0x100, argcBuf)

	// Set up registers for new process
	mu.RegWrite(unicorn.X86_REG_RIP, uint64(hdr.Entry)) // Entry point
	mu.RegWrite(unicorn.X86_REG_RCX, mainAddr)          // main() function
	mu.RegWrite(unicorn.X86_REG_RDX, argvArrayStart)     // argv array pointer
	mu.RegWrite(unicorn.X86_REG_R8, uint64(len(argv)))   // argc

	fmt.Printf("[exec] Starting process: argv[0]=%s, argc=%d\n", argv[0], len(argv))
	fmt.Printf("[exec] Entry point: 0x%x, main: 0x%x, argv: 0x%x\n", uint64(hdr.Entry), mainAddr, argvArrayStart)

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
