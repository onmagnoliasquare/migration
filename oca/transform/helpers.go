package main

import (
	"crypto/rand"
	"fmt"
)

// whatWgRND creates a byte array of values that follow the WHATWG crypto RNG standard.
// See: WHATWG crypto RNG - https://w3c.github.io/webcrypto/Overview.html
func whatWgRNG(l uint8) ([]byte, error) {
	b := make([]byte, l)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// generateByteToHex generates a 256 array of bytes in hexadecimal format.
func generateByteToHex() []string {
	byteToHex := make([]string, 256)
	for i := 0; i < len(byteToHex); i++ {
		byteToHex[i] = fmt.Sprintf("%02x", i+0x100)
	}
	return byteToHex
}

// randomKey generates a random key given a length.
func randomKey(l uint8) (string, error) {

	byteToHex := generateByteToHex()

	rnds, err := whatWgRNG(l)
	if err != nil {
		return "", err
	}

	result := ""

	for _, b := range rnds {
		result += byteToHex[b]
	}

	return result[:l], nil
}
