package nsca

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/plugins"
	"github.com/syncbak-git/nsca"
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
	srvCh := make(chan Message, 10)
	go NewNscaServerChannel(srvCh)

	// sleeping a bit to let NSCA server start
	time.Sleep(20 * time.Millisecond)

	// creating NSCA client
	cfg := &config.ConsumerConfiguration{}
	cfgStr := []byte(fmt.Sprintf(`{"type":"NSCA","server":"localhost","timeout":1000,"encryption":%d,"key":"%s"}`, nsca.ENCRYPT_XOR, EncryptKey))
	json.Unmarshal(cfgStr, cfg)

	msgCh := make(chan *nsca.Message, 10)
	exitCh := make(chan interface{}, 10)
	consumer := New(*cfg, msgCh, exitCh)

	// sending message
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	res := plugins.Result{
		Status:  plugins.STATE_CRITICAL,
		Message: "example message",
		Checker: fakeChecker{
			t: t,
		},
	}

	errCh := make(chan error, 10)
	consumer.Send(ctx, res, "hostname-example-1", errCh)

	messageReceived := false
	for {
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("err channel is not empty: %q", err)
			} else {
				t.Log("nsca send OK")
			}
			messageReceived = true
			consumer.Unload()
		case <-time.After(time.Second):
			t.Log("timed out")
			if !messageReceived {
				t.Fatal("timeout and message not received")
			}
			return
		case <-exitCh:
			t.Log("finished")
			return
		}
	}

}
