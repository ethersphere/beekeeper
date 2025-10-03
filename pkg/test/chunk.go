package test

import (
	"bytes"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

type chunkV2 struct {
	addr swarm.Address
	data []byte
}

func (c *chunkV2) Addr() swarm.Address {
	return c.addr
}

func (c *chunkV2) AddrString() string {
	return c.addr.String()
}

func (c *chunkV2) Equals(data []byte) bool {
	return bytes.Equal(c.data, data)
}

func (c *chunkV2) Contains(data []byte) bool {
	return bytes.Contains(c.data, data)
}

func (c *chunkV2) Size() int {
	return len(c.data)
}
