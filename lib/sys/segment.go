package sys

import (
	"sync"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// Segment types
const (
	SEGTYPE_NONE = 0
	SEGTYPE_TEXT = 1
	SEGTYPE_DATA = 2
	SEGTYPE_STACK = 3
)

// Segment represents a memory segment
type Segment struct {
	ID      uint64
	Base    uint64
	Size    uint64
	Addr    uint64  // Virtual address
	Type    int
	Prot    int     // Protection flags
	Data    []byte  // For private segments
	Attached bool
}

// Protection flags
const (
	SEGPROT_READ  = 1 << 0
	SEGPROT_WRITE = 1 << 1
	SEGPROT_EXEC  = 1 << 2
)

// Global segment table
var (
	segments   = make(map[uint64]*Segment)
	nextSegID  uint64 = 1
	segMutex   sync.Mutex
)

// handleSegAttach implements SEGATTACH syscall (30)
// Attach a shared memory segment
func (k *Kernel) handleSegAttach(mu unicorn.Unicorn, rsp uint64) {
	// For now, this is a simplified implementation
	// A full implementation would support real shared memory

	addr, _ := readArg(mu, rsp, 0)    // Segment address to attach
	vaddr, _ := readArg(mu, rsp, 1)   // Virtual address where to attach
	size, _ := readArg(mu, rsp, 2)    // Size of segment
	// flags, _ := readArg(mu, rsp, 3) // Attachment flags

	// Create a new segment
	segMutex.Lock()
	segID := nextSegID
	nextSegID++
	segMutex.Unlock()

	seg := &Segment{
		ID:      segID,
		Base:    addr,
		Size:    size,
		Addr:    vaddr,
		Type:    SEGTYPE_DATA,
		Prot:    SEGPROT_READ | SEGPROT_WRITE,
		Attached: true,
	}

	// Map the segment if addr is not 0
	if addr != 0 {
		// Map into unicorn address space
		err := mu.MemMap(vaddr, size)
		if err != nil {
			k.setError(mu, "failed to map segment: "+err.Error())
			return
		}

		// If we have data at the base address, copy it
		data, err := mu.MemRead(addr, size)
		if err == nil {
			mu.MemWrite(vaddr, data)
		}
	}

	segMutex.Lock()
	segments[segID] = seg
	segMutex.Unlock()

	// Return segment ID
	setReturn(mu, segID)
}

// handleSegDetach implements SEGDETACH syscall (31)
// Detach a shared memory segment
func (k *Kernel) handleSegDetach(mu unicorn.Unicorn, rsp uint64) {
	segID, _ := readArg(mu, rsp, 0)

	segMutex.Lock()
	seg, ok := segments[segID]
	segMutex.Unlock()

	if !ok {
		k.setError(mu, "invalid segment id")
		return
	}

	// Unmap from address space
	if seg.Attached && seg.Addr != 0 {
		// Unmap the memory region from unicorn
		err := mu.MemUnmap(seg.Addr, seg.Size)
		if err != nil {
			k.setError(mu, "failed to unmap segment: "+err.Error())
			return
		}
	}

	seg.Attached = false
	setReturn(mu, 0)
}

// handleSegFree implements SEGFREE syscall (32)
// Free a memory segment
func (k *Kernel) handleSegFree(mu unicorn.Unicorn, rsp uint64) {
	segID, _ := readArg(mu, rsp, 0)
	addr, _ := readArg(mu, rsp, 1)

	segMutex.Lock()
	seg, ok := segments[segID]
	segMutex.Unlock()

	if !ok {
		k.setError(mu, "invalid segment id")
		return
	}

	// If addr is provided, free only that range
	// Otherwise, free the entire segment
	if addr != 0 && seg.Data != nil {
		// Free part of the segment (not implemented in simple version)
		setReturn(mu, 0)
		return
	}

	// Unmap and remove the segment
	if seg.Attached && seg.Addr != 0 {
		mu.MemUnmap(seg.Addr, seg.Size)
	}

	segMutex.Lock()
	delete(segments, segID)
	segMutex.Unlock()

	setReturn(mu, 0)
}

// handleSegFlush implements SEGFLUSH syscall (33)
// Flush a segment to backing store
func (k *Kernel) handleSegFlush(mu unicorn.Unicorn, rsp uint64) {
	segID, _ := readArg(mu, rsp, 0)
	addr, _ := readArg(mu, rsp, 1)
	offset, _ := readArg(mu, rsp, 2)
	size, _ := readArg(mu, rsp, 3)

	segMutex.Lock()
	seg, ok := segments[segID]
	segMutex.Unlock()

	if !ok {
		k.setError(mu, "invalid segment id")
		return
	}

	// If we have a private segment with data, flush it
	if seg.Data != nil && addr != 0 {
		// Flush data to the segment
		// Calculate actual offset within segment
		if offset < seg.Size {
			endAddr := offset + size
			if endAddr > seg.Size {
				endAddr = seg.Size
			}

			// Read from virtual memory and update segment data
			data, err := mu.MemRead(addr+offset, endAddr-offset)
			if err == nil {
				copy(seg.Data[offset:endAddr], data)
			}
		}
	}

	setReturn(mu, 0)
}
