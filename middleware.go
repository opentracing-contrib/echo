package apm

import (
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type mwOptions struct {
	opNameFunc    func(c echo.Context) string
	spanObserver  func(span opentracing.Span, c echo.Context)
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
				opNameFunc: func(ec echo.Context) string {
					return ec.Request().Proto + " " + ec.Request().Method + " " + ec.Path()
				},
				spanObserver: func(span opentracing.Span, c echo.Context) {

				},
				urlTagFunc: func(u *url.URL) string {
					return u.String()
				},
			}

			carrier := opentracing.HTTPHeadersCarrier(r.Header)
			ctx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)
			op := opts.opNameFunc(c)
			sp := opentracing.StartSpan(op, ext.RPCServerOption(ctx))
			defer sp.Finish()

			ext.HTTPMethod.Set(sp, r.Method)
			ext.HTTPUrl.Set(sp, opts.urlTagFunc(r.URL))
			opts.spanObserver(sp, c)
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
			ext.HTTPStatusCode.Set(sp, uint16(c.Response().Status))

			return nil

		}
	}
}
