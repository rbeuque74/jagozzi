package gui

import (
	"fmt"
	"time"

	tm "github.com/buger/goterm"
	"github.com/rbeuque74/jagozzi/consumers"
	"github.com/rbeuque74/jagozzi/plugins"
)

// Consumer is a GUI consumer
type Consumer struct {
	messages    chan consumers.ResultWithHostname
	exit        chan interface{}
	error       chan error
	savedStatus map[string]consumers.ResultWithHostname
}

// New creates a new GUI consumer
func New() Consumer {
	c := Consumer{
		messages:    make(chan consumers.ResultWithHostname, 10),
		exit:        make(chan interface{}),
		error:       make(chan error),
		savedStatus: make(map[string]consumers.ResultWithHostname),
	}

	go c.handle()
	tm.Clear() // Clear current screen
	return c
}

// MessageChannel is the channel to be use to push messages to display terminal
func (c Consumer) MessageChannel() chan<- consumers.ResultWithHostname {
	return c.messages
}

// ExitChannel is the channel we need to close in order to shutdown GUI
func (c Consumer) ExitChannel() chan interface{} {
	return c.exit
}

// ErrorChannel is the channel that returns errors when sending a message
func (c Consumer) ErrorChannel() <-chan error {
	return c.error
}

func generateMapKey(msg consumers.ResultWithHostname) string {
	return fmt.Sprintf("%s#%s", msg.Hostname, msg.Checker.ServiceName())
}

func (c Consumer) handle() {
	for {
		select {
		case <-c.exit:
			return
		case msg := <-c.messages:
			c.savedStatus[generateMapKey(msg)] = msg
			draw(c.savedStatus)
		}
	}
}

func stateToString(state plugins.StatusEnum) string {
	switch state {
	case plugins.STATE_OK:
		return tm.Background(tm.Color(fmt.Sprintf("  %-4s", "OK"), tm.BLACK), tm.GREEN) + tm.RESET
	case plugins.STATE_WARNING:
		return tm.Background(tm.Color(fmt.Sprintf(" %-5s", "WARN"), tm.BLACK), tm.YELLOW) + tm.RESET
	case plugins.STATE_CRITICAL:
		return tm.Background(fmt.Sprintf(" %-5s", "CRIT"), tm.RED) + tm.RESET
	default:
		return fmt.Sprintf(" %-5s", "UNKN")
	}
}

var maxLenSeen = 0
var clearLine = ""

func draw(msgs map[string]consumers.ResultWithHostname) {
	// By moving cursor to top-left position we ensure that console output
	// will be overwritten each time, instead of adding new.
	tm.MoveCursor(1, 1)
	nbMsgs := len(msgs)
	eof := "\n"
	i := 0
	for _ = range msgs {
		tm.Println(clearLine)
	}
	tm.MoveCursor(1, 1)
	for _, res := range msgs {
		i++
		if i == nbMsgs {
			eof = ""
		}
		l, _ := tm.Printf("%-35s %-30s %-30s %s %s", time.Now().Truncate(time.Second), res.Hostname, res.Checker.ServiceName(), stateToString(res.Status), res.Message)

		if eof != "" {
			tm.Print(eof)
		}

		if maxLenSeen < l {
			maxLenSeen = l
			clearLine = ""
			for j := 0; j < l; j++ {
				clearLine = clearLine + " "
			}
		}
	}
	tm.Flush() // Call it every time at the end of rendering
}
