package supervisor

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/rbeuque74/jagozzi/plugins"
	"github.com/stretchr/testify/assert"
)

const (
	supervisordConfig = `[unix_http_server]
file=/tmp/supervisord.sock

[supervisord]
pidfile=/tmp/supervisord.pid
identifier=supervisor
logfile = /tmp/supervisord.log
nodaemon = true
directory = /tmp
nocleanup = false
childlogdir = /tmp

[supervisorctl]
serverurl=unix:///tmp/supervisord.sock

[program:app]
command=/bin/sleep 60
autorestart=true
exitcodes=0,2
stopsignal=TERM
stopwaitsecs=10
directory=/tmp
`
	supervisordFaultyConfig = `[unix_http_server]
file=/tmp/supervisord.sock

[supervisord]
pidfile=/tmp/supervisord.pid
identifier=supervisor
logfile = /tmp/supervisord.log
nodaemon = true
directory = /tmp
nocleanup = false
childlogdir = /tmp

[supervisorctl]
serverurl=unix:///tmp/supervisord.sock

[program:app]
command=/bin/false
autorestart=false
exitcodes=10
stopsignal=TERM
stopwaitsecs=10
directory=/tmp
`
)

var pids []*os.Process
var tmpfile *os.File
var mutex sync.Once

func start(t *testing.T, faulty bool) {
	var err error
	mutex.Do(func() {
		cmd := exec.Command("go", "get", "-v", "github.com/ochinchina/supervisord")
		if err = cmd.Run(); err != nil {
			t.Error(err)
			t.FailNow()
		}
	})

	content := []byte(supervisordConfig)
	if faulty {
		content = []byte(supervisordFaultyConfig)
	}
	tmpfile, err = ioutil.TempFile("", "supervisord-config")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := tmpfile.Write(content); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err := tmpfile.Close(); err != nil {
		t.Error(err)
		t.FailNow()
	}

	cmd := exec.Command("supervisord", "-c", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Error(err)
		t.FailNow()
	}

	pids = append(pids, cmd.Process)
}

func stop(t *testing.T) {
	for _, process := range pids {
		if process == nil {
			t.Error("process pid is nil")
			t.FailNow()
		}
		if err := process.Kill(); err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	if tmpfile == nil {
		return
	}

	os.Remove(tmpfile.Name())
	os.Remove("/tmp/supervisord.sock")
	os.Remove("/tmp/supervisord.log")
	os.Remove("/tmp/supervisord.log.0")
	os.Remove("/tmp/supervisord.pid")
}

func TestSupervisor(t *testing.T) {
	// creating supervisor server
	start(t, false)
	defer stop(t)

	time.Sleep(200 * time.Millisecond)

	// creating Command checker
	cfg := map[string]interface{}{
		"type": "services",
		"name": "test-1",
	}
	pluginCfg := map[string]interface{}{
		"serverurl": "unix:///tmp/supervisord.sock",
	}

	checker, err := NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	assert.Equal(t, "Supervisor", checker.Name())
	assert.Equal(t, "test-1", checker.ServiceName())

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Equal(t, "All services are running", result.Message)

	// service mode
	cfg["type"] = "service"
	cfg["service"] = "app"

	checker, err = NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_OK, result.Status)
	assert.Contains(t, result.Message, `Service "app" is running: pid `)

	// service not found
	cfg["type"] = "service"
	cfg["service"] = "not-found"

	checker, err = NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, `Service "not-found" not found`, result.Message)
}

func TestSupervisorFaulty(t *testing.T) {
	// creating supervisor server
	start(t, true)
	defer stop(t)

	// sleeping 2 second to let our supervisor process failing
	time.Sleep(2 * time.Second)

	// creating Command checker
	cfg := map[string]interface{}{
		"type": "services",
		"name": "test-1",
	}
	pluginCfg := map[string]interface{}{
		"serverurl": "unix:///tmp/supervisord.sock",
	}

	checker, err := NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Contains(t, result.Message, `Service "app" is currently EXITED:`)

	// service mode
	cfg["type"] = "service"
	cfg["service"] = "app"

	checker, err = NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Contains(t, result.Message, `Service "app" is currently EXITED:`)
}

func TestSupervisorNotRunning(t *testing.T) {
	// creating Command checker
	cfg := map[string]interface{}{
		"type": "services",
		"name": "test-1",
	}
	pluginCfg := map[string]interface{}{
		"serverurl": "unix:///tmp/supervisord.sock",
		"timeout":   100,
	}

	checker, err := NewSupervisorChecker(cfg, pluginCfg)
	assert.Nilf(t, err, "command checker instantiation failed: %q", err)

	ctxRun, cancelFunc1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result := checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "unable to contact supervisor daemon: dial unix /tmp/supervisord.sock: connect: no such file or directory", result.Message)

	l, err := net.Listen("unix", "/tmp/supervisord.sock")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer l.Close()

	ctxRun, cancelFunc1 = context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc1()
	result = checker.Run(ctxRun)
	assert.Equal(t, plugins.STATE_CRITICAL, result.Status)
	assert.Equal(t, "unable to contact supervisor daemon: read unix @->/tmp/supervisord.sock: i/o timeout", result.Message)
}
