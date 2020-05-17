package bee

import (
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
)

const maxChunkSize = 4096

// Chunk represents Bee chunk
type Chunk struct {
	Address swarm.Address
	data    []byte
}

// NewChunk creates new Chunk
func NewChunk() Chunk {
	return Chunk{data: []byte{}}
}

// NewRandomChunk creates pseudo random chunk
func NewRandomChunk(seed int64) (c Chunk, err error) {
	src := rand.NewSource(seed)
	r := rand.New(src)

	c = Chunk{data: make([]byte, r.Intn(maxChunkSize))}
	if _, err := r.Read(c.data); err != nil {
		return NewChunk(), err
	}

	return
}

// NewRandomChunks creates N pseudo random chunks
func NewRandomChunks(seed int64, n int) (chunks []Chunk, err error) {
	src := rand.NewSource(seed)
	r := rand.New(src)

	for i := 0; i < n; i++ {
		c := Chunk{data: make([]byte, r.Intn(maxChunkSize))}
		if _, err := r.Read(c.data); err != nil {
			return []Chunk{}, err
		}
		chunks = append(chunks, c)
	}

	return
}

// Data returns chunk's data
func (c *Chunk) Data() []byte {
	return c.data
}

// Size returns chunk size
func (c *Chunk) Size() int {
	return len(c.data)
}

// ClosestNode returns chunk's closest node
func (c *Chunk) ClosestNode(nodes []swarm.Address) (closest swarm.Address, err error) {
	closest = nodes[0]
	for _, a := range nodes[1:] {
		dcmp, err := swarm.DistanceCmp(c.Address.Bytes(), closest.Bytes(), a.Bytes())
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
