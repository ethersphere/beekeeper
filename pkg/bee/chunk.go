package bee

import (
	"encoding/binary"
	"fmt"
	"hash"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	bmtlegacy "github.com/ethersphere/bmt/legacy"
	"golang.org/x/crypto/sha3"
)

const (
	// MaxChunkSize represents max chunk size in bytes
	MaxChunkSize = 4096
	spanInfoSize = 8
)

// Chunk represents Bee chunk
type Chunk struct {
	address swarm.Address
	data    []byte
	span    int
}

// NewChunk returns new chunk
func NewChunk(address swarm.Address, data []byte) (*Chunk, error) {
	if len(data) > MaxChunkSize+spanInfoSize {
		return nil, fmt.Errorf("create chunk: requested size too big (max %d bytes)", MaxChunkSize+spanInfoSize)
	}

	return &Chunk{address: address, data: data}, nil
}

// NewRandomChunk returns new pseudorandom chunk
func NewRandomChunk(r *rand.Rand) (Chunk, error) {
	data := make([]byte, r.Intn(MaxChunkSize))
	if _, err := r.Read(data); err != nil {
		return Chunk{}, fmt.Errorf("create random chunk: %w", err)
	}

	span := len(data)
	b := make([]byte, spanInfoSize)
	binary.LittleEndian.PutUint64(b, uint64(span))
	data = append(b, data...)

	c := Chunk{data: data, span: span}
	err := c.SetAddress()
	return c, err
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

// Span returns chunk span
func (c *Chunk) Span() int {
	return c.span
}

// SetAddress calculates the address of a chunk and assign's it to address field
func (c *Chunk) SetAddress() error {
	p := bmtlegacy.NewTreePool(chunkHahser, swarm.Branches, bmtlegacy.PoolSize)
	hasher := bmtlegacy.New(p)
	err := hasher.SetSpanBytes(c.Data()[:8])
	if err != nil {
		return err
	}
	_, err = hasher.Write(c.Data()[8:])
	if err != nil {
		return err
	}
	c.address = swarm.NewAddress(hasher.Sum(nil))
	return nil
}

// ClosestNode returns chunk's closest node of a given set of nodes
func (c *Chunk) ClosestNode(nodes []swarm.Address) (closest swarm.Address, err error) {
	closest = nodes[0]
	for _, a := range nodes[1:] {
		dcmp, err := swarm.DistanceCmp(c.Address().Bytes(), closest.Bytes(), a.Bytes())
		if err != nil {
			return swarm.Address{}, fmt.Errorf("find closest node: %w", err)
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

// ClosestNodeFromMap returns chunk's closest node of a given map of nodes
func (c *Chunk) ClosestNodeFromMap(nodes map[string]swarm.Address) (closestName string, closestAddress swarm.Address, err error) {
	for k, v := range nodes {
		fmt.Println(k, v)
	}
	names := make([]string, 0, len(nodes))
	addresses := make([]swarm.Address, 0, len(nodes))
	for k, v := range nodes {
		names = append(names, k)
		addresses = append(addresses, v)
	}
	closestName = names[0]
	closestAddress = addresses[0]

	for i, a := range addresses[1:] {
		dcmp, err := swarm.DistanceCmp(c.Address().Bytes(), closestAddress.Bytes(), a.Bytes())
		if err != nil {
			return "", swarm.Address{}, fmt.Errorf("find closest node: %w", err)
		}
		switch dcmp {
		case 0:
			// do nothing
		case -1:
			// current node is closer
			closestName = names[i+1]
			closestAddress = a
		case 1:
			// closest is already closer to chunk
			// do nothing
		}
	}

	return
}

func chunkHahser() hash.Hash {
	return sha3.NewLegacyKeccak256()
}
