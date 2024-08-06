package bee

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"golang.org/x/crypto/sha3"
)

// File represents Bee file
type File struct {
	address        swarm.Address
	name           string
	hash           []byte
	dataReader     io.Reader
	size           int64
	historyAddress swarm.Address
}

// NewRandomFile returns new pseudorandom file
func NewRandomFile(r *rand.Rand, name string, size int64) File {
	return File{
		name:       name,
		dataReader: io.LimitReader(r, size),
		size:       size,
	}
}

// NewBufferFile returns new file with specified buffer
func NewBufferFile(name string, buffer *bytes.Buffer) File {
	return File{
		name:       name,
		dataReader: buffer,
		size:       int64(buffer.Len()),
	}
}

// CalculateHash calculates hash from dataReader.
// It replaces dataReader with another that will contain the data.
func (f *File) CalculateHash() error {
	h := fileHasher()

	var buf bytes.Buffer
	tee := io.TeeReader(f.DataReader(), &buf)

	_, err := io.Copy(h, tee)
	if err != nil {
		return err
	}

	f.hash = h.Sum(nil)
	f.dataReader = &buf

	return nil
}

// Address returns file's address
func (f *File) Address() swarm.Address {
	return f.address
}

func (f *File) HistroryAddress() swarm.Address {
	return f.historyAddress
}

// Name returns file's name
func (f *File) Name() string {
	return f.name
}

// Hash returns file's hash
func (f *File) Hash() []byte {
	return f.hash
}

// DataReader returns file's data reader
func (f *File) DataReader() io.Reader {
	return f.dataReader
}

// Size returns file size
func (f *File) Size() int64 {
	return f.size
}

// ClosestNode returns file's closest node of a given set of nodes
func (f *File) ClosestNode(nodes []swarm.Address) (closest swarm.Address, err error) {
	closest = nodes[0]
	for _, a := range nodes[1:] {
		dcmp, err := swarm.DistanceCmp(f.Address(), closest, a)
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

func (f *File) SetAddress(a swarm.Address) {
	f.address = a
}

func (f *File) SetHistroryAddress(a swarm.Address) {
	f.historyAddress = a
}

func (f *File) SetHash(h []byte) {
	f.hash = h
}

func fileHasher() hash.Hash {
	return sha3.New256()
}
