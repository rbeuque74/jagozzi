package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPServer(t *testing.T) {
	// creating HTTP server
	srvcfg := HttpServerTimeout{
		StatusCode: 200,
	}
	shutdown := NewHTTPServer(t, srvcfg)

	time.Sleep(20 * time.Millisecond)

	// creating HTTP checker
	cfg := map[string]interface{}{
		"type":    "request",
		"url":     "http://localhost:8080",
		"method":  "GET",
		"warn":    200,
		"crit":    400,
		"timeout": 450,
		"name":    "test-1",
	}
	checker, err := NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err := checker.Run(ctxRun)
	assert.Nilf(t, err, "http error: %q", err)
	assert.Equalf(t, "OK", result, "http bad result: %q", result)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelFunc()
	shutdown(ctx)
}

func TestHTTPServerFails(t *testing.T) {
	// creating HTTP server
	srvcfg := HttpServerTimeout{
		StatusCode: 200,
		Sleep:      time.Millisecond * 80,
	}
	shutdown := NewHTTPServer(t, srvcfg)

	time.Sleep(20 * time.Millisecond)

	// creating HTTP checker
	cfg := map[string]interface{}{
		"type":    "request",
		"url":     "http://localhost:8080",
		"method":  "GET",
		"warn":    40,
		"crit":    200,
		"timeout": 1000,
		"name":    "test-1",
	}
	checker, err := NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err := checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, "warning", err.Error())
	assert.Equal(t, "", result)

	// critical
	cfg["warn"] = 10
	cfg["crit"] = 50
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err = checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, "critical", err.Error())
	assert.Equal(t, "", result)

	// bad status code
	cfg["code"] = 400
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err = checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, "invalid status code: 200 instead of 400", err.Error())
	assert.Equal(t, "", result)

	// conn refused
	cfg["url"] = "http://localhost:8081"
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err = checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Containsf(t, err.Error(), "connection refused", "err is not connection refused: %q", err)
	assert.Equal(t, "", result)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelFunc()
	shutdown(ctx)
}
