package authenticated

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	test "github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents check options
type Options struct {
	DryRun              bool
	Role                string
	AdminPassword       string
	RestrictedGroupName string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() (opts Options) {
	return
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return new(Check)
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

	caseOpts := test.CaseOptions{
		AdminPassword:       o.AdminPassword,
		RestrictedGroupName: o.RestrictedGroupName,
		Role:                o.Role,
	}

	checkCase, err := test.NewCheckCase(ctx, cluster, caseOpts)
	if err != nil {
		return err
	}

	// filter func
	restricted := func(bee *test.BeeV2) bool {
		return bee.Restricted()
	}

	// testing closure
	checkAuth := testAuth(ctx, o)

	// execute test
	if err := checkCase.Bees().Filter(restricted).ForEach(checkAuth); err != nil {
		return err
	}

	fmt.Println("authenticated check completed successfully")
	return
}

func testAuth(ctx context.Context, o Options) test.ConsumeFunc {
	return func(bee *test.BeeV2) error {
		fmt.Println("testing authentication on", bee.Name())

		// refresh with bad token
		if _, err := bee.RefreshAuthToken(ctx, "bad-token"); err == nil {
			return errors.New("expected error when making a call while unauthenticated")
		}

		// auth with bad password
		token, err := bee.Authenticate(ctx, "wrong-password")
		if err == nil {
			return fmt.Errorf("expected error when authenticating with bad credentials")
		}
		if token != "" {
			return fmt.Errorf("want empty token got %s", token)
		}

		// successful auth
		token, err = bee.Authenticate(ctx, o.AdminPassword)
		if err != nil {
			return fmt.Errorf("authenticate: %w", err)
		}

		// successful refresh
		newToken, err := bee.RefreshAuthToken(ctx, token)
		if err != nil {
			return fmt.Errorf("refresh: %w", err)
		}
		if newToken == "" {
			return fmt.Errorf("got empty token, want %s", token)
		}

		return nil
	}
}

// dryRun does nothing
func dryRun(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	fmt.Println("running authenticated check (dry run mode)")
	return nil //success
}
