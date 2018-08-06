package u2flock

import (
	"context"
	"crypto/rand"
	"io"
	"os/exec"
	"time"

	"github.com/flynn/hid"

	"github.com/flynn/u2f/u2fhid"
	"github.com/flynn/u2f/u2ftoken"
	"github.com/seibert-media/golibs/log"
	"go.uber.org/zap"
)

// Lock when a known token gets disconnected
func (k *KeyFile) Lock(cmd string, args ...string) func(ctx context.Context, d *hid.DeviceInfo) error {
	return func(ctx context.Context, d *hid.DeviceInfo) error {
		challenge := make([]byte, 32)
		for _, token := range k.Tokens {
			// check device

			dev, err := u2fhid.Open(d)
			if err != nil {
				log.From(ctx).Error("opening device")
				continue
			}
			t := u2ftoken.NewToken(dev)

			io.ReadFull(rand.Reader, challenge)

			req := u2ftoken.AuthenticateRequest{
				Challenge:   challenge,
				Application: app,
				KeyHandle:   token.Handle,
			}
			if err := t.CheckAuthenticate(req); err != nil {
				log.From(ctx).Debug("checking handle", zap.Error(err))
				continue
			}

			log.From(ctx).Info("waiting for token removal")
			for {
				_, err := dev.Ping([]byte("echo"))
				if err != nil {
					log.From(ctx).Info("locking")
					exec.Command(cmd, args...).Run()
					break
				}
				time.Sleep(time.Second * 1)
			}
		}
		return nil
	}
}
