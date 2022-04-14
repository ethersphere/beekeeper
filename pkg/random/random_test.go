package random

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
)

func TestPseudoGenerator(t *testing.T) {
	testTable := []struct {
		seed     int64
		expected int64
	}{
		{
			seed:     0,
			expected: 3754598621523947749,
		},
		{
			seed:     10,
			expected: 4284471401503065690,
		},
		{
			seed:     -10,
			expected: 1407906994908968824,
		},
		{
			seed: time.Now().Unix(),
		},
	}

	for run, test := range testTable {
		t.Run(fmt.Sprintf("test_%v", run), func(t *testing.T) {
			g := PseudoGenerator(test.seed)
			if g != nil {
				num := g.Int63()
				if test.expected != num && test.expected != 0 {
					t.Errorf("expected: %v, got: %v", test.expected, num)
				}

				if num == g.Int63() {
					t.Errorf("calling method shouldn't return again the same number")
				}

			} else {
				t.Error("pseudo generator returned nil")
			}
		})
	}
}

func TestPseudoGenerators(t *testing.T) {
	testTable := []struct {
		seed     int64
		n        int
		expected []int64
	}{
		{
			n: 0,
		},
		{
			n: -10,
		},
		{
			seed:     10,
			expected: []int64{4284471401503065690, 1483164964273616633, 3824136261363411275, 8905602456304777631, 4805433114901367189, 8095534066216023294, 3967278764332372921, 1140148681249803769, 7855303061557817261, 1371669227469426764},
			n:        10,
		},
	}

	for run, test := range testTable {
		t.Run(fmt.Sprintf("test_%v", run), func(t *testing.T) {
			g := PseudoGenerators(test.seed, test.n)
			if test.n <= 0 && g != nil {
				t.Error("result slice should be nil")
			} else if test.n > 0 {

				if g == nil {
					t.Fatal("result slice shouldn't be nil")
				}

				if len(g) != test.n {
					t.Errorf("result slice length expected: %v, got: %v", test.n, len(g))
				}

				for i := 0; i < test.n; i++ {
					num := g[i].Int63()
					if num != test.expected[i] {
						t.Errorf("result slice on index: %v, expected: %v, got: %v", i, test.expected[i], num)
					}
					if num == g[i].Int63() {
						t.Errorf("calling method for index: %v shouldn't return again the same number", i)
					}
				}
			}
		})
	}
}

func TestInt64(t *testing.T) {
	v := Int64()
	vt := reflect.TypeOf(v).Kind()

	if vt != reflect.Int64 {
		t.Errorf("unexpected type, expected: %v, got: %v", reflect.Int64, vt)
	}

	// TODO should we check this, maybe in some loop?
	if !(v > 0 && v < math.MaxInt64) {
		t.Errorf("range not expected")
	}
}
