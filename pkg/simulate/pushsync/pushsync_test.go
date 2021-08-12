package pushsync_test

import (
	"fmt"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/simulate/pushsync"
)

func TestGetIPFromUnderlays(t *testing.T) {

	ips := []string{
		"/ip4/127.0.0.1/udp/9090/quic",
		"/ip6/::1/tcp/3217",
		"/ip4/10.0.0.0/tcp/80/http/baz.jpg",
	}
	ip := pushsync.GetIPFromUnderlays(ips)

	if ip != "10.0.0.0" {
		t.Fatalf("want %s got %s", ip, "10.0.0.0")
	}
}

func TestTobuckets(t *testing.T) {

	for _, tc := range []struct {
		name     string
		base     []string
		buckets  [][]string
		leftover []string
		start    float64
		end      float64
		step     float64
	}{
		{
			name:     "0.1-0.8-0.1",
			base:     []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			buckets:  [][]string{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}, {"6"}, {"7"}, {"8"}},
			leftover: []string{"9", "10"},
			start:    0.1,
			end:      0.8,
			step:     0.1,
		},
		{
			name:     "0.2-0.9-0.1",
			base:     []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			buckets:  [][]string{{"1", "2"}, {"3"}, {"4"}, {"5"}, {"6"}, {"7"}, {"8"}, {"9"}},
			leftover: []string{"10"},
			start:    0.2,
			end:      0.9,
			step:     0.1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buckets, leftover := pushsync.ToBuckets(tc.base, tc.start, tc.end, tc.step)

			fmt.Println(buckets)
			fmt.Println(leftover)
			fmt.Println(tc.leftover)

			if !isArrSame(leftover, tc.leftover) {
				t.Fatal("leftovers do not match")
			}

			if len(buckets) != len(tc.buckets) {
				t.Fatal("len of bucks do not match")
			}

			for i := range buckets {
				if !isArrSame(buckets[i], tc.buckets[i]) {
					t.Fatal("buckets do not match")
				}
			}
		})
	}
}

func isArrSame(a, b []string) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
