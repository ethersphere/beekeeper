package authenticated

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents check options
type Options struct {
	DryRun              bool
	Role                string
	AdminPassword       string
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running authenticated check")

	restricted, err := cluster.NodeGroup(o.RestrictedGroupName)
	if err != nil {
		return err
	}

	var node orchestration.Node
	for _, node = range restricted.Nodes() {
		client := node.Client()

		// refresh with bad token
		if _, err := client.Refresh(ctx, "bad-token"); err == nil {
			return errors.New("expected error when making a call while unauthenticated")
		}

		// auth with bad password
		token, err := node.Client().Authenticate(ctx, o.Role, "wrong-password")
		if err == nil {
			return fmt.Errorf("expected error when authenticating with bad credentials")
		}
		if token != "" {
			return fmt.Errorf("want empty token, got %s", token)
		}

		// successful auth
		token, err = client.Authenticate(ctx, o.Role, o.AdminPassword)
		if err != nil {
			return fmt.Errorf("authenticate: %w", err)
		}

		// successful refresh
		newToken, err := client.Refresh(ctx, token)
		if err != nil {
			return fmt.Errorf("refresh: %w", err)
		}

		if len(newToken) == 0 {
			return fmt.Errorf("expected a new token, got: %v", newToken)
		}
	}

	fmt.Println("authenticated check completed successfully")
	return
}

// dryRun does nothing
func dryRun(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	fmt.Println("running authenticated check (dry run mode)")
	return nil //success
}
