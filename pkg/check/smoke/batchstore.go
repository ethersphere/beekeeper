package smoke

import (
	"sync"
	"time"
)

type batch struct {
	batchID string
	expires time.Time
}

type store struct {
	mtx     sync.Mutex
	batches map[string]batch
	maxDur  time.Duration
}

func NewStore(dur time.Duration) *store {

	return &store{
		batches: map[string]batch{},
		maxDur:  dur,
	}
}

func (s *store) Get(key string) string {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if b, ok := s.batches[key]; ok {
		if time.Now().After(b.expires) {
			delete(s.batches, key)
			return ""
		}
		return b.batchID
	}

	return ""
}

func (s *store) Store(key, batchID string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.batches[key] = batch{batchID: batchID, expires: time.Now().Add(s.maxDur)}
}
