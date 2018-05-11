package http

import (
	"context"
	"testing"
	"time"

	"github.com/rbeuque74/jagozzi/plugins"
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

	assert.Equal(t, "HTTP", checker.Name())
	assert.Equal(t, "test-1", checker.ServiceName())

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Containsf(t, result.Message, "200 OK", "http bad message: %q", result.Message)
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
	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Contains(t, result.Message, "timeout: request took")
	assert.Contains(t, result.Message, "instead of 40ms")

	// critical
	cfg["warn"] = 10
	cfg["crit"] = 50
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Contains(t, result.Message, "critical timeout: request took")
	assert.Contains(t, result.Message, "instead of 50ms")

	// bad status code
	cfg["code"] = 400
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "invalid status code: 200 instead of 400", result.Message)

	// bad method
	cfg["code"] = 200
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	httpChecker := checker.(*HTTPChecker)
	httpChecker.cfg.Method = "http not valid method"

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = httpChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, `net/http: invalid method "http not valid method"`, result.Message)

	// conn refused
	cfg["url"] = "http://localhost:8081"
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Containsf(t, result.Message, "connection refused", "err is not connection refused: %q", result.Message)
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

	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Contains(t, result.Message, "personnalised: critical timeout")
	assert.Contains(t, result.Message, "instead of 50ms")

	// bad status code
	cfg["code"] = 400
	checker, err = NewHTTPChecker(cfg, nil)
	assert.Nilf(t, err, "http checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "personnalised: received status code 200, I was looking for 400; original: invalid status code: 200 instead of 400", result.Message)
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

	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "error while fetching: json message field", result.Message)
}
