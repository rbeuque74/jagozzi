package nsca

import (
	"context"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
	"github.com/syncbak-git/nsca"
)

// Consumer is the representation of a NSCA consumer
type Consumer struct {
	cfg            config.ConsumerConfiguration
	messageChannel chan *nsca.Message
	exitChannel    chan interface{}
}

// New generates a new NSCA Consumer instance
func New(messageChannel chan *nsca.Message, exitChannel chan interface{}) Consumer {
	return Consumer{
		messageChannel: messageChannel,
		exitChannel:    exitChannel,
	}
}

// Send will produce a message to the specified consumer
func (consumer Consumer) Send(ctx context.Context, result plugins.Result, hostname string, errorChannel chan error) {
	if consumer.messageChannel == nil {
		log.Warnf("consumer: message channel is empty")
		return
	}

	msg := nsca.Message{
		State:   int16(result.Status),
		Host:    hostname,
		Service: result.Checker.ServiceName(),
		Message: result.Message,
		Status:  errorChannel,
	}

	log.Debugf("consumer: send message %+v", msg)

	ch := consumer.messageChannel
	select {
	case ch <- &msg:
		return
	case <-ctx.Done():
		log.Warnf("consumer: timeout to push message to consumer message channel: %s", ctx.Err())
		return
	}
}

// Unload cleans all current operation/goroutine of consumer
func (consumer Consumer) Unload() {
	if consumer.exitChannel == nil {
		return
	}

	log.Debugf("consumer: sent 'quit' information to receiver")
	consumer.exitChannel <- true
}
