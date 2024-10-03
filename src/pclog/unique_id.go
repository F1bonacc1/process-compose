package pclog

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateUniqueID(length int) string {
	if length%2 != 0 {
		length += 1
	}
	b := make([]byte, length/2) //each byte is 2 chars
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
