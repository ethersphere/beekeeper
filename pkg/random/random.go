package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"math/rand"
)

// PseudoGenerator returns *rand.Rand
func PseudoGenerator(seed int64) (g *rand.Rand) {
	rnd := rand.New(rand.NewSource(seed))
	return rand.New(rand.NewSource(rnd.Int63()))
}

// PseudoGenerators returns list of n *rand.Rand.
// This is needed in cases where random number generators are used in different
// goroutines, so that predictability of the generators can be maintained.
func PseudoGenerators(seed int64, n int) (g []*rand.Rand) {
	rnd := rand.New(rand.NewSource(seed))
	for range n {
		g = append(g, rand.New(rand.NewSource(rnd.Int63())))
	}
	return
}

// Int64 returns random int64
func Int64() int64 {
	var src CryptoSource
	rnd := rand.New(src)
	return rnd.Int63()
}

// CryptoSource is used to create random source
type CryptoSource struct{}

func (s CryptoSource) Seed(_ int64) {}

func (s CryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s CryptoSource) Uint64() (v uint64) {
	err := binary.Read(crand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
