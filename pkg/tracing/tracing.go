package tracing

import (
	"io"
	"time"

	"github.com/uber/jaeger-client-go"
	jconfig "github.com/uber/jaeger-client-go/config"

	"github.com/opentracing/opentracing-go"
)

type Options struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
}

// NewTracer creates a new Tracer and returns a closer which needs to be closed
// when the Tracer is no longer used to flush remaining traces.
func NewTracer(o *Options) (opentracing.Tracer, io.Closer, error) {
	if o == nil {
		o = new(Options)
	}

	cfg := jconfig.Configuration{
		Disabled:    !o.Enabled,
		ServiceName: o.ServiceName,
		Sampler: &jconfig.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  o.Endpoint,
		},
	}

	t, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, nil, err
	}
	return t, closer, nil
}
