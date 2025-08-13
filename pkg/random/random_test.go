package random_test

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/random"
)

// abs returns the absolute value of x (helper function for tests)
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

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
			g := random.PseudoGenerator(test.seed)
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
			g := random.PseudoGenerators(test.seed, test.n)
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

func TestInt64_Type(t *testing.T) {
	v := random.Int64()
	vt := reflect.TypeOf(v).Kind()

	if vt != reflect.Int64 {
		t.Errorf("unexpected type, expected: %v, got: %v", reflect.Int64, vt)
	}

	if !(v > 0) {
		t.Errorf("value not in expected range, expected to be greater then 0 and less then MaxInt64")
	}
}

func TestCryptoSource_Uint64(t *testing.T) {
	cs := random.CryptoSource{}
	v := cs.Uint64()
	vt := reflect.TypeOf(v).Kind()

	if vt != reflect.Uint64 {
		t.Errorf("unexpected type, expected: %v, got: %v", reflect.Uint64, vt)
	}

	if v == 0 {
		t.Errorf("value should not be 0")
	}
}

func TestCryptoSource_Int63(t *testing.T) {
	cs := random.CryptoSource{}
	v := cs.Int63()
	vt := reflect.TypeOf(v).Kind()

	if vt != reflect.Int64 {
		t.Errorf("unexpected type, expected: %v, got: %v", reflect.Int64, vt)
	}

	if !(v > 0) {
		t.Errorf("value not in expected range, expected to be greater then 0 and less then MaxInt64")
	}
}

func TestCryptoSource_Seed(_ *testing.T) {
	cs := random.CryptoSource{}
	cs.Seed(10)
}

func TestGetRandom_NonUnique(t *testing.T) {
	testCases := []struct {
		name     string
		minVal   int
		maxVal   int
		expected string
	}{
		{
			name:     "positive range",
			minVal:   1,
			maxVal:   10,
			expected: "range 1-10",
		},
		{
			name:     "negative range",
			minVal:   -10,
			maxVal:   -1,
			expected: "range 1-10",
		},
		{
			name:     "mixed range",
			minVal:   -5,
			maxVal:   5,
			expected: "range 5-5",
		},
		{
			name:     "reversed range",
			minVal:   10,
			maxVal:   1,
			expected: "range 1-10",
		},
		{
			name:     "single value",
			minVal:   5,
			maxVal:   5,
			expected: "range 5-5",
		},
		{
			name:     "zero range",
			minVal:   0,
			maxVal:   0,
			expected: "range 0-0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := random.NewGenerator(false) // non-unique

			// Generate multiple values to ensure they're within range
			for range 100 {
				result, err := g.GetRandom(tc.minVal, tc.maxVal)
				if err != nil {
					t.Fatalf("GetRandom failed: %v", err)
				}

				// Calculate expected min and max after abs() and swap
				expectedMin := abs(tc.minVal)
				expectedMax := abs(tc.maxVal)
				if expectedMin > expectedMax {
					expectedMin, expectedMax = expectedMax, expectedMin
				}

				if result < expectedMin || result > expectedMax {
					t.Errorf("result %d is outside expected range [%d, %d]", result, expectedMin, expectedMax)
				}
			}
		})
	}
}

func TestGetRandom_Unique(t *testing.T) {
	testCases := []struct {
		name     string
		minVal   int
		maxVal   int
		expected int // expected number of unique values
	}{
		{
			name:     "small range",
			minVal:   1,
			maxVal:   5,
			expected: 5,
		},
		{
			name:     "negative range",
			minVal:   -3,
			maxVal:   -1,
			expected: 3,
		},
		{
			name:     "mixed range",
			minVal:   -2,
			maxVal:   2,
			expected: 1, // abs(-2) = 2, abs(2) = 2, so range is [2,2] = 1 value
		},
		{
			name:     "single value",
			minVal:   5,
			maxVal:   5,
			expected: 1,
		},
		{
			name:     "zero range",
			minVal:   0,
			maxVal:   0,
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := random.NewGenerator(true) // unique

			// Calculate expected min and max after abs() and swap
			expectedMin := abs(tc.minVal)
			expectedMax := abs(tc.maxVal)
			if expectedMin > expectedMax {
				expectedMin, expectedMax = expectedMax, expectedMin
			}

			generated := make(map[int]bool)

			// Generate all possible unique values
			for i := 0; i < tc.expected; i++ {
				result, err := g.GetRandom(tc.minVal, tc.maxVal)
				if err != nil {
					t.Fatalf("GetRandom failed at iteration %d: %v", i, err)
				}

				// Check range
				if result < expectedMin || result > expectedMax {
					t.Errorf("result %d is outside expected range [%d, %d]", result, expectedMin, expectedMax)
				}

				// Check uniqueness
				if generated[result] {
					t.Errorf("duplicate value generated: %d", result)
				}
				generated[result] = true
			}

			// Verify we got exactly the expected number of unique values
			if len(generated) != tc.expected {
				t.Errorf("expected %d unique values, got %d", tc.expected, len(generated))
			}

			// Try to generate one more - should fail with ErrUniqueNumberExhausted
			_, err := g.GetRandom(tc.minVal, tc.maxVal)
			if !errors.Is(err, random.ErrUniqueNumberExhausted) {
				t.Errorf("expected ErrUniqueNumberExhausted, got: %v", err)
			}
		})
	}
}

