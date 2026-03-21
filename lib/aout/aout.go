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