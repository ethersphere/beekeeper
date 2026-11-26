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

func TestFormatNeighborhood(t *testing.T) {
	t.Parallel()

	tests := []struct {
		radius uint8
		val    int
		want   string
	}{
		{radius: 1, val: 0, want: "0"},
		{radius: 1, val: 1, want: "1"},
		{radius: 2, val: 2, want: "10"},
		{radius: 3, val: 0, want: "000"},
		{radius: 3, val: 1, want: "001"},
		{radius: 3, val: 3, want: "011"},
		{radius: 3, val: 7, want: "111"},
		{radius: 8, val: 5, want: "00000101"},
	}

	for _, tc := range tests {
		if got := formatNeighborhood(tc.radius, tc.val); got != tc.want {
			t.Errorf("formatNeighborhood(%d, %d) = %q, want %q", tc.radius, tc.val, got, tc.want)
		}
	}
}
