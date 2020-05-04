package check

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
)

const maxChunkSize = 4096

// PushSyncOptions ...
type PushSyncOptions struct {
	NodeCount   int
	Namespace   string
	Seed        int64
	RandomSeed  bool
	URLTemplate string
}

var errPushSync = errors.New("push sync")

// PushSync ...
func PushSync(opts PushSyncOptions) (err error) {
	var seed int64
	if opts.RandomSeed {
		var src cryptoSource
		rnd := rand.New(src)
		seed = rnd.Int63()
	} else {
		seed = opts.Seed
	}
	rand.Seed(seed)
	fmt.Println("seed: ", seed)

	chunkSize := rand.Intn(maxChunkSize)
	fmt.Println("chunkSize: ", chunkSize)

	token := make([]byte, chunkSize)
	if _, err := rand.Read(token); err != nil {
		return err
	}

	return
}

type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(crand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
