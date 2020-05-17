package bee

import (
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
)

const maxChunkSize = 4096

var errChunkSize = fmt.Errorf("chunk size too big (max %d)", maxChunkSize)

// Chunk represents Bee chunk
type Chunk struct {
	address swarm.Address
	data    []byte
}

// NewChunk returns new chunk
func NewChunk(data []byte) (Chunk, error) {
	if len(data) > maxChunkSize {
		return Chunk{}, errChunkSize
	}

	return Chunk{data: data}, nil
}

// NewRandomChunk returns new pseudorandom chunk
func NewRandomChunk(seed int64) (c Chunk, err error) {
	src := rand.NewSource(seed)
	r := rand.New(src)

	c = Chunk{data: make([]byte, r.Intn(maxChunkSize))}
	if _, err := r.Read(c.data); err != nil {
		return Chunk{}, err
	}

	return
}

// NewRandomChunks returns N pseudorandom chunks
func NewRandomChunks(seed int64, n int) (cs []Chunk, err error) {
	src := rand.NewSource(seed)
	r := rand.New(src)

	for i := 0; i < n; i++ {
		c := Chunk{data: make([]byte, r.Intn(maxChunkSize))}
		if _, err := r.Read(c.data); err != nil {
			return []Chunk{}, err
		}
		cs = append(cs, c)
	}

	return
}

// setAddress sets chunk's address
func (c *Chunk) setAddress(a swarm.Address) {
	c.address = a
}

// Address returns chunk's address
func (c *Chunk) Address() swarm.Address {
	return c.address
}

// Data returns chunk's data
func (c *Chunk) Data() []byte {
	return c.data
}

// Size returns chunk size
func (c *Chunk) Size() int {
	return len(c.data)
}

// ClosestNode returns chunk's closest node of a given set of nodes
func (c *Chunk) ClosestNode(nodes []swarm.Address) (closest swarm.Address, err error) {
	closest = nodes[0]
	for _, a := range nodes[1:] {
		dcmp, err := swarm.DistanceCmp(c.Address().Bytes(), closest.Bytes(), a.Bytes())
		if err != nil {
			return swarm.Address{}, err
		}
		switch dcmp {
		case 0:
			// do nothing
		case -1:
			// current node is closer
			closest = a
		case 1:
			// closest is already closer to chunk
			// do nothing
		}
	}

	return
}
