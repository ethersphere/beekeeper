package beekeeper

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/opentracing/opentracing-go"
)

var _ Action = (*actionMiddleware)(nil)

type actionMiddleware struct {
	tracer     opentracing.Tracer
	action     Action
	actionName string
}

// ActionMiddleware tracks request, and adds spans
// to context.
func NewActionMiddleware(tracer opentracing.Tracer, action Action, actionName string) Action {
	return &actionMiddleware{
		tracer:     tracer,
		action:     action,
		actionName: actionName,
	}
}

// Run implements beekeeper.Action
func (am *actionMiddleware) Run(ctx context.Context, cluster orchestration.Cluster, o any) (err error) {
	span := createSpan(ctx, am.tracer, am.actionName)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return am.action.Run(ctx, cluster, o)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
