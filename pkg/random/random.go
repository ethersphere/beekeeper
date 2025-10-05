package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"sync"
)

var ErrUniqueNumberExhausted = errors.New("all unique numbers in the range have been generated")

// Generator is a random number generator that can optionally enforce uniqueness of generated numbers.
type Generator struct {
	rnd    *rand.Rand
	unique bool
	used   map[int]struct{}
	mu     sync.Mutex
}

func NewGenerator(unique bool) *Generator {
	return &Generator{
		rnd:    rand.New(CryptoSource{}),
		unique: unique,
		used:   make(map[int]struct{}),
	}
}

// GetRandom accepts will return positive random int between absolute values of min and max.
// If the generator is set to unique, it will attempt to return a number
// that has not been returned before within the given range.
func (g *Generator) GetRandom(minVal, maxVal int) (int, error) {
	minVal = abs(minVal)
	maxVal = abs(maxVal)
	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	span := maxVal - minVal + 1
	if span <= 0 {
		return 0, errors.New("invalid range: max - min overflows int64")
	}

	if !g.unique {
		return minVal + g.rnd.Intn(span), nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.used) >= span {
		return 0, ErrUniqueNumberExhausted
	}

	for {
		code := minVal + g.rnd.Intn(span)
		if _, found := g.used[code]; !found {
			g.used[code] = struct{}{}
			return code, nil
		}
	}
}

// Reset clears the history of used numbers for a unique generator.
// This allows the generator to be reused for a new sequence of unique numbers.
func (g *Generator) Reset() {
	g.mu.Lock()
	g.used = make(map[int]struct{})
	g.mu.Unlock()
}

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
	return g
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

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
