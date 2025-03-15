package apm

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type mwOptions struct {
	opNameFunc    func(r *http.Request) string
	spanObserver  func(span opentracing.Span, r *http.Request)
	urlTagFunc    func(u *url.URL) string
	componentName string
}

func Middleware(componentName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			tracer := opentracing.GlobalTracer()

			opts := mwOptions{
				componentName: componentName,
				opNameFunc: func(r *http.Request) string {
					return "HTTP " + r.Method + " " + r.URL.Path
				},
				spanObserver: func(span opentracing.Span, r *http.Request) { //nolint:revive
				},
				urlTagFunc: func(u *url.URL) string {
					return u.String()
				},
			}

			carrier := opentracing.HTTPHeadersCarrier(r.Header)
			ctx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)
			op := opts.opNameFunc(r)
			sp := opentracing.StartSpan(op, ext.RPCServerOption(ctx))
			defer sp.Finish()

			ext.HTTPMethod.Set(sp, r.Method)
			ext.HTTPUrl.Set(sp, opts.urlTagFunc(r.URL))
			opts.spanObserver(sp, r)
			ext.Component.Set(sp, opts.componentName)

			r = r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
			c.SetRequest(r)

			err := tracer.Inject(sp.Context(), opentracing.HTTPHeaders, carrier)
			if err != nil {
				panic("SpanContext Inject Error!")
			}

			if err := next(c); err != nil {
				sp.SetTag("error", true)
				c.Error(err)
			}

			sp.SetTag("error", false)
			if status := c.Response().Status; status >= 0 && status <= 65535 {
				ext.HTTPStatusCode.Set(sp, uint16(status))
			} else {
				// Either use a default value or log the issue
				ext.HTTPStatusCode.Set(sp, 0) // Using 0 to indicate invalid status
			}

			return nil
		}
	}
}
