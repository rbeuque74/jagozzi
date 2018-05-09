package command

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"text/template"

	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
	defaults "gopkg.in/mcuadros/go-defaults.v1"
)

type rawCommandConfig struct {
	Command      string       `json:"command" validate:"required"`
	Type         string       `json:"type"`
	Name         string       `json:"name" validate:"required"`
	RawTemplates rawTemplates `json:"templates"`
}

type commandConfig struct {
	rawCommandConfig
	templates templates
}

type rawTemplates struct {
	ErrTimeout  string `default:"command {{.Cfg.Command}} took too long to execute"`
	ErrExitCode string `default:"command {{.Cfg.Command}} exited with status code {{.ExitCode}}"`
}

type templates struct {
	ErrTimeout  *template.Template `json:"-"`
	ErrExitCode *template.Template `json:"-"`
}

func (cfg *commandConfig) UnmarshalJSON(b []byte) error {
	raw := &rawCommandConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(raw); err != nil {
		return err
	}

	defaults.SetDefaults(raw)

	cfg.rawCommandConfig = *raw

	var err error
	var tmpl *template.Template
	if tmpl, err = testTemplate("CommandErrTimeout", raw.RawTemplates.ErrTimeout); err != nil {
		return err
	}
	cfg.templates.ErrTimeout = tmpl

	if tmpl, err = testTemplate("CommandErrExitCode", raw.RawTemplates.ErrExitCode); err != nil {
		return err
	}
	cfg.templates.ErrExitCode = tmpl

	return nil
}

func testTemplate(templateName, stringTemplate string) (*template.Template, error) {
	// testing that we can parse template
	tmpl, err := template.New(templateName).Parse(stringTemplate)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("/bin/echo", "arg1", "arg2")
	cmd.Process = &os.Process{
		Pid: 1234,
	}
	model := result{
		Cfg:    commandConfig{},
		Cmd:    *cmd,
		Stdout: "stdout",
		Stderr: "error: stderr",
		Err:    errors.New("standard error"),
	}

	// testing that we can apply template to model
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, model)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
