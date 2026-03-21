package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/kryonlabs/9xe/lib/aout"
	"github.com/kryonlabs/9xe/lib/sys" // Ensure this path matches your go.mod
	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: 9xe <path_to_plan9_binary>")
		return
	}

	// 1. Open the Plan 9 Binary
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	// 2. Parse the Header (Now using your dynamic CalculateMagic)
	hdr, err := aout.ReadHeader(f)
	if err != nil {
		log.Fatalf("Error parsing header: %v", err)
	}

	fmt.Printf("--- 9xe Executive: TaijiOS Loader ---\n")
	fmt.Printf("Architecture: %s\n", hdr.GetArchitecture())
	fmt.Printf("Magic:        0x%x\n", hdr.Magic)
	fmt.Printf("Entry Point:  0x%x\n", hdr.Entry)
	fmt.Printf("Text Segment: %d bytes\n", hdr.Text)
	fmt.Printf("Data Segment: %d bytes\n", hdr.Data)
	fmt.Printf("--------------------------------------\n")

	// 3. Prepare Virtual Memory
	const BaseAddr = 0x200000
	const MemSize = 64 * 1024 * 1024 // 64MB Virtual RAM
	virtRAM := make([]byte, MemSize)

	// Seek past the 32-byte Plan 9 Header
	if _, err := f.Seek(32, 0); err != nil {
		log.Fatalf("Seek failed: %v", err)
	}

	// Load Text (Code) Segment
	textSegment := virtRAM[0:hdr.Text]
	if _, err := io.ReadFull(f, textSegment); err != nil {
		log.Fatalf("Failed to read Text segment: %v (Expected %d bytes)", err, hdr.Text)
	}

	// Load Data (Variables) Segment - Aligned to 4KB Page
	dataOffset := (hdr.Text + 4095) &^ 4095
	dataSegment := virtRAM[dataOffset : dataOffset+hdr.Data]
	if _, err := io.ReadFull(f, dataSegment); err != nil {
		log.Fatalf("Failed to read Data segment: %v (Expected %d bytes)", err, hdr.Data)
	}

	fmt.Printf("Memory: Text mapped at 0x%x, Data mapped at 0x%x\n", BaseAddr, BaseAddr+uint64(dataOffset))

	// 4. Ignite the CPU
	mu, err := unicorn.NewUnicorn(unicorn.ARCH_X86, unicorn.MODE_64)
	if err != nil {
		log.Fatalf("Failed to initialize Unicorn: %v", err)
	}

	// Map memory into the CPU
	if err := mu.MemMap(BaseAddr, MemSize); err != nil {
		log.Fatalf("Failed to map CPU memory: %v", err)
	}

	// Write our loaded segments into the CPU's memory
	if err := mu.MemWrite(BaseAddr, virtRAM); err != nil {
		log.Fatalf("Failed to write to CPU memory: %v", err)
	}

	// Set the Stack Pointer (RSP) - end of RAM
	if err := mu.RegWrite(unicorn.X86_REG_RSP, BaseAddr+MemSize-4096); err != nil {
		log.Fatalf("Failed to set Stack Pointer: %v", err)
	}

	// Set the Instruction Pointer (RIP) to the Entry Point
	if err := mu.RegWrite(unicorn.X86_REG_RIP, uint64(hdr.Entry)); err != nil {
		log.Fatalf("Failed to set RIP: %v", err)
	}

	// 5. Syscall Bridge Hook
	// We use HOOK_CODE to manually detect the 2-byte SYSCALL opcode (0x0F 0x05)
	// This avoids issues with different Unicorn library builds.
	mu.HookAdd(unicorn.HOOK_CODE, func(mu unicorn.Unicorn, addr uint64, size uint32) {
		instruction, _ := mu.MemRead(addr, uint64(size))

		if len(instruction) >= 2 && instruction[0] == 0x0F && instruction[1] == 0x05 {
			// Delegate to your lib/sys package
			sys.Handle(mu)
		}
	}, 1, 0)

	// 6. Trace Hook for debugging visibility
	mu.HookAdd(unicorn.HOOK_CODE, func(mu unicorn.Unicorn, addr uint64, size uint32) {
		fmt.Printf("[Trace] Executing at 0x%x (size %d)\n", addr, size)
	}, 1, 0)

	fmt.Printf("CPU: Starting execution at 0x%x...\n", hdr.Entry)
	err = mu.Start(uint64(hdr.Entry), 0)
	if err != nil {
		fmt.Printf("\n[9xe] Emulation halted: %v\n", err)
		rax, _ := mu.RegRead(unicorn.X86_REG_RAX)
		fmt.Printf("[9xe] Last RAX (Syscall ID): %d\n", rax)
	}
}
