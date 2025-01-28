package tracer

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

// Init creates a new instance of Jaeger tracer.
func Init(serviceName string) (opentracing.Tracer, io.Closer) {

	cfg, err := config.FromEnv()

	if err != nil {
		fmt.Printf("cannot parse jaeger env vars: %v\n", err.Error())
		//os.Exit(1)
		return nil, nil
	}

	cfg.ServiceName = serviceName
	cfg.Sampler.Type = "const"
	cfg.Sampler.Param = 1

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		fmt.Printf("cannot initialize jaeger tracer: %v\n", err.Error())
		//os.Exit(1)
		return nil, nil
	}
	return tracer, closer
}
