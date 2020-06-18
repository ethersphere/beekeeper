package bee

import (
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
)

const (
	charSet     = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxFileSize = 104857600 // 100MB
)

// File represents Bee file
type File struct {
	address swarm.Address
	name    string
	data    []byte
	size    int
}

// NewRandomFile returns new pseudorandom file
func NewRandomFile(r *rand.Rand, name string, size int) (f File, err error) {
	if size > maxFileSize {
		return File{}, fmt.Errorf("create random file: requested size too big (max %d bytes)", maxFileSize)
	}

	data := make([]byte, size)
	if _, err := r.Read(data); err != nil {
		return File{}, fmt.Errorf("create random file: %w", err)
	}

	f = File{name: name, data: data, size: size}

	return
}

// Address returns file's address
func (f *File) Address() swarm.Address {
	return f.address
}

// Name returns file's name
func (f *File) Name() string {
	return f.name
}

// Data returns file's data
func (f *File) Data() []byte {
	return f.data
}

// Size returns file size
func (f *File) Size() int {
	return f.size
}

// ClosestNode returns file's closest node of a given set of nodes
func (f *File) ClosestNode(nodes []swarm.Address) (closest swarm.Address, err error) {
	closest = nodes[0]
	for _, a := range nodes[1:] {
		dcmp, err := swarm.DistanceCmp(f.Address().Bytes(), closest.Bytes(), a.Bytes())
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
