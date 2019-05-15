package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// Empty checks if a string referenced by s or s itself is empty.
func Empty(s *string) bool {
	return s == nil || *s == ""
}

// UUID will generate a random v14 unique identifier based upon random nunbers
func UUID() string {
	version := byte(4)
	uuid := make([]byte, 16)
	rand.Read(uuid)

	// Set version
	uuid[6] = (uuid[6] & 0x0f) | (version << 4)

	// Set variant
	uuid[8] = (uuid[8] & 0xbf) | 0x80

	buf := make([]byte, 36)
	var dash byte = '-'
	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = dash
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = dash
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = dash
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = dash
	hex.Encode(buf[24:], uuid[10:])

	return string(buf)
}
