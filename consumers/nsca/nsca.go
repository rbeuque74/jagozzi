package nsca

import (
	"fmt"
	"strings"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/consumers"
	"github.com/rbeuque74/nsca"
	log "github.com/sirupsen/logrus"
)

var replacer = strings.NewReplacer(",", "", "\"", "")

// Consumer is the representation of a NSCA consumer
type Consumer struct {
	cfg      config.ConsumerConfiguration
	nscaMsg  chan *nsca.Message
	exit     chan interface{}
	messages chan consumers.ResultWithHostname
	error    chan error
}

// New generates a new NSCA Consumer instance
func New(cfg config.ConsumerConfiguration) Consumer {
	if cfg.Port == 0 {
		// default nsca port
		cfg.Port = 5667
	}

	nscaMessageChannel := make(chan *nsca.Message, 10)
	errorChannel := make(chan error)
	messagesChannel := make(chan consumers.ResultWithHostname, 100)
	exitChannel := make(chan interface{})

	serv := nsca.ServerInfo{
		Host:             cfg.Server,
		Port:             fmt.Sprintf("%d", cfg.Port),
		EncryptionMethod: int(cfg.Encryption),
		Password:         cfg.Key,
		Timeout:          cfg.Timeout,
		NoKeepAlive:      true,
	}

	log.Infof("consumer: starting %d NSCA server to %s:%d", cfg.Instances, cfg.Server, cfg.Port)
	for i := int64(0); i < cfg.Instances; i++ {
		go nsca.RunEndpoint(serv, exitChannel, nscaMessageChannel)
	}

	consumer := Consumer{
		messages: messagesChannel,
		error:    errorChannel,
		exit:     exitChannel,
		nscaMsg:  nscaMessageChannel,
	}
	go consumer.handle()
	return consumer
}

// MessageChannel is the channel to be use to push messages to remote provider
func (consumer Consumer) MessageChannel() chan<- consumers.ResultWithHostname {
	return consumer.messages
}

// ExitChannel is the channel we need to close in order to shutdown NSCA server
func (consumer Consumer) ExitChannel() chan interface{} {
	return consumer.exit
}

// ErrorChannel is the channel that returns errors when sending a message
func (consumer Consumer) ErrorChannel() <-chan error {
	return consumer.error
}

func (consumer Consumer) handle() {
	for {
		select {
		case <-consumer.exit:
			return
		case result := <-consumer.messages:
			msg := &nsca.Message{
				State:   int16(result.Status),
				Host:    result.Hostname,
				Service: result.Checker.ServiceName(),
				Message: replacer.Replace(result.Message),
				Status:  consumer.error,
			}
			log.Debugf("consumer: send message %+v", *msg)

			afterTwoSecs := time.NewTimer(2 * time.Second)
			select {
			case consumer.nscaMsg <- msg:
				afterTwoSecs.Stop()
			case <-afterTwoSecs.C:
				log.Warnf("consumer: timeout to push message to consumer message channel")
			case <-consumer.exit:
				afterTwoSecs.Stop()
				return
			}
		}
	}
}
