package supervisor

import (
	"fmt"
	"net/url"
	"time"

	"github.com/ghodss/yaml"
	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
	defaults "gopkg.in/mcuadros/go-defaults.v1"
)

type checkerConfig struct {
	rawCheckerConfig
}

type rawCheckerConfig struct {
	Type    string  `json:"type" validate:"required,eq=service|eq=services"`
	Name    string  `json:"name" validate:"required"`
	Service *string `json:"service"`
}

type pluginConfig struct {
	rawPluginConfig
	Timeout   time.Duration
	ServerURL url.URL
}

type rawPluginConfig struct {
	RawServerURL string `json:"serverurl" default:"unix:///var/run/supervisor.sock"`
	RPCNamespace string `json:"rpc_namespace" default:"supervisor"`
	RawTimeout   *int64 `json:"timeout" default:"5000"`
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

	if cfg.RawTimeout != nil {
		cfg.Timeout = time.Duration(*cfg.RawTimeout) * time.Millisecond
	}

	defaults.SetDefaults(&cfg)
	defaults.SetDefaults(&cfg.rawPluginConfig)

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return cfg, err
	}

	serverURL, err := url.Parse(cfg.RawServerURL)
	if err != nil {
		return cfg, err
	}

	cfg.ServerURL = *serverURL

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

	if cfg.Service != nil && cfg.Type == "services" {
		return cfg, fmt.Errorf("type 'servicess' and service key are incompatible")
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
