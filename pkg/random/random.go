package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"math/rand"
)

// Int64 returns random int64
func Int64() int64 {
	var src cryptoSource
	rnd := rand.New(src)
	return rnd.Int63()
}

// cryptoSource is used to create random source
type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(crand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
