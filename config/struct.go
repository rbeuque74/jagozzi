package config

import (
	"time"
)

// Configuration is the jagozzi main configuration structure
type Configuration struct {
	rawConfiguration
	// Periodicity is time span between two iterations of a check
	Periodicity time.Duration `json:"-"`
}

type rawConfiguration struct {
	RawPeriodicity int64                   `json:"periodicity"`
	Hostname       string                  `json:"hostname"`
	Consumers      []ConsumerConfiguration `json:"consumers"`
	Plugins        []PluginConfiguration   `json:"plugins"`
}

// UnmarshalJSON explicits some variables from configuration file to proper Golang type
func (cfg *Configuration) UnmarshalJSON(b []byte) error {
	raw := &rawConfiguration{}

	if err := UnmarshalConfig(b, raw); err != nil {
		return err
	}

	cfg.rawConfiguration = *raw
	cfg.Periodicity = time.Duration(raw.RawPeriodicity) * time.Second

	return nil
}

// ConsumerConfiguration is the configuration of a consumer
type ConsumerConfiguration struct {
	rawConsumerConfiguration
	Timeout time.Duration `json:"-"`
}

type rawConsumerConfiguration struct {
	Type       string `json:"type"`
	Server     string `json:"server"`
	Port       int64  `json:"port"`
	RawTimeout int64  `json:"timeout"`
	Encryption int64  `json:"encryption"`
	Key        string `json:"key"`
}

// UnmarshalJSON explicits some variables from configuration file to proper Golang type
func (cfg *ConsumerConfiguration) UnmarshalJSON(b []byte) error {
	raw := &rawConsumerConfiguration{}

	if err := UnmarshalConfig(b, raw); err != nil {
		return err
	}

	cfg.rawConsumerConfiguration = *raw
	cfg.Timeout = time.Duration(raw.RawTimeout) * time.Millisecond

	return nil
}

// PluginConfiguration represents the configuration of a plugin
type PluginConfiguration struct {
	// Type is the name of the plugin that will run
	Type string `json:"type"`
	// Config is the custom configuration of the plugin
	Config interface{} `json:"config,omitempty"`
	// Checks is the list of all checks that plugin will run
	Checks []interface{} `json:"checks"`
}
