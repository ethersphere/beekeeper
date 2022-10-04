package smoke

import (
	"sync"
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/random"
)

func TestMovingWindow(t *testing.T) {
	nodes := []string{"a", "b", "c", "d", "e"}
	rnd := random.PseudoGenerator(time.Now().Unix())

	t.Run("given batch size equal to peers", func(t *testing.T) {
		batchSize := 5
		t.Run("returns all peers for any iteration", func(t *testing.T) {
			sel := movingWindow(0, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
			sel = movingWindow(1, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
			sel = movingWindow(11, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
		})
	})
	t.Run("given batch size larger than peers", func(t *testing.T) {
		batchSize := 15
		t.Run("returns all peers for any iteration", func(t *testing.T) {
			sel := movingWindow(0, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
			sel = movingWindow(1, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
			sel = movingWindow(11, batchSize, nodes, rnd)
			if len(sel) != len(nodes) {
				t.Fatal()
			}
		})
	})
	t.Run("given batch size smaller than peers", func(t *testing.T) {
		batchSize := 2
		t.Run("wraps around", func(t *testing.T) {
			sel := movingWindow(0, batchSize, nodes, rnd)
			if len(sel) != 2 {
				t.Fatal()
			}
			sel = movingWindow(1, batchSize, nodes, rnd)
			if len(sel) != 2 {
				t.Fatal()
			}
			sel = movingWindow(2, batchSize, nodes, rnd)
			if len(sel) != 1 {
				t.Fatal()
			}
			sel = movingWindow(3, batchSize, nodes, rnd)
			if len(sel) != 2 {
				t.Fatal()
			}
			sel = movingWindow(4, batchSize, nodes, rnd)
			if len(sel) != 2 {
				t.Fatal()
			}
		})
	})
}

func Test112(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		wg.Done()
	}()

	wg.Done()

	time.Sleep(time.Second)
}
