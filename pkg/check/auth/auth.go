package auth

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
)

// Options represents check options
type Options struct {
	DryRun bool

	Role              string
	AdminUsername     string
	AdminPasswordHash string

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
		fmt.Println("running auth health (dry mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running auth health")

	restricted, err := cluster.NodeGroup(o.RestrictedGroupName)
	if err != nil {
		return err
	}

	for _, node := range restricted.Nodes() {
		client := node.Client()

		token, err := client.Authenticate(ctx, o.Role, o.AdminUsername, o.AdminPasswordHash)
		if err != nil {
			return fmt.Errorf("authorize: %w", err)
		}

		fmt.Println("got token", token)

		status, err := client.Health(ctx, token)
		if err != nil {
			return fmt.Errorf("health check: %w", err)
		}

		if status != "ok" {
			return fmt.Errorf("expected status 'ok', got: %s", status)
		}
	}

	fmt.Println("authenticated health check completed successfully")
	return
}

// dryRun does nothing
func dryRun(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	return
}