func TestGetRandom_Reset(t *testing.T) {
	g := random.NewGenerator(true)

	for range 3 {
		_, err := g.GetRandom(1, 3)
		if err != nil {
			t.Fatalf("GetRandom failed: %v", err)
		}
	}

	_, err := g.GetRandom(1, 3)
	if !errors.Is(err, random.ErrUniqueNumberExhausted) {
		t.Errorf("expected ErrUniqueNumberExhausted, got: %v", err)
	}

	g.Reset()

	for range 3 {
		_, err := g.GetRandom(1, 3)
		if err != nil {
			t.Fatalf("GetRandom failed after reset: %v", err)
		}
	}
}

func TestGetRandom_EdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		minVal  int
		maxVal  int
		wantErr bool
	}{
		{
			name:    "overflow case",
			minVal:  math.MinInt64,
			maxVal:  math.MaxInt64,
			wantErr: true,
		},
		{
			name:    "large range",
			minVal:  1,
			maxVal:  1000000,
			wantErr: false,
		},
		{
			name:    "zero to large",
			minVal:  0,
			maxVal:  100000,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := random.NewGenerator(false) // non-unique

			result, err := g.GetRandom(tc.minVal, tc.maxVal)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedMin := abs(tc.minVal)
			expectedMax := abs(tc.maxVal)
			if expectedMin > expectedMax {
				expectedMin, expectedMax = expectedMax, expectedMin
			}

			if result < expectedMin || result > expectedMax {
				t.Errorf("result %d is outside expected range [%d, %d]", result, expectedMin, expectedMax)
			}
		})
	}
}

func TestGetRandom_Concurrent(t *testing.T) {
	g := random.NewGenerator(true) // unique
	const numGoroutines = 10
	const valuesPerGoroutine = 10

	results := make(chan int, numGoroutines*valuesPerGoroutine)
	errors := make(chan error, numGoroutines*valuesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < valuesPerGoroutine; j++ {
				result, err := g.GetRandom(1, 100)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}
		}()
	}

	generated := make(map[int]bool)
	for range numGoroutines * valuesPerGoroutine {
		select {
		case result := <-results:
			if result < 1 || result > 100 {
				t.Errorf("result %d is outside expected range [1, 100]", result)
			}
			if generated[result] {
				t.Errorf("duplicate value generated: %d", result)
			}
			generated[result] = true
		case err := <-errors:
			t.Fatalf("goroutine error: %v", err)
		}
	}

	expectedUnique := numGoroutines * valuesPerGoroutine
	if len(generated) != expectedUnique {
		t.Errorf("expected %d unique values, got %d", expectedUnique, len(generated))
	}
}

func TestGetRandom_ZeroRange(t *testing.T) {
	t.Run("both_zero_non_unique", func(t *testing.T) {
		g := random.NewGenerator(false) // non-unique

		// Generate multiple values when both min and max are 0
		for i := 0; i < 10; i++ {
			result, err := g.GetRandom(0, 0)
			if err != nil {
				t.Fatalf("GetRandom(0, 0) failed: %v", err)
			}

			// Should always return 0
			if result != 0 {
				t.Errorf("expected 0, got %d", result)
			}
		}
	})

	t.Run("both_zero_unique", func(t *testing.T) {
		g := random.NewGenerator(true) // unique

		// First call should return 0
		result, err := g.GetRandom(0, 0)
		if err != nil {
			t.Fatalf("GetRandom(0, 0) failed: %v", err)
		}
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}

		// Second call should fail with ErrUniqueNumberExhausted
		_, err = g.GetRandom(0, 0)
		if !errors.Is(err, random.ErrUniqueNumberExhausted) {
			t.Errorf("expected ErrUniqueNumberExhausted, got: %v", err)
		}
	})
}

func FuzzPseudoGenerators(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed int64, n int) {
		g := random.PseudoGenerators(seed, n)
		if n <= 0 && g != nil {
			t.Fatal("result slice should be nil")
		}

		if n > 0 {
			if g == nil {
				t.Fatal("result slice shouldn't be nil")
			}

			if len(g) != n {
				t.Errorf("result slice length expected: %v, got: %v", n, len(g))
			}

			rnd := rand.New(rand.NewSource(seed))

			for i := range n {
				num := g[i].Int63()

				expected := rand.New(rand.NewSource(rnd.Int63()))
				if num != expected.Int63() {
					t.Errorf("value not as expected")
				}

				if num == g[i].Int63() {
					t.Errorf("calling method for index: %v shouldn't return again the same number", i)
				}
			}
		}
	})
}
