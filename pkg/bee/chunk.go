package bee

import (
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
)

// NewRandomChunk returns new pseudorandom chunk
func NewRandomChunk(r *rand.Rand) (c swarm.Chunk, err error) {
	data := make([]byte, r.Intn(swarm.ChunkSize))
	if _, err := r.Read(data); err != nil {
		return nil, fmt.Errorf("create random chunk: %w", err)
	}

	span := len(data)
	b := make([]byte, swarm.SpanSize)
	binary.LittleEndian.PutUint64(b, uint64(span))

	hasher := GetBmt()
	defer PutBmt(hasher)

	err = hasher.SetSpanBytes(b)
	if err != nil {
		return nil, err
	}
	_, err = hasher.Write(data)
	if err != nil {
		return nil, err
	}
	address := swarm.NewAddress(hasher.Sum(nil))
	data = append(b, data...)

	return swarm.NewChunk(address, data), nil
}

// ClosestNode returns chunk's closest node of a given set of nodes
func ClosestNode(c swarm.Chunk, nodes []swarm.Address) (closest swarm.Address, err error) {
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
