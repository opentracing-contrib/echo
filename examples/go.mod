module github.com/opentracing-contrib/echo/examples

go 1.12

replace github.com/flachnetz/echo-apm-middleware => ../

require (
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/flachnetz/echo-apm-middleware v0.0.0-00010101000000-000000000000
	github.com/labstack/echo/v4 v4.7.2
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	go.uber.org/atomic v1.4.0 // indirect
)
