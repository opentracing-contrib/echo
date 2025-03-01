package apm

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const defaultComponentName = "labstack/echo"

type options struct {
	opNameFunc    func(c echo.Context) string
	spanFilter    func(c echo.Context) bool
	spanObserver  func(span opentracing.Span, c echo.Context)
	urlTagFunc    func(u *url.URL) string
	componentName string
}

// Option controls the behavior of the Middleware.
type Option func(*options)

// OperationNameFunc returns a Option that uses given function f
// to generate operation name for each server-side span.
func OperationNameFunc(f func(c echo.Context) string) Option {
	return func(o *options) {
		o.opNameFunc = f
	}
}

// ComponentName returns a Option that sets the component name
// for the server-side span.
func ComponentName(componentName string) Option {
	return func(o *options) {
		o.componentName = componentName
	}
}

// SpanFilter returns a Option that filters requests from creating a span
// for the server-side span.
// Span won't be created if it returns false.
func SpanFilter(f func(c echo.Context) bool) Option {
	return func(o *options) {
		o.spanFilter = f
	}
}

// SpanObserver returns a Option that observe the span
// for the server-side span.
func SpanObserver(f func(span opentracing.Span, c echo.Context)) Option {
	return func(o *options) {
		o.spanObserver = f
	}
}

// URLTagFunc returns a Option that uses given function f
// to set the span's http.url tag. Can be used to change the default
// http.url tag, eg to redact sensitive information.
func URLTagFunc(f func(u *url.URL) string) Option {
	return func(o *options) {
		o.urlTagFunc = f
	}
}

// Middleware wraps an http.Handler and traces incoming requests.
// Additionally, it adds the span to the request's context.
//
// By default, the operation name of the spans is set to "HTTP {method}".
// This can be overriden with options.
//
// Example:
//   e := echo.New()
//   e.Use(apm.Middleware())
//   e.GET("/", func(c echo.Context) error {
//      return c.String(http.StatusOK, "Hello, World!")
//   })
//   e.Logger.Fatal(e.Start(":1323"))
//
// The options allow fine tuning the behavior of the middleware.
//
// Example:
//   mw := apm.Middleware(
//      tracer,
//      apm.OperationNameFunc(func(c echo.Context) string {
//	        return c.Request().Proto + " " + c.Request().Method + ":/api/customers"
//      }),
//      apm.SpanObserver(func(sp opentracing.Span, c echo.Context) {
//			sp.SetTag("http.uri", c.Request().URL.EscapedPath())
//		}),
//   )
func Middleware(tracer opentracing.Tracer, o ...Option) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			r := c.Request()
			opts := options{
				opNameFunc: func(ec echo.Context) string {
					url := ec.Path()
					if url == "" {
						url = ec.Request().URL.String()
					}
					return ec.Request().Proto + " " + ec.Request().Method + " " + url
				},
				spanFilter:   func(c echo.Context) bool { return true },
				spanObserver: func(span opentracing.Span, c echo.Context) {},
				urlTagFunc: func(u *url.URL) string {
					return u.String()
				},
			}

			for _, opt := range o {
				opt(&opts)
			}

			if !opts.spanFilter(c) {
				return next(c)
			}

			carrier := opentracing.HTTPHeadersCarrier(r.Header)
			ctx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)

			op := opts.opNameFunc(c)
			sp := tracer.StartSpan(op, ext.RPCServerOption(ctx))

			ext.HTTPMethod.Set(sp, r.Method)
			ext.HTTPUrl.Set(sp, opts.urlTagFunc(r.URL))
			opts.spanObserver(sp, c)

			componentName := opts.componentName
			if componentName == "" {
				componentName = defaultComponentName
			}
			ext.Component.Set(sp, componentName)

			r = r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
			c.SetRequest(r)

			defer func() {
				if v := recover(); v != nil {
					err, ok := v.(error)
					if !ok {
						err = fmt.Errorf("%v", v)
					}

					c.Error(err)
					ext.HTTPStatusCode.Set(sp, uint16(0))
					ext.Error.Set(sp, true)
					sp.Finish()
					return
				}

				status := c.Response().Status
				ext.HTTPStatusCode.Set(sp, uint16(status))
				if status >= http.StatusInternalServerError {
					ext.Error.Set(sp, true)
				}
				sp.Finish()
			}()

			err := next(c)
			return err

		}
	}
}
