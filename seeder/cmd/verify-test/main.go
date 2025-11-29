package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Data from test files
	infohashHex := "e716530f4dbaf4ced3dac767633566ab0649ae32303db5a0f98058e44030d94a"
	pubkeyB64 := "CMMjjaZie5rvd8zAVFuGtZR/p44tv1XcD288EgSweKU="
	signatureB64 := "Q+7oc++fklcZ8Q027rP+kcw87ssraPFVGxdO7msXLJTXxnsDMnw18jQVW904ueBPES2vwtwJdVNtSad/awAMAw=="

	// Decode public key
	pubkeyBytes, err := base64.StdEncoding.DecodeString(pubkeyB64)
	if err != nil {
		fmt.Printf("Failed to decode pubkey: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Pubkey length: %d bytes\n", len(pubkeyBytes))
	fmt.Printf("Pubkey hex: %x\n", pubkeyBytes)

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		fmt.Printf("Failed to decode signature: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Signature length: %d bytes\n", len(sigBytes))
	fmt.Printf("Signature hex: %x\n", sigBytes)

	// Decode infohash
	infohashBytes, err := hex.DecodeString(infohashHex)
	if err != nil {
		fmt.Printf("Failed to decode infohash: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Infohash length: %d bytes\n", len(infohashBytes))

	// Verify with full infohash string (with prefix)
	infohashFull := "sha256:" + infohashHex
	fmt.Printf("\n--- Test 1: Verify with full string (sha256:...) ---\n")
	valid := ed25519.Verify(pubkeyBytes, []byte(infohashFull), sigBytes)
	fmt.Printf("Signature valid: %v\n", valid)

	// Verify with just hex string
	fmt.Printf("\n--- Test 2: Verify with hex string only ---\n")
	valid = ed25519.Verify(pubkeyBytes, []byte(infohashHex), sigBytes)
	fmt.Printf("Signature valid: %v\n", valid)

	// Verify with raw hash bytes
	fmt.Printf("\n--- Test 3: Verify with raw hash bytes ---\n")
	valid = ed25519.Verify(pubkeyBytes, infohashBytes, sigBytes)
	fmt.Printf("Signature valid: %v\n", valid)

	// Test with lowercase hex
	fmt.Printf("\n--- Test 4: Verify with lowercase hex ---\n")
	valid = ed25519.Verify(pubkeyBytes, []byte(strings.ToLower(infohashHex)), sigBytes)
	fmt.Printf("Signature valid: %v\n", valid)

	// Test with uppercase hex
	fmt.Printf("\n--- Test 5: Verify with uppercase hex ---\n")
	valid = ed25519.Verify(pubkeyBytes, []byte(strings.ToUpper(infohashHex)), sigBytes)
	fmt.Printf("Signature valid: %v\n", valid)
}
