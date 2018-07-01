package processes

import (
	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
)

type rawProcessesConfig struct {
	config.GenericPluginConfiguration
	Command  string `json:"exec" validate:"required"`
	Args     string `json:"args"`
	Type     string `json:"type"`
	Warning  int64  `json:"warn"`
	Critical int64  `json:"crit"`
}

type processesConfig struct {
	rawProcessesConfig
}

func (cfg *processesConfig) UnmarshalJSON(b []byte) error {
	raw := &rawProcessesConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(raw); err != nil {
		return err
	}

	cfg.rawProcessesConfig = *raw
	return nil
}
