package aout

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Magic386   = 0x00000407 // 80386 (Common for legacy Plan 9)
	MagicAMD64 = 0x00008007 // amd64 (Modern 64-bit)
	MagicArm   = 0x00000647 // ARM (Great for Android/Termux)
)

type Header struct {
	Magic uint32 
	Text  uint32
	Data  uint32
	Bss   uint32
	Syms  uint32
	Entry uint32
	Pcsz  uint32
	Sysz  uint32
}

func ReadHeader(r io.Reader) (*Header, error) {
	var hdr Header

	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return nil, err
	}

	return &hdr, nil
}

