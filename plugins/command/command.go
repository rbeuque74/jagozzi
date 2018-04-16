package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

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

func (c *CommandChecker) Name() string {
	return pluginName
}

func (c CommandChecker) ServiceName() string {
	return c.cfg.Name
}

func (c *CommandChecker) Run(ctx context.Context) (string, error) {
	cmd := exec.Command(c.command, c.args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		log.Warn("command: can't start")
		return "KO", err
	}
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			return "KO", fmt.Errorf("command: context expired, kill pid %q failed: %s", cmd.Process.Pid, err)
		}
		return "KO", errors.New("command: context finished before command finished execution")
	case err := <-done:
		if typedErr, ok := err.(*exec.ExitError); ok {
			return "KO", fmt.Errorf("%s: %s", typedErr, string(stderr.String()))
		} else if err != nil {
			return "KO", fmt.Errorf("%s: %s", err, string(stderr.String()))
		} else {
			return string(stdout.String()), nil
		}
	}
}

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
