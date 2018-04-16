package command

import (
	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
)

type rawCommandConfig struct {
	Command string `json:"command" validate:"required"`
	Type    string `json:"type" validate:"required"`
	Name    string `json:"name" validate:"required"`
}

type commandConfig struct {
	rawCommandConfig
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
	return nil
}
