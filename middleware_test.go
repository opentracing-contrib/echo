package apm_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	apm "github.com/opentracing-contrib/echo"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
)

func TestOperationNameOption(t *testing.T) {
	fn := func(c echo.Context) string {
		return "HTTP " + c.Request().Method + ": /root"
	}

	tests := []struct {
		options []apm.Option
		url     string
		opName  string
	}{
		{nil, "/", "HTTP/1.1 GET /"},
		{nil, "/users/10", "HTTP/1.1 GET /users/:id"},
		{nil, "/groups/10", "HTTP/1.1 GET /groups/10"},
		{[]apm.Option{apm.OperationNameFunc(fn)}, "/", "HTTP GET: /root"},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.opName, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr, testCase.options...)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})
			e.GET("/users/:id", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			e.ServeHTTP(rec, req)

			spans := tr.FinishedSpans()
			if got, want := len(spans), 1; got != want {
				t.Fatalf("got %d spans, expected %d", got, want)
			}

			if got, want := spans[0].OperationName, testCase.opName; got != want {
				t.Fatalf("got %s operation name, expected %s", got, want)
			}
		})
	}
}

func TestSpanObserverOption(t *testing.T) {
	opNamefn := func(c echo.Context) string {
		return "HTTP " + c.Request().Method + ": /root"
	}
	spanObserverfn := func(sp opentracing.Span, c echo.Context) {
		sp.SetTag("http.uri", c.Request().URL.EscapedPath())
	}
	wantTags := map[string]interface{}{"http.uri": "/"}

	tests := []struct {
		options []apm.Option
		opName  string
		Tags    map[string]interface{}
	}{
		{nil, "HTTP/1.1 GET /", nil},
		{[]apm.Option{apm.OperationNameFunc(opNamefn)}, "HTTP GET: /root", nil},
		{[]apm.Option{apm.SpanObserver(spanObserverfn)}, "HTTP/1.1 GET /", wantTags},
		{[]apm.Option{apm.OperationNameFunc(opNamefn), apm.SpanObserver(spanObserverfn)}, "HTTP GET: /root", wantTags},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.opName, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr, testCase.options...)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			e.ServeHTTP(rec, req)

			spans := tr.FinishedSpans()
			if got, want := len(spans), 1; got != want {
				t.Fatalf("got %d spans, expected %d", got, want)
			}

			if got, want := spans[0].OperationName, testCase.opName; got != want {
				t.Fatalf("got %s operation name, expected %s", got, want)
			}

			defaultLength := 5
			if len(spans[0].Tags()) != len(testCase.Tags)+defaultLength {
				t.Fatalf("got tag length %d, expected %d", len(spans[0].Tags()), len(testCase.Tags))
			}
			for k, v := range testCase.Tags {
				if tag := spans[0].Tag(k); v != tag.(string) {
					t.Fatalf("got %v tag, expected %v", tag, v)
				}
			}
		})
	}
}

func TestSpanFilterOption(t *testing.T) {
	spanFilterfn := func(c echo.Context) bool {
		return !strings.HasPrefix(c.Request().Header.Get("User-Agent"), "kube-probe")
	}
	noAgentReq := httptest.NewRequest("GET", "/root", nil)
	noAgentReq.Header.Del("User-Agent")
	probeReq1 := httptest.NewRequest("GET", "/root", nil)
	probeReq1.Header.Add("User-Agent", "kube-probe/1.12")
	probeReq2 := httptest.NewRequest("GET", "/root", nil)
	probeReq2.Header.Add("User-Agent", "kube-probe/9.99")
	postmanReq := httptest.NewRequest("GET", "/root", nil)
	postmanReq.Header.Add("User-Agent", "PostmanRuntime/7.3.0")
	tests := []struct {
		options            []apm.Option
		request            *http.Request
		opName             string
		ExpectToCreateSpan bool
	}{
		{nil, noAgentReq, "No filter", true},
		{[]apm.Option{apm.SpanFilter(spanFilterfn)}, noAgentReq, "No User-Agent", true},
		{[]apm.Option{apm.SpanFilter(spanFilterfn)}, probeReq1, "User-Agent: kube-probe/1.12", false},
		{[]apm.Option{apm.SpanFilter(spanFilterfn)}, probeReq2, "User-Agent: kube-probe/9.99", false},
		{[]apm.Option{apm.SpanFilter(spanFilterfn)}, postmanReq, "User-Agent: PostmanRuntime/7.3.0", true},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.opName, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr, testCase.options...)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, testCase.request)

			spans := tr.FinishedSpans()
			if spanCreated := len(spans) == 1; spanCreated != testCase.ExpectToCreateSpan {
				t.Fatalf("spanCreated %t, ExpectToCreateSpan %t", spanCreated, testCase.ExpectToCreateSpan)
			}
		})
	}
}

