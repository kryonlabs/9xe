package aout

import (
    "encoding/binary"
    "fmt"
    "io"
)

// The _MAGIC macro translated perfectly to Go
func CalculateMagic(f uint32, b uint32) uint32 {
    return f | ((((4 * b) + 0) * b ) + 7)
}

// FIXED: Removed the extra zero. 
// This perfectly matches #define HDR_MAGIC 0x00008000 in a.out.h
const HDR_MAGIC = 0x00008000

type Header struct {
    Magic uint32 // The calculated _MAGIC
    Text  uint32 // Size of code
    Data  uint32 // Size of initialized data
    Bss   uint32 // Size of uninitialized data
    Syms  uint32 // Size of symbol table
    Entry uint32 // Entry point (32-bit long in your header)
    Spsz  uint32 // Size of PC/SP offset table
    Pcsz  uint32 // Size of PC/line number table
}

func (h *Header) GetArchitecture() string {
    switch h.Magic {
    case CalculateMagic(0, 11):         return "i386 (I_MAGIC)"
    case CalculateMagic(0, 20):         return "ARM (E_MAGIC)"
    case CalculateMagic(HDR_MAGIC, 26): return "AMD64 (S_MAGIC)"
    case CalculateMagic(HDR_MAGIC, 28): return "ARM64 (R_MAGIC)"
    case CalculateMagic(0, 8):          return "68020 (A_MAGIC)"
    default:                            return fmt.Sprintf("Unknown/Custom (0x%x)", h.Magic)
    }
}

func ReadHeader(r io.Reader) (*Header, error) {
    var hdr Header
    // Standard Plan 9 Header is 32 bytes (8 * 4-byte longs)
    err := binary.Read(r, binary.BigEndian, &hdr)
    if err != nil {
        return nil, err
    }
    return &hdr, nil
}

// GetEntryOffset returns the offset in the header where the entry point is stored
// For binaries with HDR_MAGIC flag set, entry is at offset 32 (expanded header)
// Otherwise, entry is at offset 24 (standard entry field)
func (h *Header) GetEntryOffset() int {
    if h.Magic&HDR_MAGIC != 0 {
        return 32 // Entry point is in hdr[0] (first expanded header entry)
    }
    return 24 // Entry point is in entry field
}

// ReadEntryAddress reads the actual entry point address from the file
// This handles both standard and expanded headers based on HDR_MAGIC flag
func ReadEntryAddress(r io.ReadSeeker, hdr *Header) (uint64, error) {
    if hdr.Magic&HDR_MAGIC != 0 {
        // Seek to expanded header entry point (offset 32)
        _, err := r.Seek(32, io.SeekStart)
        if err != nil {
            return 0, err
        }
        var entry uint64
        err = binary.Read(r, binary.BigEndian, &entry)
        if err != nil {
            return 0, err
        }
        return entry, nil
    }
    return uint64(hdr.Entry), nil
}

// Symbol represents a Plan 9 symbol table entry
type Symbol struct {
    Value uint64 // Symbol value (address)
    Type  byte   // Symbol type
    Name  string // Symbol name
}

// ReadSymbolTable reads the Plan 9 symbol table
// Plan 9 symbols are variable-length: value (4 or 8 bytes) + type (1 byte) + name (null-terminated)
func ReadSymbolTable(r io.Reader, numBytes uint32) ([]Symbol, error) {
    if numBytes == 0 {
        return nil, nil
    }

    var symbols []Symbol
    bytesRemaining := int(numBytes)

    for bytesRemaining > 0 {
        // Read symbol value (8 bytes for AMD64 S_MAGIC)
        var symValue [8]byte
        n, err := r.Read(symValue[:])
        if err != nil || n != 8 {
            break
        }
        bytesRemaining -= 8

        value := binary.BigEndian.Uint64(symValue[:])

        // Read symbol type (1 byte)
        var symType [1]byte
        n, err = r.Read(symType[:])
        if err != nil || n != 1 {
            break
        }
        bytesRemaining--

        // Read symbol name (null-terminated string)
        var nameBytes []byte
        for {
            var b [1]byte
            n, err := r.Read(b[:])
            if err != nil || n != 1 {
                break
            }
            bytesRemaining--

            if b[0] == 0 {
                break // End of name
            }
            nameBytes = append(nameBytes, b[0])
        }

        symbols = append(symbols, Symbol{
            Value: value,
            Type:  symType[0],
            Name:  string(nameBytes),
        })
    }

    return symbols, nil
}

// FindMainSymbol searches for the main() function in the symbol table
// Plan 9 binaries can have various entry point names: main, <binary_name>, etc.
func FindMainSymbol(symbols []Symbol, binaryName string) uint64 {
    // Try common entry point names
    possibleNames := []string{"main", "_main", "mainp"}

    // Add binary name without path
    if len(binaryName) > 0 {
        // Extract just the filename
       	for i := len(binaryName) - 1; i >= 0; i-- {
       		if binaryName[i] == '/' || binaryName[i] == '\\' {
       			binaryName = binaryName[i+1:]
       			break
       		}
       	}
        possibleNames = append(possibleNames, binaryName)
    }

    // Search for any of the possible names
    for _, name := range possibleNames {
        for _, sym := range symbols {
            if sym.Name == name && (sym.Type == 'T' || sym.Type == 't' || sym.Type == 0xd4 || sym.Type == 0xb4) {
                return sym.Value
            }
        }
    }

    return 0 // Not found
}

// FindTextSymbol searches for any text symbol by name
func FindTextSymbol(symbols []Symbol, name string) uint64 {
    for _, sym := range symbols {
        if sym.Name == name && (sym.Type == 'T' || sym.Type == 't' || sym.Type == 'L' || sym.Type == 'l') {
            return sym.Value
        }
    }
    return 0 // Not found
}