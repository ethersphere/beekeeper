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
			timeout:  checkConfig.Timeout,
		})
	}

	checkResults := make([]checkResult, 0, len(validatedChecks))
	hasFailures := false

	// run checks
	for _, check := range validatedChecks {
		c.logger.WithFields(map[string]any{
			"type":    check.typeName,
			"options": fmt.Sprintf("%+v", check.options),
		}).Infof("running check: %s", check.name)

		err := check.Run(ctx, c.cluster)
		if err != nil {
			hasFailures = true
			c.logger.WithFields(map[string]any{
				"type":  check.typeName,
				"error": err,
			}).Errorf("'%s' check failed", check.name)
		} else {
			c.logger.WithField("type", check.typeName).Infof("'%s' check completed successfully", check.name)
		}

		// append check result
		checkResults = append(checkResults, checkResult{
			check:     check.name,
			err:       err,
			timestamp: time.Now(),
		})
	}

	if hasFailures {
		return formatErrorReport(checkResults)
	}

	c.logger.WithField("total_checks", len(checkResults)).Info("All checks completed successfully")
	return nil
}

type checkRun struct {
	name     string
	typeName string
	action   beekeeper.Action
	options  any
	timeout  *time.Duration
}

func (c *checkRun) Run(ctx context.Context, cluster orchestration.Cluster) error {
	checkCtx, cancelCheck := createChildContext(ctx, c.timeout)
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
		return err
	}
}

func createChildContext(ctx context.Context, timeout *time.Duration) (context.Context, context.CancelFunc) {
	if timeout != nil {
		return context.WithTimeout(ctx, *timeout)
	}
	return context.WithCancel(ctx)
}

type checkResult struct {
	check     string
	err       error
	timestamp time.Time
}

func formatErrorReport(results []checkResult) error {
	var failedChecks []string
	var failedDetails []string

	// if there is only one error, return it directly
	if len(results) == 1 {
		return results[0].err
	}

	for _, result := range results {
		if result.err != nil {
			failedChecks = append(failedChecks, result.check)
			failedDetails = append(failedDetails, result.DetailString())
		}
	}

	totalChecks := len(results)
	failedCount := len(failedChecks)

	return fmt.Errorf("CHECK_FAILED | %d/%d checks failed | Checks: %s | %s",
		failedCount,
		totalChecks,
		strings.Join(failedChecks, ","),
		strings.Join(failedDetails, " | "))
}

func (e checkResult) String() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.check, e.err)
	}
	return fmt.Sprintf("%s: success", e.check)
}

func (e checkResult) DetailString() string {
	timestamp := e.timestamp.Format("2006-01-02T15:04:05")
	if e.err != nil {
		return fmt.Sprintf(`{"check":"%s","time":"%s","error":"%v"}`,
			e.check, timestamp, e.err)
	}
	return fmt.Sprintf(`{"check":"%s","time":"%s","status":"success"}`,
		e.check, timestamp)
}
