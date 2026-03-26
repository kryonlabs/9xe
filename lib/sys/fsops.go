package sys

import (
	"os"
	"path/filepath"

	"github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// handleChdir implements CHDIR syscall (3)
func (k *Kernel) handleChdir(mu unicorn.Unicorn, rsp uint64) {
	pathPtr, _ := readArg(mu, rsp, 0)

	path, err := readString(mu, pathPtr, 1024)
	if err != nil {
		k.setError(mu, "bad path ptr")
		return
	}

	// Resolve path relative to current directory
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(k.cwd, path)
	}

	// Check if directory exists
	info, err := os.Stat(fullPath)
	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	if !info.IsDir() {
		k.setError(mu, "not a directory")
		return
	}

	// Update current directory
	k.cwd = fullPath
	setReturn(mu, 0)
}

// handleCreate implements CREATE syscall (22)
func (k *Kernel) handleCreate(mu unicorn.Unicorn, rsp uint64) {
	pathPtr, _ := readArg(mu, rsp, 0)
	mode, _ := readArg(mu, rsp, 1)
	perm, _ := readArg(mu, rsp, 2)

	path, err := readString(mu, pathPtr, 1024)
	if err != nil {
		k.setError(mu, "bad path ptr")
		return
	}

	// Resolve path relative to current directory
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(k.cwd, path)
	}

	// Create file with appropriate flags
	var flag int
	switch mode & 3 {
	case OREAD:
		flag = os.O_RDONLY | os.O_CREATE
	case OWRITE:
		flag = os.O_WRONLY | os.O_CREATE
	case ORDWR:
		flag = os.O_RDWR | os.O_CREATE
	default:
		flag = os.O_RDWR | os.O_CREATE
	}

	if mode&OTRUNC != 0 {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(fullPath, flag, os.FileMode(perm))
	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	setReturn(mu, uint64(k.allocFd(f)))
}

// handleRemove implements REMOVE syscall (25)
func (k *Kernel) handleRemove(mu unicorn.Unicorn, rsp uint64) {
	pathPtr, _ := readArg(mu, rsp, 0)

	path, err := readString(mu, pathPtr, 1024)
	if err != nil {
		k.setError(mu, "bad path ptr")
		return
	}

	// Resolve path relative to current directory
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(k.cwd, path)
	}

	// Remove file or directory
	err = os.RemoveAll(fullPath)
	if err != nil {
		k.setError(mu, err.Error())
		return
	}

	setReturn(mu, 0)
}
