package nsca

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/consumers"
	"github.com/rbeuque74/jagozzi/plugins"
	"github.com/rbeuque74/nsca"
	nsrv "github.com/tubemogul/nscatools"
)

type fakeChecker struct {
	t *testing.T
}

func (fc fakeChecker) Name() string {
	return "fake-checker"
}

func (fc fakeChecker) ServiceName() string {
	return "fake-service-name"
}

func (fc fakeChecker) Periodicity() *time.Duration {
	return nil
}

func (fc fakeChecker) Run(ctx context.Context) plugins.Result {
	fc.t.Fatal("fake checker should not run")
	return plugins.Result{
		Status:  plugins.STATE_CRITICAL,
		Message: "fake checker should never run",
		Checker: fc,
	}
}

func TestConsumerSendMessage(t *testing.T) {
	// creating NSCA server
	srvCh := make(chan nsrv.DataPacket, 10)
	go NewNscaServerChannel(srvCh)

	// sleeping a bit to let NSCA server start
	time.Sleep(20 * time.Millisecond)

	// creating NSCA client
	cfg := &config.ConsumerConfiguration{}
	cfgStr := []byte(fmt.Sprintf(`{"type":"NSCA","server":"localhost","timeout":1000,"encryption":%d,"key":"%s"}`, nsca.ENCRYPT_XOR, EncryptKey))
	json.Unmarshal(cfgStr, cfg)

	consumer := New(*cfg)

	var messages []string

	var message = "example message"
	res := plugins.Result{
		Status:  plugins.STATE_CRITICAL,
		Message: message,
		Checker: fakeChecker{
			t: t,
		},
	}
	messages = append(messages, message)

	consumer.MessageChannel() <- consumers.ResultWithHostname{
		Result:   res,
		Hostname: "hostname-example-1",
	}

	message = "message with unallowed characters, \"multiple ,characters\""
	res = plugins.Result{
		Status:  plugins.STATE_CRITICAL,
		Message: message,
		Checker: fakeChecker{
			t: t,
		},
	}
	messages = append(messages, "message with unallowed characters multiple characters")

	consumer.MessageChannel() <- consumers.ResultWithHostname{
		Result:   res,
		Hostname: "hostname-example-1",
	}

	messageReceived := 0
	for {
		select {
		case err := <-consumer.ErrorChannel():
			if err != nil {
				t.Fatalf("err channel is not empty: %q", err)
			} else {
				t.Log("nsca send OK")
			}
			messageReceived += 1

			msg := <-srvCh
			if msg.PluginOutput != messages[0] {
				t.Fatalf("message received incorrect: %s", msg.PluginOutput)
				return
			}

			if messageReceived == 2 {
				close(consumer.ExitChannel())
			} else {
				messages = messages[1:]
			}
		case <-time.After(time.Second):
			t.Log("timed out")
			if messageReceived != 2 {
				t.Fatal("timeout and message not received")
			}
			return
		case <-consumer.ExitChannel():
			t.Log("finished")
			return
		}
	}

}
