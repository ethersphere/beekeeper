package nuker

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()

	c := New(&ClientConfig{
		UseRandomNeighborhood: true,
		Image:                 "bee:test",
		ForgetOverlay:         true,
		ForgetStamps:          true,
	})

	if !c.useRandomNeighborhood {
		t.Error("expected useRandomNeighborhood to be set")
	}
	if c.image != "bee:test" {
		t.Errorf("expected image bee:test, got %s", c.image)
	}
	if !c.forgetOverlay {
		t.Error("expected forgetOverlay to be set")
	}
	if !c.forgetStamps {
		t.Error("expected forgetStamps to be set")
	}
}
