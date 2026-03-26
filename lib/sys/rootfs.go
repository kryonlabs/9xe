package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// RootFS provides a Plan 9 root filesystem that maps Plan 9 paths to host filesystem paths.
// It manages environment variables and provides path translation for Plan 9 binaries.
type RootFS struct {
	root     string        // Host filesystem root path
	cwd      string        // Current working directory (Plan 9 path)
	env      map[string]string // Environment variables
	envMutex sync.RWMutex
}

// NewRootFS creates a new root filesystem.
// rootPath is the host directory that will serve as the Plan 9 root ("/").
// If empty, uses the current directory.
func NewRootFS(rootPath string) (*RootFS, error) {
	// Use current directory if rootPath is empty
	if rootPath == "" {
		rootPath = "."
	}

	// Get absolute path
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", rootPath, err)
	}

	// Verify root exists
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("root path %q does not exist: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path %q is not a directory", absRoot)
	}

	rfs := &RootFS{
		root: absRoot,
		cwd:  "/",
		env:  make(map[string]string),
	}

	// Initialize with standard Plan 9 environment variables
	rfs.initDefaultEnv()

	return rfs, nil
}

// initDefaultEnv initializes default Plan 9 environment variables
func (rfs *RootFS) initDefaultEnv() {
	rfs.env["PATH"] = "/bin"
	rfs.env["HOME"] = "/usr/user"
	rfs.env["USER"] = "glenda"
	rfs.env["cputype"] = "amd64"
	rfs.env["objtype"] = "amd64"
	rfs.env["sysname"] = "9xe"
	rfs.env["service"] = "terminal"
	rfs.env["rootdir"] = "/"
	rfs.env["obj"] = "/usr/$user/tmp"
}

// LocalPath converts a Plan 9 path to a host filesystem path.
// Absolute Plan 9 paths (starting with /) are resolved relative to the root.
// Relative paths are resolved relative to the current working directory.
func (rfs *RootFS) LocalPath(plan9Path string) string {
	if plan9Path == "" || plan9Path == "/" {
		return rfs.root
	}

	// Handle relative paths
	if !strings.HasPrefix(plan9Path, "/") {
		// Resolve relative to current working directory
		absPath := filepath.Join(rfs.cwd, plan9Path)
		plan9Path = absPath
	}

	// Remove leading slash and join with root
	plan9Path = strings.TrimPrefix(plan9Path, "/")

	// Join with root (use filepath.Join to handle platform-specific separators)
	return filepath.Join(rfs.root, plan9Path)
}

// Plan9Path converts a host path to a Plan 9 path (relative to root)
func (rfs *RootFS) Plan9Path(hostPath string) string {
	// Get absolute host path
	absHost, err := filepath.Abs(hostPath)
	if err != nil {
		return hostPath
	}

	// Get absolute root path
	absRoot, err := filepath.Abs(rfs.root)
	if err != nil {
		return hostPath
	}

	// If the path is under root, convert to Plan 9 path
	if strings.HasPrefix(absHost, absRoot) {
		relPath := strings.TrimPrefix(absHost, absRoot)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			return "/"
		}
		return "/" + relPath
	}

	// Path is outside root, return as-is
	return hostPath
}

// GetCwd returns the current working directory as a Plan 9 path
func (rfs *RootFS) GetCwd() string {
	return rfs.cwd
}

// SetCwd sets the current working directory (Plan 9 path)
func (rfs *RootFS) SetCwd(plan9Path string) error {
	// Normalize path
	if !strings.HasPrefix(plan9Path, "/") {
		plan9Path = filepath.Join(rfs.cwd, plan9Path)
		plan9Path = filepath.Clean(plan9Path)
	}

	// Verify path exists under root
	hostPath := rfs.LocalPath(plan9Path)
	info, err := os.Stat(hostPath)
	if err != nil {
		return fmt.Errorf("directory %q does not exist: %w", plan9Path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", plan9Path)
	}

	rfs.cwd = plan9Path
	return nil
}

// GetEnv gets an environment variable value
func (rfs *RootFS) GetEnv(name string) string {
	rfs.envMutex.RLock()
	defer rfs.envMutex.RUnlock()
	return rfs.env[name]
}

// SetEnv sets an environment variable
func (rfs *RootFS) SetEnv(name, value string) {
	rfs.envMutex.Lock()
	defer rfs.envMutex.Unlock()
	rfs.env[name] = value
}

// DelEnv deletes an environment variable
func (rfs *RootFS) DelEnv(name string) {
	rfs.envMutex.Lock()
	defer rfs.envMutex.Unlock()
	delete(rfs.env, name)
}

// GetAllEnv returns a copy of all environment variables
func (rfs *RootFS) GetAllEnv() map[string]string {
	rfs.envMutex.RLock()
	defer rfs.envMutex.RUnlock()

	result := make(map[string]string, len(rfs.env))
	for k, v := range rfs.env {
		result[k] = v
	}
	return result
}

// GetRoot returns the root filesystem path
func (rfs *RootFS) GetRoot() string {
	return rfs.root
}

// Exists checks if a Plan 9 path exists in the root filesystem
func (rfs *RootFS) Exists(plan9Path string) bool {
	hostPath := rfs.LocalPath(plan9Path)
	_, err := os.Stat(hostPath)
	return err == nil
}

// IsDir checks if a Plan 9 path is a directory
func (rfs *RootFS) IsDir(plan9Path string) bool {
	hostPath := rfs.LocalPath(plan9Path)
	info, err := os.Stat(hostPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Mkdir creates a directory at the given Plan 9 path
func (rfs *RootFS) Mkdir(plan9Path string, perm os.FileMode) error {
	hostPath := rfs.LocalPath(plan9Path)
	return os.MkdirAll(hostPath, perm)
}
