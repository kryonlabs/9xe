package sys

import (
	"fmt"
	"sync"
	"time"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// TimedSemaphore represents a semaphore with timeout support
type TimedSemaphore struct {
	count    int32
	waiters  int
	mu       sync.Mutex
	cond     *sync.Cond
}

// TsemManager manages timed semaphores
type TsemManager struct {
	semaphores map[uint64]*TimedSemaphore
	mu         sync.Mutex
}

// NewTsemManager creates a new timed semaphore manager
func NewTsemManager() *TsemManager {
	tm := &TsemManager{
		semaphores: make(map[uint64]*TimedSemaphore),
	}
	return tm
}

// GetOrCreate gets or creates a semaphore
func (tm *TsemManager) GetOrCreate(addr uint64) *TimedSemaphore {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if sem, ok := tm.semaphores[addr]; ok {
		return sem
	}

	sem := &TimedSemaphore{
		count:   1, // Start with count of 1
		waiters: 0,
	}
	sem.cond = sync.NewCond(&sem.mu)

	tm.semaphores[addr] = sem
	return sem
}

// handleTsemacquire implements TSEMACQUIRE syscall (52)
// Timed semaphore acquire
// Format: tsemacquire(addr, milliseconds)
func (k *Kernel) handleTsemacquire(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 0)
	ms, _ := readArg(mu, rsp, 8)

	fmt.Printf("[sys] TSEMACQUIRE(addr=0x%x, ms=%d)\n", addr, ms)

	// Get or create semaphore
	if k.tsemMgr == nil {
		k.tsemMgr = NewTsemManager()
	}

	sem := k.tsemMgr.GetOrCreate(addr)

	// Try to acquire with timeout
	acquired := k.tryAcquire(sem, ms)

	if acquired {
		fmt.Printf("[sys] TSEMACQUIRE: acquired semaphore at 0x%x\n", addr)
		setReturn(mu, 1) // success
	} else {
		fmt.Printf("[sys] TSEMACQUIRE: timeout waiting for semaphore at 0x%x\n", addr)
		setReturn(mu, 0) // timeout
	}
}

// tryAcquire tries to acquire a semaphore with timeout
func (k *Kernel) tryAcquire(sem *TimedSemaphore, ms uint64) bool {
	sem.mu.Lock()
	defer sem.mu.Unlock()

	// Try immediate acquire
	if sem.count > 0 {
		sem.count--
		return true
	}

	// If no timeout, return immediately
	if ms == 0 {
		return false
	}

	// Wait with timeout
	acquired := make(chan bool, 1)
	sem.waiters++

	go func() {
		sem.cond.Wait()
		if sem.count > 0 {
			sem.count--
			acquired <- true
		} else {
			acquired <- false
		}
		sem.waiters--
	}()

	// Wait for acquisition or timeout
	timeout := time.After(time.Duration(ms) * time.Millisecond)

	select {
	case result := <-acquired:
		return result
	case <-timeout:
		// Timeout - signal to stop waiting
		sem.cond.Broadcast()
		return false
	}
}

// handleTsemrelease implements TSEMRELEASE (if it exists as a separate syscall)
// Release a semaphore
func (k *Kernel) handleTsemrelease(mu unicorn.Unicorn, rsp uint64) {
	addr, _ := readArg(mu, rsp, 0)

	fmt.Printf("[sys] TSEMRELEASE(addr=0x%x)\n", addr)

	if k.tsemMgr == nil {
		setReturn(mu, ^uint64(0)) // error
		return
	}

	k.tsemMgr.mu.Lock()
	sem, ok := k.tsemMgr.semaphores[addr]
	k.tsemMgr.mu.Unlock()

	if !ok {
		fmt.Printf("[sys] TSEMRELEASE: no semaphore at 0x%x\n", addr)
		setReturn(mu, ^uint64(0)) // error
		return
	}

	sem.mu.Lock()
	sem.count++
	sem.mu.Unlock()

	// Wake up one waiter
	sem.cond.Signal()

	fmt.Printf("[sys] TSEMRELEASE: released semaphore at 0x%x\n", addr)
	setReturn(mu, 0) // success
}
