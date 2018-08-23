package nsca

import (
	"errors"

	nsrv "github.com/tubemogul/nscatools"
)

const (
	// EncryptKey is the NSCA encryption key for unit tests
	EncryptKey = "toto"
)

func NewNscaServerChannel(ch chan<- nsrv.DataPacket) {
	cfg := nsrv.NewConfig("localhost", 5667, nsrv.EncryptXOR, EncryptKey, func(p *nsrv.DataPacket) error {
		if p == nil {
			return errors.New("packet is nil")
		}
		ch <- *p
		return nil
	})
	nsrv.StartServer(cfg, true)
}
