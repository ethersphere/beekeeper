package check

import (
	"fmt"
	"math/rand"
)

// PushSyncOptions ...
type PushSyncOptions struct {
	NodeCount   int
	Namespace   string
	Seed        int64
	RandomSeed  bool
	URLTemplate string
}

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

	data := make([]byte, chunkSize)
	if _, err := rand.Read(data); err != nil {
		return err
	}

	return
}
