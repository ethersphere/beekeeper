package check

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/push"
)

type CheckRunner struct {
	globalConfig  config.CheckGlobalConfig
	checks        map[string]config.Check
	cluster       orchestration.Cluster
	metricsPusher *push.Pusher
	tracer        opentracing.Tracer
	logger        logging.Logger
}

func NewCheckRunner(
	globalConfig config.CheckGlobalConfig,
	checks map[string]config.Check,
	cluster orchestration.Cluster,
	metricsPusher *push.Pusher,
	tracer opentracing.Tracer,
	logger logging.Logger,
) *CheckRunner {
	if logger == nil {
		logger = logging.New(io.Discard, 0)
	}
	return &CheckRunner{
		globalConfig:  globalConfig,
		checks:        checks,
		cluster:       cluster,
		metricsPusher: metricsPusher,
		tracer:        tracer,
		logger:        logger,
	}
}

func (c *CheckRunner) Run(ctx context.Context, checks []string) error {
	if len(checks) == 0 {
		return nil
	}

	validatedChecks := make([]checkRun, 0, len(checks))

	// validate and prepare checks
	for _, checkName := range checks {
		checkName = strings.TrimSpace(checkName)
		// get configuration
		checkConfig, ok := c.checks[checkName]
		if !ok {
			return fmt.Errorf("check '%s' doesn't exist", checkName)
		}

		// choose checkType type
		checkType, ok := config.Checks[checkConfig.Type]
		if !ok {
			return fmt.Errorf("check %s not implemented", checkConfig.Type)
		}

		// create check options
		o, err := checkType.NewOptions(c.globalConfig, checkConfig)
		if err != nil {
			return fmt.Errorf("creating check %s options: %w", checkName, err)
		}

		// create check action
		chk := checkType.NewAction(c.logger)
		if r, ok := chk.(metrics.Reporter); ok && c.metricsPusher != nil {
			metrics.RegisterCollectors(c.metricsPusher, r.Report()...)
		}
		chk = beekeeper.NewActionMiddleware(c.tracer, chk, checkName)

		// append to validated checks
		validatedChecks = append(validatedChecks, checkRun{
			name:     checkName,
			typeName: checkConfig.Type,
			action:   chk,
			options:  o,
			timeout:  *checkConfig.Timeout,
		})
	}

	type errorCheck struct {
		check string
		err   error
	}

	var errors []errorCheck

	for _, check := range validatedChecks {
		c.logger.WithField("type", check.typeName).Infof("running check: %s", check.name)
		c.logger.Debugf("check options: %+v", check.options)

		if err := check.Run(ctx, c.cluster); err != nil {
			c.logger.WithField("type", check.typeName).Errorf("check '%s' failed: %v", check.name, err)
			errors = append(errors, errorCheck{
				check: check.name,
				err:   fmt.Errorf("check %s failed: %w", check.name, err),
			})
		} else {
			c.logger.WithField("type", check.typeName).Infof("%s check completed successfully", check.name)
		}
	}

	if len(errors) == 1 {
		return errors[0].err
	} else if len(errors) > 1 {
		var errStrings []string
		for _, e := range errors {
			errStrings = append(errStrings, fmt.Sprintf("[%s]: {%v}", e.check, e.err))
			c.logger.Errorf("%s: %v", e.check, e.err)
		}
		return fmt.Errorf("multiple checks failed: %s", strings.Join(errStrings, "; "))
	}

	return nil
}

type checkRun struct {
	name     string
	typeName string
	action   beekeeper.Action
	options  interface{}
	timeout  time.Duration
}

func (c *checkRun) Run(ctx context.Context, cluster orchestration.Cluster) error {
	checkCtx, cancelCheck := createChildContext(ctx, &c.timeout)
	defer cancelCheck()

	ch := make(chan error, 1)
	go func() {
		ch <- c.action.Run(checkCtx, cluster, c.options)
		close(ch)
	}()

	select {
	case <-checkCtx.Done():
		deadline, ok := checkCtx.Deadline()
		if ok {
			return fmt.Errorf("%w: deadline %v", checkCtx.Err(), deadline)
		}
		return checkCtx.Err()
	case err := <-ch:
		if err != nil {
			return err
		}
		return nil
	}
}

func createChildContext(ctx context.Context, timeout *time.Duration) (context.Context, context.CancelFunc) {
	if timeout != nil {
		return context.WithTimeout(ctx, *timeout)
	}
	return context.WithCancel(ctx)
}
