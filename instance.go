package main

import (
	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/consumers"
	"github.com/rbeuque74/jagozzi/consumers/gui"
	"github.com/rbeuque74/jagozzi/consumers/nsca"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

// Jagozzi is an instance of jagozzi checker
type Jagozzi struct {
	cfg       config.Configuration
	checkers  []plugins.Checker
	consumers []consumers.Consumer
}

// Load is loading configuration from file and returns a jagozzi configuration
func Load(cfg config.Configuration) (*Jagozzi, error) {
	y := Jagozzi{
		cfg: cfg,
	}

	// Consumers initialisation
	for _, consumer := range y.cfg.Consumers {
		if consumer.Type != "NSCA" {
			log.Warnf("config: found an unknown consumer type %q", consumer.Type)
			continue
		}

		consumerInstance := nsca.New(consumer)
		y.consumers = append(y.consumers, consumerInstance)
		go ListenForConsumersError(consumerInstance)
	}

	if guiConsumer != nil && *guiConsumer {
		consumer := gui.New()
		y.consumers = append(y.consumers, consumer)
	}

	// Pluggins initialisation
	for _, plugin := range y.cfg.Plugins {
		for _, check := range plugin.Checks {
			checker, err := plugins.CreateChecker(plugin.Type, check, plugin.Config)
			if err != nil && err == plugins.ErrUnknownCheckerType {
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
		close(consumer.ExitChannel())
	}
}

// SendConsumers will send a NSCA message to all consumers
func (y Jagozzi) SendConsumers(result plugins.Result) {
	for _, consumer := range y.consumers {
		consumer.MessageChannel() <- consumers.ResultWithHostname{
			Result:   result,
			Hostname: y.cfg.Hostname,
		}
	}
}

// Checkers returns the list of checkers
func (y Jagozzi) Checkers() []plugins.Checker {
	return y.checkers
}

// ListenForConsumersError will log every consumers errors that fails to be reported to remote notification service
func ListenForConsumersError(c consumers.Consumer) {
	errors := c.ErrorChannel()
	exit := c.ExitChannel()
	for {
		select {
		case err := <-errors:
			if err != nil {
				log.Errorf("consumer: problem while sending to consumer: %s", err)
			} else {
				log.Debug("consumer: message sent!")
			}
		case <-exit:
			log.Debug("consumer: stop listening for errors")
			return
		}
	}
}
