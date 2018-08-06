package u2flock

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"os/exec"
	"time"

	"github.com/flynn/hid"
	"github.com/flynn/u2f/u2fhid"
	"github.com/flynn/u2f/u2ftoken"
	"github.com/seibert-media/golibs/log"
	"go.uber.org/zap"
)

// Authenticate against KeyFile
func (k *KeyFile) Authenticate(ctx context.Context, d *hid.DeviceInfo) error {
	log := log.From(ctx)

	if len(k.Tokens) < 1 {
		return errors.New("no tokens in keyFile")
	}

	challenge := make([]byte, 32)
	for _, token := range k.Tokens {
		// check device

		dev, err := u2fhid.Open(d)
		if err != nil {
			log.Error("opening device")
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
			log.Debug("checking handle", zap.Error(err))
			continue
		}

		// start auth
		io.ReadFull(rand.Reader, challenge)

		log.Debug("authenticating")
		for {
			res, err := t.Authenticate(req)
			if err == u2ftoken.ErrPresenceRequired {
				time.Sleep(200 * time.Millisecond)
				continue
			} else if err != nil {
				log.Error("authenticating", zap.Error(err))
				return err
			}
			log.Debug("auth info", zap.Uint32("counter", res.Counter), zap.ByteString("signature", res.Signature))
			log.Info("killing i3lock")
			exec.Command("killall", "i3lock").Run()
			k.Done <- true
			return nil
		}
	}
	return nil
}
