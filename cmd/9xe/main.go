package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kryonlabs/9xe/lib/aout"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: 9xe <path_to_plan9_binary>"
		return
	}

	target := os.Args[1]
	f, err := os.Open(target
	if err != nil {
		log.Fatalf("Error opening binary: %v", err)
	}
	defer f.Close()

	hdr, err := aout.ReadHeader(f)
	if err != nil {
		log.Fatalf("Error parsing Plan9 header: %v", err)
	}

	fmt.Printf("--- 9xe Executive: Binary Analysis ---\n")
	fmt.Printf("File:    %s\n", target)
	fmt.Printf("Magic:   0x%x\n", hdr.Magic)
	fmt.Printf("Entry:   0x%x\n", hdr.Entry)
	fmt.Printf("Code:    %d bytes\n", hdr.Text)
	fmt.Printf("Data:    %d bytes\n", hdr.Data)
	fmt.Printf("---------------------------------------\n")
}
