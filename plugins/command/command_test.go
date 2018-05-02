package command

import (
	"context"
	"testing"
	"time"

	"github.com/rbeuque74/jagozzi/plugins"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	// creating Command checker
	cfg := map[string]interface{}{
		"command": "/bin/echo HelloWorld",
		"name":    "test-1",
	}
	checker, err := NewCommandChecker(cfg, nil)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Containsf(t, result.Message, "HelloWorld\n", "command bad message: %q", result.Message)

	// bad exit code
	cfg["command"] = "/bin/false"

	checker, err = NewCommandChecker(cfg, nil)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, result.Message, "command /bin/false exited with status code 1")

	// timeout
	cfg["command"] = "/bin/sleep 2"

	checker, err = NewCommandChecker(cfg, nil)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, result.Message, "command /bin/sleep 2 took too long to execute")
}
