package config

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
)

// Load is loading configuration from file and returns a jagozzi configuration
func Load(configurationFile string) (*Configuration, error) {
	stream, err := ioutil.ReadFile(configurationFile)
	if err != nil {
		return nil, err
	}

	cfg := &Configuration{}
	if err = yaml.Unmarshal(stream, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
