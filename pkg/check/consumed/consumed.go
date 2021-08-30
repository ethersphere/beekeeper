package consumed

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
)

// Options represents check options
type Options struct {
	DryRun bool

	Role          string
	AdminUsername string
	AdminPassword string

	RestrictedGroupName string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		DryRun: false,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		fmt.Println("running authenticated 'consumed' check (dry run mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running authenticated 'consumed' check")

	restricted, err := cluster.NodeGroup(o.RestrictedGroupName)
	if err != nil {
		return err
	}

	for _, node := range restricted.Nodes() {
		client := node.Client()

		if _, err := client.Consumed(ctx, "fake-token"); err == nil {
			return errors.New("expected error when making a call while unauthenticated")
		}

		token, err := client.Authenticate(ctx, o.Role, o.AdminUsername, o.AdminPassword)
		if err != nil {
			return fmt.Errorf("authenticate: %w", err)
		}

		balances, err := client.Consumed(ctx, token)
		if err != nil {
			return fmt.Errorf("consumed check: %w", err)
		}

		if len(balances) == 0 {
			return fmt.Errorf("expected non empty balances, got: %v", balances)
		}
	}

	fmt.Println("authenticated 'consumed' check completed successfully")
	return
}

// dryRun does nothing
func dryRun(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	return
}
