package bee

import (
	"fmt"
	"math/rand"
	"os"
)

const (
	charSet     = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxFileSize = 104857600 // 100MB
)

// NewRandomFile returns new pseudorandom file (size in bytes)
func NewRandomFile(r *rand.Rand, filename string, size int) (f *os.File, err error) {
	if size > maxFileSize {
		return nil, fmt.Errorf("create random file: requested size too big (max %d bytes)", maxFileSize)
	}

	f, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("create random file: %w", err)
	}

	b := make([]byte, size)
	for i := range b {
		b[i] = charSet[r.Intn(len(charSet))]
	}

	n, err := f.WriteString(string(b))
	if err != nil {
		return nil, fmt.Errorf("create random file: %w", err)
	}
	if n != size {
		return nil, fmt.Errorf("create random file: created file of size %d, requested %d", n, size)
	}

	return
}
