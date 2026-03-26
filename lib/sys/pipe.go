package sys

import (
	"errors"
	"sync"

	"github.com/kryonlabs/9xe/lib/draw"
	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// Pipe represents a Unix-style pipe
type Pipe struct {
	data   []byte
	rwait  chan struct{}
	wwait  chan struct{}
	closed bool
	mu     sync.Mutex
}

// NewPipe creates a new pipe
func NewPipe() (*Pipe, *Pipe) {
	rpipe := &Pipe{
		data:    make([]byte, 0, 8192),
		rwait:   make(chan struct{}),
		wwait:   make(chan struct{}),
		closed:  false,
	}
	wpipe := &Pipe{
		data:    make([]byte, 0, 8192),
		rwait:   make(chan struct{}),
		wwait:   make(chan struct{}),
		closed:  false,
	}
	return rpipe, wpipe
}

// Read reads from the pipe
func (p *Pipe) Read(buf []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Wait for data if empty
	for len(p.data) == 0 && !p.closed {
		p.mu.Unlock()
		<-p.rwait
		p.mu.Lock()
	}

	if len(p.data) == 0 && p.closed {
		return 0, errors.New("EOF")
	}

	n := copy(buf, p.data)
	p.data = p.data[n:]

	// Signal writers
	select {
	case <-p.wwait:
	default:
		close(p.wwait)
		p.wwait = make(chan struct{})
	}

	return n, nil
}

// Write writes to the pipe
func (p *Pipe) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return 0, errors.New("write to closed pipe")
	}

	// Append to buffer
	p.data = append(p.data, data...)

	// Signal readers
	select {
	case <-p.rwait:
	default:
		close(p.rwait)
		p.rwait = make(chan struct{})
	}

	return len(data), nil
}

// Close closes the pipe
func (p *Pipe) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	close(p.rwait)
	return nil
}

// handlePipe implements PIPE syscall (21)
// Format: pipe(fd) - creates a pipe pair
// Writes [rfd, wfd, 0] to fd
func (k *Kernel) handlePipe(mu unicorn.Unicorn, rsp uint64) {
	fdPtr, _ := readArg(mu, rsp, 0)

	// Create pipe
	rpipe, wpipe := NewPipe()

	// Allocate file descriptors
	rfd := k.allocVFile(&draw.VirtualFile{
		Name:   "pipe-read",
		Reader: rpipe,
	})
	wfd := k.allocVFile(&draw.VirtualFile{
		Name:   "pipe-write",
		Writer: wpipe,
	})

	// Create result array: [rfd, wfd, 0]
	result := make([]byte, 24) // 3 * uint64
	mu.MemWrite(fdPtr, result)

	// Write rfd
	rfdBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		if i < 4 {
			rfdBytes[i] = byte(rfd >> (uint(i) * 8))
		}
	}
	mu.MemWrite(fdPtr, rfdBytes)

	// Write wfd
	wfdBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		if i < 4 {
			wfdBytes[i] = byte(wfd >> (uint(i) * 8))
		}
	}
	mu.MemWrite(fdPtr+8, wfdBytes)

	// Write 0
	mu.MemWrite(fdPtr+16, make([]byte, 8))

	setReturn(mu, 0)
}
