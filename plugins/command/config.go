package command

import (
	"bytes"
	"os/exec"
	"text/template"

	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
)

type rawCommandConfig struct {
	Command  string `json:"command" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Template string `json:"template"`
}

type commandConfig struct {
	rawCommandConfig
	template *template.Template
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

	cfg.rawCommandConfig = *raw

	if cfg.Template != "" {
		// testing that we can parse template
		tmpl, err := template.New("commandTemplate").Parse(cfg.Template)
		if err != nil {
			return err
		}

		model := result{
			Cfg:    *cfg,
			Cmd:    exec.Cmd{},
			Stdout: "",
			Stderr: "",
		}

		// testing that we can apply template to model
		buf := new(bytes.Buffer)
		err = tmpl.Execute(buf, model)
		if err != nil {
			return err
		}

		cfg.template = tmpl
	}

	return nil
}
