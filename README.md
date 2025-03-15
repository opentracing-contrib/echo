# echo

[![CI](https://github.com/opentracing-contrib/echo/actions/workflows/ci.yml/badge.svg)](https://github.com/opentracing-contrib/echo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/opentracing-contrib/echo)](https://goreportcard.com/report/github.com/opentracing-contrib/echo)
[![GoDoc](https://godoc.org/github.com/opentracing-contrib/echo?status.svg)](https://pkg.go.dev/github.com/opentracing-contrib/echo)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/opentracing-contrib/echo?logo=github&sort=semver)](https://github.com/opentracing-contrib/echo/releases/latest)

A middleware for the echov4 web framework to use OpenTracing

```go
package main

import (
  "github.com/labstack/echo/v4"
  apmecho "github.com/opentracing-contrib/echo"
  "github.com/opentracing-contrib/echo/examples/tracer"
  "github.com/opentracing/opentracing-go"
  "net/http"
  "os"
)

const (
  DefaultComponentName = "echo-demo"
)

func main() {

  flag := os.Getenv("JAEGER_ENABLED")
  if flag == "true" {
    // 1. init tracer
    tracer, closer := tracer.Init(DefaultComponentName)
    if closer != nil {
      defer closer.Close()
    }
    // 2. ste the global tracer
    if tracer != nil {
      opentracing.SetGlobalTracer(tracer)
    }
  }

  e := echo.New()

  if flag == "true" {
    // 3. use the middleware
    e.Use(apmecho.Middleware(DefaultComponentName))
  }

  e.GET("/", func(c echo.Context) error {
    return c.String(http.StatusOK, "Hello, World!")
  })

  e.Logger.Fatal(e.Start(":1323"))
}

```

Example: [echo-example](./examples)

![Echo tracing example screenshot 1](./examples/imgs/img1.jpg)

![Echo tracing example screenshot 2](./examples/imgs/img2.jpg)
