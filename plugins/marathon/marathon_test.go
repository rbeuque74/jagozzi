package marathon

import (
	"context"
	"testing"
	"time"

	marathonlib "github.com/gambol99/go-marathon"
	"github.com/loopfz/gadgeto/amock"
	"github.com/rbeuque74/jagozzi/plugins"
	"github.com/stretchr/testify/assert"
)

type app struct {
	Application *marathonlib.Application `json:"app"`
}

func intPtr(i int) *int {
	return &i
}

var mockApp = app{
	Application: &marathonlib.Application{
		ID:             "/production/app",
		TasksHealthy:   2,
		TasksRunning:   2,
		TasksStaged:    0,
		TasksUnhealthy: 0,
		Instances:      intPtr(2),
		Tasks: []*marathonlib.Task{
			&marathonlib.Task{
				ID:        "app1",
				AppID:     "/production/app",
				StagedAt:  time.Now().UTC().Add(-3 * time.Hour).Format(timeFormat),
				StartedAt: time.Now().UTC().Add(-3 * time.Hour).Format(timeFormat),
			},
			&marathonlib.Task{
				ID:        "app2",
				AppID:     "/production/app",
				StagedAt:  time.Now().UTC().Add(-4 * time.Hour).Format(timeFormat),
				StartedAt: time.Now().UTC().Add(-4 * time.Hour).Format(timeFormat),
			},
		},
	},
}

func TestMarathon(t *testing.T) {
	// creating Marathon checker
	cfg := map[string]interface{}{
		"type": "minimum_healthy_instances",
		"id":   mockApp.Application.ID,
		"warn": 2,
		"crit": 1,
		"name": "test-1",
	}
	pluginCfg := map[string]interface{}{
		"host":     "http://example.com",
		"user":     "marathon",
		"password": "password",
	}
	checker, err := NewMarathonChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "marathon checker instantiation failed: %q", err)
	marathonChecker := checker.(*MarathonChecker)
	mock := amock.NewMock()
	mock.Expect(200, &mockApp).OnIdentifier("app").OnFunc(marathonChecker.Run).Sticky()
	marathonChecker.roundtripper.defaultRoundTripper = mock

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result := marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// warning
	mockApp.Application.TasksHealthy = 1
	mockApp.Application.TasksRunning = 1

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "1/2 instances running, threshold: 2", result.Message)

	// critical
	mockApp.Application.TasksHealthy = 0
	mockApp.Application.TasksRunning = 0

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "0/2 instances running, threshold: 1", result.Message)

	// unhealthy
	mockApp.Application.TasksHealthy = 0
	mockApp.Application.TasksRunning = 4
	mockApp.Application.TasksUnhealthy = 4

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "4 unhealthy; 0/2 healthy instances running", result.Message)

	// tasks
	mockApp.Application.TasksHealthy = 0
	mockApp.Application.TasksUnhealthy = 0
	mockApp.Application.TasksRunning = 2
	mockApp.Application.TasksStaged = 2
	mockApp.Application.Tasks[0].StartedAt = ""

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "task stagged since 15 minutes", result.Message)

	// not found app
	cfg["id"] = "/production/unknown-app"
	checker, err = NewMarathonChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "marathon checker instantiation failed: %q", err)
	marathonChecker = checker.(*MarathonChecker)
	mock = amock.NewMock()
	mock.Expect(404, map[string]string{"message": "application not found"}).OnIdentifier("unknown-app").OnFunc(marathonChecker.Run)
	marathonChecker.roundtripper.defaultRoundTripper = mock

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "Marathon API error: application not found", result.Message)
}
