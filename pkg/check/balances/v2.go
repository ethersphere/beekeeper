package balances

import (
	"context"
	"fmt"

	beev2 "github.com/ethersphere/beekeeper/pkg/check/bee"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

func (c *Check) RunV2(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		fmt.Println("running balances (dry mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running balances")

	var checkCase *beev2.CheckCase

	caseOpts := beev2.CaseOptions{
		FileName:      o.FileName,
		FileSize:      o.FileSize,
		GasPrice:      o.GasPrice,
		PostageAmount: o.PostageAmount,
		PostageLabel:  o.PostageLabel,
		PostageWait:   o.PostageWait,
		Seed:          o.Seed,
	}

	if checkCase, err = beev2.NewCheckCase(ctx, cluster, caseOpts); err != nil {
		return err
	}

	if err := checkCase.SaveBalances(); err != nil {
		return err
	}

	if err := checkCase.ExpectValidInitialBalances(ctx); err != nil {
		return err
	}

	// repeats
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload/check
		node := checkCase.RandomNode()

		file, err := node.UploadRandomFile(ctx)

		if err != nil {
			return err
		}
		if err := checkCase.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}

		// download/check
		if err := checkCase.RandomNode().ExpectToHaveFile(ctx, file); err != nil {
			return err
		}
		if err := checkCase.SaveBalances(); err != nil {
			return err
		}
		if err := checkCase.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}
	}

	return nil
}
