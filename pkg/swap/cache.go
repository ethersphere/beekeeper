package swap

import (
	"sync"
)

type cache struct {
	blockTime int64
	m         sync.Mutex
}

func newCache() *cache {
	return &cache{}
}

func (c *cache) SetBlockTime(blockTime int64) {
	c.m.Lock()
	c.blockTime = blockTime
	c.m.Unlock()
}

func (c *cache) BlockTime() int64 {
	c.m.Lock()
	defer c.m.Unlock()
	return c.blockTime
}
