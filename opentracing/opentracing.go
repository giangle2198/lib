package opentracing

import (
	"strings"

	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

func InitOpentracing(addr, name string) error {
	cfg := config.Configuration{
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort: addr,
			LogSpans:           false,
		},
	}
	_, err := cfg.InitGlobalTracer(
		name,
		config.Logger((jaeger.StdLogger)),
		config.Sampler(jaeger.NewConstSampler(true)),
		config.Metrics(metrics.NullFactory),
	)
	if err != nil {
		return err
	}
	return nil
}

func InitOpentracingWithProtocol(addr, name, protocol string) error {
	cfg := config.Configuration{}

	switch strings.ToLower(protocol) {
	case "http":
		cfg = config.Configuration{
			Reporter: &config.ReporterConfig{
				LocalAgentHostPort: addr,
				LogSpans:           false,
			},
		}
	case "udp":
		cfg = config.Configuration{
			Reporter: &config.ReporterConfig{
				LocalAgentHostPort: addr,
				LogSpans:           false,
			},
		}
	}

	_, err := cfg.InitGlobalTracer(
		name,
		config.Logger(jaeger.StdLogger),
		config.Sampler(jaeger.NewConstSampler(true)),
		config.Metrics(metrics.NullFactory),
	)
	if err != nil {
		return err
	}
	return nil
}
