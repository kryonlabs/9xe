# Phase 2 Debug Report - I/O Issues Root Cause

## Date: 2026-03-27

## Critical Discovery

**Root Cause Found**: Cat is calling OPEN syscall with **NULL path pointer (pathPtr = 0)**

### Evidence from Syscall Tracing
```
[SYSCALL] Syscall 14 (OPEN)
[sys] OPEN: NULL path pointer, using empty string
[sys] OPEN: path="" mode=0
[sys] OPEN failed: open : no such file or directory
```

## Analysis

### The Problem
1. Cat binary starts correctly
2. Tries to open a file with **pathPtr = 0** (NULL pointer)
3. OPEN fails with "no such file or directory"
4. Cat likely doesn't handle this error properly and gets stuck

### Why pathPtr is NULL
**Hypothesis**: Cat may be trying to read from **stdin (file descriptor 0)** by default when no file is specified, but the pathPtr is NULL.

In Plan 9, when you run `cat` without arguments, it reads from stdin. The binary might be:
- Checking if arguments were provided
- If no args, trying to open stdin somehow
- The OPEN call with NULL path might be an attempt to access stdin

### Test Results
| Test | Result | Notes |
|------|--------|-------|
| `cat file.txt` | ❌ Hangs | OPEN with NULL path |
| `cat` (no args) | ❌ Hangs | Same issue |
| `echo "test" \| cat` | ❌ Hangs | Still OPEN with NULL path |
| `date` | ✅ Works | No file I/O needed |

## Key Insight

**The hang is NOT an infinite loop** - cat completes execution quickly but doesn't produce output.

Looking at the final state:
```
[Final] RIP=20011c RSP=41fff70
[Final] RAX=0 RBX=0 RCX=4330000000000000 RDX=41fff78
[Summary] Executed 0 instructions, 0 syscalls
```

Wait, this says "Executed 0 instructions, 0 syscalls" but we saw OPEN being called! This suggests the syscall counting is not working properly or there's an issue with how execution is being tracked.

## Next Steps

### Option 1: Fix stdin handling
Cat might expect to read from stdin when given no arguments. We need to:
1. Check if cat is trying to read from fd 0 (stdin)
2. Implement proper stdin reading
3. Handle the case where pathPtr is NULL but mode indicates stdin reading

### Option 2: Check argument passing
The NULL pathPtr might indicate that:
1. Arguments aren't being passed correctly to cat
2. Cat expects a different calling convention
3. There's a mismatch in how we're setting up the process

### Option 3: Test with file that exists
Let's create a test file and see if cat can open it when we force the path.

## Recommended Fix

**Implement proper stdin handling for NULL pathPtr**:

```go
// In handleOpen syscall
if pathPtr == 0 && mode == 0 {
    // NULL path with OREAD mode - likely wants stdin
    // Return fd 0 (stdin)
    fmt.Printf("[sys] OPEN: NULL path with OREAD, returning stdin (fd=0)\n")
    setReturn(mu, 0) // Return stdin fd
    return
}
```

This would allow cat to read from stdin when no file is specified.

## Progress Summary

### ✅ What We've Accomplished
1. **Phase 0**: Test infrastructure ✅
2. **Phase 1**: 7 critical syscalls ✅
3. **Phase 2**: Improved EXEC + argument passing ✅
4. **Debug**: Found root cause (NULL pathPtr) ✅

### 🔧 What's Left
- Fix stdin handling
- Test with actual files
- Get cat working with proper file paths

The foundation is solid - we just need to fix the stdin handling!
