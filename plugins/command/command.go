package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/ghodss/yaml"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

const pluginName = "Command"

func init() {
	plugins.Register(pluginName, NewCommandChecker)
}

// CommandChecker is a plugin to check status code of command
type CommandChecker struct {
	cfg     commandConfig
	command string
	args    []string
}

// Name returns the name of the checker
func (c CommandChecker) Name() string {
	return pluginName
}

// ServiceName returns the name of the NSCA service associated to the checker
func (c CommandChecker) ServiceName() string {
	return c.cfg.Name
}

type result struct {
	Cfg      commandConfig
	Cmd      exec.Cmd
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

func (res result) Error() error {
	return res.Err
}

// Run is performing the checker protocol
func (c *CommandChecker) Run(ctx context.Context) plugins.Result {
	cmd := exec.Command(c.command, c.args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	model := result{
		Cfg:    c.cfg,
		Cmd:    *cmd,
		Stdout: "",
		Stderr: "",
		Err:    nil,
	}

	if err := cmd.Start(); err != nil {
		return plugins.ResultFromError(c, err, "")
	}
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	select {
	case <-ctx.Done():
		model.Stderr = stderr.String()
		model.Stdout = stdout.String()

		if err := cmd.Process.Kill(); err != nil {
			return plugins.ResultFromError(c, err, fmt.Sprintf("command: context expired, kill pid %q failed", cmd.Process.Pid))
		}

		model.Err = ctx.Err()
		err := fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrTimeout, model))
		return plugins.ResultFromError(c, err, "")
	case err := <-done:
		model.Stderr = stderr.String()
		model.Stdout = stdout.String()

		if typedErr, ok := err.(*exec.ExitError); ok {
			model.ExitCode = typedErr.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			model.Err = fmt.Errorf("%s: %s", typedErr, model.Stderr)
			err = fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrExitCode, model))
			return plugins.ResultFromError(c, err, "")
		} else if err != nil {
			return plugins.ResultFromError(c, fmt.Errorf("%s: %s", err, model.Stderr), "")
		} else {
			return plugins.Result{
				Status:  plugins.STATE_OK,
				Message: model.Stdout,
				Checker: c,
			}
		}
	}
}

// NewCommandChecker create a Command checker
func NewCommandChecker(conf interface{}, pluginConf interface{}) (plugins.Checker, error) {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}

	cfg := commandConfig{}
	err = yaml.Unmarshal(out, &cfg)
	if err != nil {
		return nil, err
	}

	p := shellwords.NewParser()
	p.ParseEnv = true
	args, err := p.Parse(cfg.Command)
	if err != nil {
		return nil, err
	}

	checker := &CommandChecker{
		cfg: cfg,
	}

	first := true
	for _, arg := range args {
		if first {
			checker.command = arg
			first = false
		} else {
			checker.args = append(checker.args, arg)
		}
	}

	log.Infof("command: Checker activated for watching %q", checker.cfg.Command)
	return checker, nil
}
