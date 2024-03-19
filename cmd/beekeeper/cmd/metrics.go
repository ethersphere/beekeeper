package cmd

import (
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// newMetricsPusher returns a new metrics pusher and a cleanup function.
func newMetricsPusher(pusherAddress, job string, logger logging.Logger) (*push.Pusher, func()) {
	metricsPusher := push.New(pusherAddress, job)
	metricsPusher.Format(expfmt.NewFormat(expfmt.TypeTextPlain))

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
					logger.Debugf("metrics pusher periodic push: %v", err)
				}
			}
		}
	}()
	cleanupFn := func() {
		close(killC)
		wg.Wait()
		// push metrics before returning
		if err := metricsPusher.Push(); err != nil {
			logger.Infof("metrics pusher push: %v", err)
		}
	}
	return metricsPusher, cleanupFn
}
