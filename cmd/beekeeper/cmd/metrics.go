package cmd

import (
	"maps"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// newMetricsPusher returns a new metrics pusher and a cleanup function. job sets
// the Pushgateway "job" label; groupingLabels become part of the grouping key, so
// Pushgateway attaches them to every pushed series (Vec and non-Vec alike).
func newMetricsPusher(pusherAddress, job string, groupingLabels map[string]string, logger logging.Logger) (*push.Pusher, func()) {
	metricsPusher := push.New(pusherAddress, job)
	metricsPusher.Format(expfmt.NewFormat(expfmt.TypeTextPlain))

	for name, value := range groupingLabels {
		if value == "" {
			continue
		}
		metricsPusher.Grouping(name, value)
	}

	killC := make(chan struct{})
	var wg sync.WaitGroup

	// start period flusher
	wg.Go(func() {
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
	})
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

// metricsGroupingLabels merges the run's identity labels (cluster and namespace,
// always present so metrics can be filtered by them) with operator-supplied
// labels, which take precedence. Keys must be valid Prometheus label names,
// otherwise every push fails.
func metricsGroupingLabels(clusterName, namespace string, custom map[string]string) map[string]string {
	labels := map[string]string{
		"cluster":   clusterName,
		"namespace": namespace,
	}
	maps.Copy(labels, custom)
	return labels
}
