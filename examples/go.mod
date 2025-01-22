module github.com/opentracing-contrib/echo/examples

go 1.12

replace github.com/opentracing-contrib/echo => ../

require (
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/labstack/echo/v4 v4.13.3
	github.com/opentracing-contrib/echo v0.0.0-20190807091611-5fe2e1308f06
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.16.0+incompatible
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	go.uber.org/atomic v1.4.0 // indirect
)
