package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateReference(size int) string {
	buf := make([]byte, size)

	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}

	return hex.EncodeToString(buf)
}