func TestURLTagOption(t *testing.T) {
	fn := func(u *url.URL) string {
		// Log path only (no query parameters etc)
		return u.Path
	}

	tests := []struct {
		options []apm.Option
		url     string
		tag     string
	}{
		{[]apm.Option{}, "/root?token=123", "/root?token=123"},
		{[]apm.Option{apm.URLTagFunc(fn)}, "/root?token=123", "/root"},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.tag, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr, testCase.options...)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", func(c echo.Context) error {
				return nil
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			e.ServeHTTP(rec, req)

			spans := tr.FinishedSpans()
			if got, want := len(spans), 1; got != want {
				t.Fatalf("got %d spans, expected %d", got, want)
			}

			tag := spans[0].Tags()["http.url"]
			if got, want := tag, testCase.tag; got != want {
				t.Fatalf("got %s tag name, expected %s", got, want)
			}
		})
	}
}

func TestSpanError(t *testing.T) {
	wantTags := map[string]interface{}{string(ext.Error): true}

	tests := []struct {
		url  string
		Tags map[string]interface{}
	}{
		{"/root", make(map[string]interface{})},
		{"/error", wantTags},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.url, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})
			e.GET("/error", func(c echo.Context) error {
				return c.NoContent(500)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			e.ServeHTTP(rec, req)

			spans := tr.FinishedSpans()
			if got, want := len(spans), 1; got != want {
				t.Fatalf("got %d spans, expected %d", got, want)
			}

			for k, v := range testCase.Tags {
				if tag := spans[0].Tag(k); v != tag.(bool) {
					t.Fatalf("got %v tag, expected %v", tag, v)
				}
			}
		})
	}
}

func BenchmarkStatusCodeTrackingOverhead(b *testing.B) {
	tr := &mocktracer.MockTracer{}
	mw := apm.Middleware(tr)

	e := echo.New()
	e.Use(mw)
	e.GET("/root", func(c echo.Context) error {
		return nil
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			e.ServeHTTP(rec, req)
		}
	})
}

func TestMiddlewareHandlerPanic(t *testing.T) {
	tests := []struct {
		handler func(c echo.Context) error
		status  uint16
		isError bool
		tag     string
	}{
		{
			func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			},
			http.StatusOK,
			false,
			"OK",
		},
		{
			func(c echo.Context) error {
				panic("panic test")
			},
			0,
			true,
			"Panic",
		},
		{
			func(c echo.Context) error {
				return c.String(http.StatusInternalServerError, "InternalServerError")
			},
			http.StatusInternalServerError,
			true,
			"InternalServerError",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.tag, func(t *testing.T) {
			tr := &mocktracer.MockTracer{}
			mw := apm.Middleware(tr)

			e := echo.New()
			e.Use(mw)
			e.GET("/root", testCase.handler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/root", nil)
			e.ServeHTTP(rec, req)

			spans := tr.FinishedSpans()
			if got, want := len(spans), 1; got != want {
				t.Fatalf("got %d spans, expected %d", got, want)
			}
			actualStatus := spans[0].Tag(string(ext.HTTPStatusCode)).(uint16)
			if testCase.status != actualStatus {
				t.Fatalf("got status code %d, expected %d", actualStatus, testCase.status)
			}
			actualErr, ok := spans[0].Tag(string(ext.Error)).(bool)
			if !ok {
				actualErr = false
			}
			if testCase.isError != actualErr {
				t.Fatalf("got span error %v, expected %v", actualErr, testCase.isError)
			}
		})
	}
}
