package balances

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	beev2 "github.com/ethersphere/beekeeper/pkg/check/bee"
)

func (c *Check) RunV2(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		fmt.Println("running balances (dry mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running balances")

	var clusterV2 *beev2.ClusterV2

	clusterOpts := beev2.ClusterOptions{
		DryRun:             o.DryRun,
		FileName:           o.FileName,
		FileSize:           o.FileSize,
		GasPrice:           o.GasPrice,
		PostageAmount:      o.PostageAmount,
		PostageLabel:       o.PostageLabel,
		PostageWait:        o.PostageWait,
		Seed:               o.Seed,
		UploadNodeCount:    o.UploadNodeCount,
		WaitBeforeDownload: o.WaitBeforeDownload,
	}

	if clusterV2, err = beev2.NewClusterV2(ctx, cluster, clusterOpts); err != nil {
		return err
	}

	if err := clusterV2.SaveBalances(); err != nil {
		return err
	}

	if err := clusterV2.ExpectValidInitialBalances(ctx); err != nil {
		return err
	}

	// repeats
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload/check
		node := clusterV2.RandomNode()

		file, err := node.UploadRandomFile(ctx)

		if err != nil {
			return err
		}
		if err := clusterV2.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}

		// download/check
		if err := clusterV2.RandomNode().ExpectToHaveFile(ctx, file); err != nil {
			return err
		}
		if err := clusterV2.SaveBalances(); err != nil {
			return err
		}
		if err := clusterV2.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}
	}

	return nil
}
