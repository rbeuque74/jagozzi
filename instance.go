package main

import (
	"context"
	"fmt"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/consumers/nsca"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
	nscalib "github.com/syncbak-git/nsca"
)

// Jagozzi is an instance of jagozzi checker
type Jagozzi struct {
	cfg                  config.Configuration
	checkers             []plugins.Checker
	consumers            []nsca.Consumer
	consumerErrorChannel chan error
}

// Load is loading configuration from file and returns a jagozzi configuration
func Load(cfg config.Configuration) (*Jagozzi, error) {
	y := Jagozzi{
		cfg:                  cfg,
		consumerErrorChannel: make(chan error),
	}

	// Consumers initialisation
	for _, consumer := range y.cfg.Consumers {
		if consumer.Type != "NSCA" {
			log.Warnf("config: found an unknown consumer type %q", consumer.Type)
			continue
		}
		if consumer.Port == 0 {
			// default nsca port
			consumer.Port = 5667
		}
		serv := nscalib.ServerInfo{
			Host:             consumer.Server,
			Port:             fmt.Sprintf("%d", consumer.Port),
			EncryptionMethod: int(consumer.Encryption),
			Password:         consumer.Key,
			Timeout:          consumer.Timeout,
		}

		exitChannel := make(chan interface{})
		messagesChannel := make(chan *nscalib.Message)

		log.Infof("consumer: starting NSCA server to %s:%d", consumer.Server, consumer.Port)
		go nscalib.RunEndpoint(serv, exitChannel, messagesChannel)

		consumerInstance := nsca.New(messagesChannel, exitChannel)
		y.consumers = append(y.consumers, consumerInstance)
	}

	// Pluggins initialisation
	for _, plugin := range y.cfg.Plugins {
		for _, check := range plugin.Checks {
			checker, err := plugins.CreateChecker(plugin.Type, check, plugin.Config)
			if err != nil && err == plugins.UnknownCheckerTypeErr {
				log.WithField("type", plugin.Type).Warn(err)
				continue
			} else if err != nil {
				return nil, err
			}
			y.checkers = append(y.checkers, checker)
		}
	}

	return &y, nil
}

// Unload cleans all current operation/goroutine loaded by configuration and configuration childs
func (y Jagozzi) Unload() {
	for _, consumer := range y.consumers {
		consumer.Unload()
	}
}

// SendConsumers will send a NSCA message to all consumers
func (y Jagozzi) SendConsumers(ctx context.Context, result plugins.Result) {
	for _, consumer := range y.consumers {
		consumer.Send(ctx, result, y.cfg.Hostname, y.consumerErrorChannel)
	}
}

// Checkers returns the list of checkers
func (y Jagozzi) Checkers() []plugins.Checker {
	return y.checkers
}
