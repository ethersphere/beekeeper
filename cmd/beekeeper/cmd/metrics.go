package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// newMetricsPusher returns a new metrics pusher and a cleanup function.
func newMetricsPusher(pusherAddress, job string) (*push.Pusher, func()) {
	metricsPusher := push.New(pusherAddress, job)
	metricsPusher.Format(expfmt.FmtText)

	killC := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)

	// start period flusher
	go func() {
		defer wg.Done()
		for {
			select {
			case <-killC:
				return
			case <-time.After(time.Second):
				if err := metricsPusher.Push(); err != nil {
					fmt.Printf("metrics pusher periodic push: %v\n", err)
				}
			}
		}
	}()
	cleanupFn := func() {
		close(killC)
		wg.Wait()
		// push metrics before returning
		if err := metricsPusher.Push(); err != nil {
			fmt.Printf("metrics pusher push: %v\n", err)
		}
	}
	return metricsPusher, cleanupFn
}
