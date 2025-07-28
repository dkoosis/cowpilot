// Package spektrix provides custom HMAC-SHA1 implementation for Spektrix API
// This is ported from the JavaScript implementation in the sandy project
//
// CRITICAL: Native Go crypto/hmac produces different signatures than Spektrix expects
// This custom implementation matches the sandy project's proven working version
package spektrix

import (
	"encoding/hex"
	"fmt"
)

// hmacSHA1 generates HMAC-SHA1 signature using custom implementation
// This matches the JavaScript HMACJS.sha1() function from sandy project
func hmacSHA1(message, key string) ([]byte, error) {
	keyBytes := []byte(key)

	// If key is longer than 64 bytes, hash it
	if len(keyBytes) > 64 {
		hashedKey, err := sha1Hash(key)
		if err != nil {
			return nil, err
		}
		keyBytes = hashedKey
	}

	// Prepare ipad and opad
	ipad := make([]byte, 64)
	opad := make([]byte, 64)

	for i := 0; i < 64; i++ {
		var keyByte byte
		if i < len(keyBytes) {
			keyByte = keyBytes[i]
		}
		ipad[i] = keyByte ^ 0x36
		opad[i] = keyByte ^ 0x5C
	}

	// Inner hash: SHA1(ipad + message)
	innerInput := string(ipad) + message
	innerHash, err := sha1Hash(innerInput)
	if err != nil {
		return nil, err
	}

	// Outer hash: SHA1(opad + innerHash)
	outerInput := string(opad) + string(innerHash)
	outerHash, err := sha1Hash(outerInput)
	if err != nil {
		return nil, err
	}

	return outerHash, nil
}

// sha1Hash implements SHA1 algorithm matching the JavaScript version
func sha1Hash(input string) ([]byte, error) {
	message := []byte(input)

	// Convert message to 32-bit words
	words := bytesToWords(message)
	msgLen := len(input) * 8

	// Pre-processing: padding
	words = append(words, 0x80000000)
	for len(words)%16 != 14 {
		words = append(words, 0)
	}
	words = append(words, uint32(msgLen>>32), uint32(msgLen&0xFFFFFFFF))

	// Initialize hash values
	h0 := uint32(0x67452301)
	h1 := uint32(0xEFCDAB89)
	h2 := uint32(0x98BADCFE)
	h3 := uint32(0x10325476)
	h4 := uint32(0xC3D2E1F0)

	// Process message in 512-bit chunks
	for i := 0; i < len(words); i += 16 {
		w := make([]uint32, 80)

		// Copy chunk into first 16 words
		copy(w, words[i:i+16])

		// Extend the sixteen 32-bit words into eighty 32-bit words
		for j := 16; j < 80; j++ {
			temp := w[j-3] ^ w[j-8] ^ w[j-14] ^ w[j-16]
			w[j] = leftRotate(temp, 1)
		}

		// Initialize hash value for this chunk
		a, b, c, d, e := h0, h1, h2, h3, h4

		// Main loop
		for j := 0; j < 80; j++ {
			var f, k uint32

			if j < 20 {
				f = (b & c) | ((^b) & d)
				k = 0x5A827999
			} else if j < 40 {
				f = b ^ c ^ d
				k = 0x6ED9EBA1
			} else if j < 60 {
				f = (b & c) | (b & d) | (c & d)
				k = 0x8F1BBCDC
			} else {
				f = b ^ c ^ d
				k = 0xCA62C1D6
			}

			temp := leftRotate(a, 5) + f + e + k + w[j]
			e = d
			d = c
			c = leftRotate(b, 30)
			b = a
			a = temp
		}

		// Add this chunk's hash to result
		h0 += a
		h1 += b
		h2 += c
		h3 += d
		h4 += e
	}

	// Convert hash values to bytes
	result := make([]byte, 20)
	writeUint32(result[0:4], h0)
	writeUint32(result[4:8], h1)
	writeUint32(result[8:12], h2)
	writeUint32(result[12:16], h3)
	writeUint32(result[16:20], h4)

	return result, nil
}

// bytesToWords converts byte slice to uint32 slice (big-endian)
func bytesToWords(bytes []byte) []uint32 {
	words := make([]uint32, (len(bytes)+3)/4)
	for i := 0; i < len(bytes); i++ {
		words[i/4] |= uint32(bytes[i]) << (24 - (i%4)*8)
	}
	return words
}

// leftRotate performs left rotation of 32-bit integer
func leftRotate(value uint32, amount uint) uint32 {
	return (value << amount) | (value >> (32 - amount))
}

// writeUint32 writes uint32 to byte slice in big-endian format
func writeUint32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// hexToBytes converts hex string to byte slice
func hexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}
