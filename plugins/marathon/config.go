package marathon

import (
	"errors"

	"github.com/ghodss/yaml"
	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
)

type checkerConfig struct {
	rawCheckerConfig
}

type rawCheckerConfig struct {
	Type     string `json:"type" validate:"required"`
	ID       string `json:"id" validate:"required"`
	Warning  int64  `json:"warn"`
	Critical int64  `json:"crit"`
	Name     string `json:"name" validate:"required"`
}

type pluginConfig struct {
	rawPluginConfig
}

type rawPluginConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host" validate:"required"`
}

func (cfg checkerConfig) ServiceName() string {
	return cfg.Name
}

func loadPluginConfiguration(conf interface{}) (pluginConfig, error) {
	cfg := pluginConfig{}

	out, err := yaml.Marshal(conf)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(out, &cfg)
	if err != nil {
		return cfg, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return cfg, err
	}

	if cfg.Host == "" {
		return cfg, errors.New("host is empty")
	}

	return cfg, nil
}

func loadConfiguration(conf interface{}) (checkerConfig, error) {
	cfg := checkerConfig{}

	out, err := yaml.Marshal(conf)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(out, &cfg)
	if err != nil {
		return cfg, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (cfg *checkerConfig) UnmarshalJSON(b []byte) error {
	raw := &rawCheckerConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	cfg.rawCheckerConfig = *raw
	return nil
}

func (cfg *pluginConfig) UnmarshalJSON(b []byte) error {
	raw := &rawPluginConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	cfg.rawPluginConfig = *raw
	return nil
}
