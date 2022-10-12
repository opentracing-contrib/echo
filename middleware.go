package apm

import (
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net/http"
	"net/url"
)

type mwOptions struct {
	opNameFunc    func(r *http.Request) string
	spanObserver  func(span opentracing.Span, r *http.Request)
	urlTagFunc    func(u *url.URL) string
	componentName string
}

type ErrorHandler func(echo.Context, error, *http.Request, opentracing.Span)

func DefaultErrorHandlerError(c echo.Context, err error, _ *http.Request, sp opentracing.Span) {
	sp.SetTag("error", true)
}

func Middleware(componentName string) echo.MiddlewareFunc {
	return MiddlewareWithErrorHandler(componentName, DefaultErrorHandlerError)
}

func MiddlewareWithErrorHandler(componentName string, errorHandler ErrorHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			r := c.Request()
			tracer := opentracing.GlobalTracer()

			opts := mwOptions{
				componentName: componentName,
				opNameFunc: func(r *http.Request) string {
					return "HTTP " + r.Method + " " + r.URL.Path
				},
				spanObserver: func(span opentracing.Span, r *http.Request) {

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

			err = next(c)
			if err != nil {
				errorHandler(c, err, r, sp)
			} else {
				sp.SetTag("error", false)
			}
			
			ext.HTTPStatusCode.Set(sp, uint16(c.Response().Status))

			return err

		}
	}
}
