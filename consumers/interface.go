package consumers

import (
	"github.com/rbeuque74/jagozzi/plugins"
)

// ResultWithHostname is data-model used by consumer to declare issues to remote
type ResultWithHostname struct {
	plugins.Result
	Hostname string
}

// Consumer is the interface that allow jagozzi to send plugins results
type Consumer interface {
	MessageChannel() chan<- ResultWithHostname
	ExitChannel() chan interface{}
	ErrorChannel() <-chan error
}
