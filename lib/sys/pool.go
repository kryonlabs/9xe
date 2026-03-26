package sys

import (
	"fmt"
	"sync"
)

// Pool implements 9front's pool allocator
// Pools are used for efficient allocation of fixed-size blocks
// Based on 9front's pool implementation in mem.c
type Pool struct {
	name      string
	maxsize   uintptr    // Maximum total size
	cursize   uintptr    // Current total size
	quantum   uintptr    // Allocation quantum (alignment)
	minblock  uintptr    // Minimum block size
	arenas    []*Arena   // Memory arenas
	mu        sync.Mutex
}

// Arena represents a contiguous region of memory for pool allocations
type Arena struct {
	base    uintptr     // Base address
	size    uintptr     // Total size
	free    []byte      // Free block bitmap (simplified)
	locks   []byte      // Lock bits for concurrent access
	blocks  *Block      // Free blocks list (linked list)
	mu      sync.Mutex
}

// Block represents a free block in an arena
type Block struct {
	addr    uintptr     // Start address
	size    uintptr     // Block size
	used    bool        // In-use flag
	next    *Block      // Next free block
}

// Pool constants
const (
	// Default pool configuration
	DefaultMaxSize   = 16 * 1024 * 1024  // 16MB default max
	DefaultQuantum   = 8                 // 8-byte alignment
	DefaultMinBlock  = 32                // 32-byte minimum block

	// Arena sizes
	ArenaSize       = 1024 * 1024        // 1MB per arena
	MaxArenaSize    = 4 * 1024 * 1024    // 4MB maximum arena
)

// NewPool creates a new memory pool
func NewPool(name string, maxsize, quantum, minblock uintptr) *Pool {
	if maxsize == 0 {
		maxsize = DefaultMaxSize
	}
	if quantum == 0 {
		quantum = DefaultQuantum
	}
	if minblock == 0 {
		minblock = DefaultMinBlock
	}

	return &Pool{
		name:      name,
		maxsize:   maxsize,
		quantum:   quantum,
		minblock:  minblock,
		arenas:    make([]*Arena, 0),
	}
}

// Alloc allocates memory from the pool
func (p *Pool) Alloc(size uintptr) (uintptr, error) {
	// Round up to quantum boundary
	size = (size + p.quantum - 1) &^ (p.quantum - 1)

	// Ensure minimum size
	if size < p.minblock {
		size = p.minblock
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to allocate from existing arenas
	for _, arena := range p.arenas {
		if addr, ok := p.allocFromArena(arena, size); ok {
			return addr, nil
		}
	}

	// Need to create a new arena
	arenaSize := uintptr(ArenaSize)
	if arenaSize < size {
		arenaSize = size
	}
	if arenaSize > uintptr(MaxArenaSize) {
		arenaSize = uintptr(MaxArenaSize)
	}

	// Check maxsize limit
	if p.cursize+arenaSize > p.maxsize {
		return 0, fmt.Errorf("pool %s: exceeded maximum size", p.name)
	}

	// Create new arena
	arena := p.newArena(arenaSize)
	p.arenas = append(p.arenas, arena)
	p.cursize += arenaSize

	// Allocate from new arena
	addr, ok := p.allocFromArena(arena, size)
	if !ok {
		return 0, fmt.Errorf("pool %s: failed to allocate from new arena", p.name)
	}

	fmt.Printf("[pool] %s: allocated %d bytes at 0x%x (arena size %d)\n",
		p.name, size, addr, arenaSize)

	return addr, nil
}

// allocFromArena tries to allocate from a specific arena
func (p *Pool) allocFromArena(arena *Arena, size uintptr) (uintptr, bool) {
	arena.mu.Lock()
	defer arena.mu.Unlock()

	// Try free blocks list first
	for block := arena.blocks; block != nil; block = block.next {
		if !block.used && block.size >= size {
			// Found a suitable block
			block.used = true

			// If block is much larger than needed, split it
			if block.size >= size*2 {
				p.splitBlock(arena, block, size)
			}

			return block.addr, true
		}
	}

	return 0, false
}

// splitBlock splits a block into two
func (p *Pool) splitBlock(arena *Arena, block *Block, size uintptr) {
	// Create new block for remaining space
	remaining := block.size - size
	if remaining < p.minblock {
		return // Too small to split
	}

	newBlock := &Block{
		addr:   block.addr + size,
		size:   remaining,
		used:   false,
		next:   block.next,
	}

	block.size = size
	block.next = newBlock
}

// Free frees memory back to the pool
func (p *Pool) Free(addr uintptr) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, arena := range p.arenas {
		arena.mu.Lock()
		for block := arena.blocks; block != nil; block = block.next {
			if block.addr == addr && block.used {
				block.used = false

				// Try to merge with adjacent free blocks
				p.coalesceBlocks(arena)

				fmt.Printf("[pool] %s: freed block at 0x%x\n", p.name, addr)
				return
			}
		}
		arena.mu.Unlock()
	}

	fmt.Printf("[pool] %s: warning - double free or invalid address 0x%x\n", p.name, addr)
}

// coalesceBlocks merges adjacent free blocks
func (p *Pool) coalesceBlocks(arena *Arena) {
	for block := arena.blocks; block != nil; block = block.next {
		if !block.used && block.next != nil && !block.next.used {
			// Merge with next block
			block.size += block.next.size
			block.next = block.next.next
		}
	}
}

// newArena creates a new memory arena
func (p *Pool) newArena(size uintptr) *Arena {
	// In a real implementation, this would allocate actual memory
	// For emulation, we just track the metadata
	arena := &Arena{
		base:   0, // Will be set by brk
		size:   size,
		blocks: &Block{
			addr:   0, // Will be set when allocated
			size:   size,
			used:   false,
			next:   nil,
		},
	}

	return arena
}

// PoolManager manages all pools in the system
type PoolManager struct {
	pools  map[string]*Pool
	mainPool *Pool  // Main memory pool
	brk    uintptr     // Current break address
	mu     sync.Mutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	pm := &PoolManager{
		pools: make(map[string]*Pool),
	}

	// Create main memory pool
	pm.mainPool = NewPool("main", DefaultMaxSize, DefaultQuantum, DefaultMinBlock)
	pm.pools["main"] = pm.mainPool

	return pm
}

// GetPool gets or creates a named pool
func (pm *PoolManager) GetPool(name string) *Pool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pool, ok := pm.pools[name]; ok {
		return pool
	}

	pool := NewPool(name, DefaultMaxSize, DefaultQuantum, DefaultMinBlock)
	pm.pools[name] = pool
	return pool
}

// SetBrk sets the break address for the main pool
func (pm *PoolManager) SetBrk(addr uintptr) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.brk = addr
}

// GetBrk returns the current break address
func (pm *PoolManager) GetBrk() uintptr {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.brk
}

// Malloc allocates memory from the main pool
func (pm *PoolManager) Malloc(size uintptr) (uintptr, error) {
	pool := pm.GetPool("main")
	addr, err := pool.Alloc(size)
	if err != nil {
		return 0, err
	}

	return addr, nil
}

// Free frees memory back to the main pool
func (pm *PoolManager) Free(addr uintptr) {
	pool := pm.GetPool("main")
	pool.Free(addr)
}
