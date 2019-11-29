package config

import (
	"strconv"
	"strings"
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
	Instances  int64  `json:"instances"`
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

// GenericPluginConfiguration is a generic plugin configuration
type GenericPluginConfiguration struct {
	Name            string        `json:"name" validate:"required"`
	PeriodicityJSON *jsonDuration `json:"periodicity"`
}

type jsonDuration time.Duration

// Periodicity returns the proper Periodicity as a time.Duration
func (c *GenericPluginConfiguration) Periodicity() *time.Duration {
	if c.PeriodicityJSON == nil {
		return nil
	}

	dur := time.Duration(*c.PeriodicityJSON)
	return &dur
}

func (d *jsonDuration) UnmarshalJSON(b []byte) error {
	str := string(b)
	durationInt, err := strconv.ParseUint(str, 10, 64)
	var duration time.Duration
	if err == nil {
		duration = time.Duration(durationInt) * time.Second
		*d = (jsonDuration)(duration)
		return nil
	}

	str = strings.Replace(str, "\"", "", 2)
	duration, err = time.ParseDuration(str)
	if err != nil {
		return err
	}

	*d = (jsonDuration)(duration)
	return nil
}
