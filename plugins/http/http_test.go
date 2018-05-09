package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPServer(t *testing.T) {
	// creating HTTP server
	srvcfg := FakeTestHTTPServer{
		StatusCode: 200,
	}
	shutdown := NewHTTPServer(t, srvcfg)
	defer func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelFunc()
		shutdown(ctx)
	}()

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
}

func TestHTTPServerFails(t *testing.T) {
	// creating HTTP server
	srvcfg := FakeTestHTTPServer{
		StatusCode: 200,
		Sleep:      time.Millisecond * 80,
	}
	shutdown := NewHTTPServer(t, srvcfg)
	defer func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelFunc()
		shutdown(ctx)
	}()

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
	assert.Contains(t, err.Error(), "timeout: request took")
	assert.Contains(t, err.Error(), "instead of 40ms")
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
	assert.Contains(t, err.Error(), "critical timeout: request took")
	assert.Contains(t, err.Error(), "instead of 50ms")
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

	// bad method
	cfg["code"] = 200
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	httpChecker := checker.(*HTTPChecker)
	httpChecker.cfg.Method = "http not valid method"

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err = httpChecker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, `net/http: invalid method "http not valid method"`, err.Error())
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
}

func TestHTTPServerFailsTemplating(t *testing.T) {
	// creating HTTP server
	srvcfg := FakeTestHTTPServer{
		StatusCode: 200,
		Sleep:      time.Millisecond * 80,
	}
	shutdown := NewHTTPServer(t, srvcfg)
	defer func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelFunc()
		shutdown(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	// creating HTTP checker
	templatesCfg := rawTemplates{
		ErrStatusCode:      "personnalised: received status code {{.Response.StatusCode}}, I was looking for {{.Cfg.Code}}; original: {{.Err}}",
		ErrTimeoutCritical: "personnalised: critical timeout {{.ElapsedTime}} instead of {{.Cfg.Critical}}",
	}
	cfg := map[string]interface{}{
		"type":      "request",
		"url":       "http://localhost:8080",
		"method":    "GET",
		"warn":      10,
		"crit":      50,
		"timeout":   1000,
		"name":      "test-1",
		"templates": templatesCfg,
	}
	checker, err := NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result, err := checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Contains(t, err.Error(), "personnalised: critical timeout")
	assert.Contains(t, err.Error(), "instead of 50ms")
	assert.Equal(t, "", result)

	// bad status code
	cfg["code"] = 400
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result, err = checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, "personnalised: received status code 200, I was looking for 400; original: invalid status code: 200 instead of 400", err.Error())
	assert.Equal(t, "", result)

}

func TestHTTPServerFailsTemplatingJSON(t *testing.T) {
	// creating HTTP server
	srvcfg := FakeTestHTTPServer{
		StatusCode: 400,
		JSONBody:   true,
	}
	shutdown := NewHTTPServer(t, srvcfg)
	defer func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelFunc()
		shutdown(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	// creating HTTP checker
	templatesCfg := rawTemplates{
		ErrStatusCode: `error while fetching: {{.ResponseBody.message}}`,
	}
	cfg := map[string]interface{}{
		"type":      "request",
		"url":       "http://localhost:8080",
		"method":    "GET",
		"warn":      200,
		"crit":      200,
		"timeout":   1000,
		"name":      "test-1",
		"templates": templatesCfg,
	}
	checker, err := NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result, err := checker.Run(ctxRun)
	assert.NotNilf(t, err, "http no error but should")
	assert.Equal(t, "error while fetching: json message field", err.Error())
	assert.Equal(t, "", result)
}
