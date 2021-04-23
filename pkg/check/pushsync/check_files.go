package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckFiles uploads given files on cluster and verifies expected tag state
func CheckFiles(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			rnd := rnds[i]
			fileSize := o.FileSize + int64(j)
			file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d-%d", "file", i, j), fileSize)

			client := ng.NodeClient(nodeName)

			tagResponse, err := client.CreateTag(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			tagUID := tagResponse.Uid

			// add some buffer to ensure depth is enough
			depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
			batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, depth, "test-label")
			if err != nil {
				return fmt.Errorf("node %s: created batched id %w", nodeName, err)
			}

			fmt.Printf("node %s: created batched id %s\n", nodeName, batchID)

			time.Sleep(o.PostageWait)

			if err := client.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID, Tag: tagUID}); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[nodeName].String())

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retires: %w", errPushSync)
				}

				time.Sleep(o.RetryDelay)

				afterUploadTagResponse, err := ng.NodeClient(nodeName).GetTag(ctx, tagUID)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}

				tagSplitCount := afterUploadTagResponse.Split
				tagSentCount := afterUploadTagResponse.Sent
				tagSeenCount := afterUploadTagResponse.Seen

				diff := tagSplitCount - (tagSentCount + tagSeenCount)

				if diff != 0 {
					fmt.Printf("File %s tag counters do not match (diff: %d): %+v\n", file.Address().String(), diff, afterUploadTagResponse)
					continue
				}

				fmt.Printf("File %s tag counters: %+v\n", file.Address().String(), afterUploadTagResponse)

				// check succeeded
				break
			}

		}
	}

	return
}
