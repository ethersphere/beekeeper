package bee

import (
	"math"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

const MinimumBatchDepth = 2

func EstimatePostageBatchDepth(contentLength int64) uint64 {
	depth := uint64(math.Log2(float64(calculateNumberOfChunks(contentLength, false))))
	if depth < MinimumBatchDepth {
		depth = MinimumBatchDepth
	}
	return depth
}

// calculateNumberOfChunks calculates the number of chunks in an arbitrary
// content length.
func calculateNumberOfChunks(contentLength int64, isEncrypted bool) int64 {
	if contentLength <= swarm.ChunkSize {
		return 1
	}
	branchingFactor := swarm.Branches
	if isEncrypted {
		branchingFactor = swarm.EncryptedBranches
	}

	dataChunks := math.Ceil(float64(contentLength) / float64(swarm.ChunkSize))
	totalChunks := dataChunks
	intermediate := dataChunks / float64(branchingFactor)

	for intermediate > 1 {
		totalChunks += math.Ceil(intermediate)
		intermediate = intermediate / float64(branchingFactor)
	}

	return int64(totalChunks) + 1
}
