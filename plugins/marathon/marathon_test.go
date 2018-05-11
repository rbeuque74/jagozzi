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
	marathonMock := mockApp
	app := *marathonMock.Application
	marathonMock.Application = &app
	// creating Marathon checker
	cfg := map[string]interface{}{
		"type": "minimum_healthy_instances",
		"id":   marathonMock.Application.ID,
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

	assert.Equal(t, "Marathon", checker.Name())
	assert.Equal(t, "test-1", checker.ServiceName())

	marathonChecker := checker.(*MarathonChecker)
	mock := amock.NewMock()
	mock.Expect(200, &marathonMock).OnIdentifier("app").OnFunc(marathonChecker.Run).Sticky()
	marathonChecker.roundtripper.defaultRoundTripper = mock

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result := marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// warning
	marathonMock.Application.TasksHealthy = 1
	marathonMock.Application.TasksRunning = 1

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "1/2 instances running, threshold: 2", result.Message)

	// critical
	marathonMock.Application.TasksHealthy = 0
	marathonMock.Application.TasksRunning = 0

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "0/2 instances running, threshold: 1", result.Message)

	// unhealthy
	marathonMock.Application.TasksHealthy = 0
	marathonMock.Application.TasksRunning = 4
	marathonMock.Application.TasksUnhealthy = 4

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "4 unhealthy; 0/2 healthy instances running", result.Message)

	// tasks
	marathonMock.Application.TasksHealthy = 0
	marathonMock.Application.TasksUnhealthy = 0
	marathonMock.Application.TasksRunning = 2
	marathonMock.Application.TasksStaged = 2
	marathonMock.Application.Tasks[0].StartedAt = ""

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

func TestMarathonLastTaskFailure(t *testing.T) {
	marathonMock := mockApp
	app := *marathonMock.Application
	marathonMock.Application = &app
	marathonMock.Application.Tasks = []*marathonlib.Task{}
	// creating Marathon checker
	cfg := map[string]interface{}{
		"type": "minimum_healthy_instances",
		"id":   marathonMock.Application.ID,
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

	marathonMock.Application.LastTaskFailure = &marathonlib.LastTaskFailure{
		AppID:     "/production/app",
		Message:   " exit code: 2",
		State:     "TASK_FAILED",
		TaskID:    "task1",
		Timestamp: time.Now().UTC().Add(-6 * time.Hour).Format(timeFormat),
		Version:   time.Now().UTC().Add(-7 * time.Hour).Format(timeFormat),
	}

	marathonChecker := checker.(*MarathonChecker)
	mock := amock.NewMock()
	mock.Expect(200, &marathonMock).OnIdentifier("app").OnFunc(marathonChecker.Run).Sticky()
	marathonChecker.roundtripper.defaultRoundTripper = mock

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result := marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// second task failure
	marathonMock.Application.LastTaskFailure.TaskID = "task2"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-5 * time.Hour).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// third task failure
	marathonMock.Application.LastTaskFailure.TaskID = "task3"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-2 * time.Hour).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 1 real
	marathonMock.Application.LastTaskFailure.TaskID = "task4"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-10 * time.Minute).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 2 real
	marathonMock.Application.LastTaskFailure.TaskID = "task5"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-9 * time.Minute).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 3 real
	marathonMock.Application.LastTaskFailure.TaskID = "task6"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-8 * time.Minute).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 4 real
	marathonMock.Application.LastTaskFailure.TaskID = "task7"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-7 * time.Minute).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 4 real same
	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "OK: 2 running; 0 unhealthy; 0 staged", result.Message)

	// task failure - 5
	marathonMock.Application.LastTaskFailure.TaskID = "task8"
	marathonMock.Application.LastTaskFailure.Timestamp = time.Now().UTC().Add(-5 * time.Minute).Format(timeFormat)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()

	result = marathonChecker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_WARNING, result.Status)
	assert.Equal(t, "Last 5 tasks failed (during last 15min):  exit code: 2", result.Message)
}
