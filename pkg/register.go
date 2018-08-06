package u2flock

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/flynn/u2f/u2fhid"
	"github.com/flynn/u2f/u2ftoken"
	"github.com/seibert-media/golibs/log"
	"go.uber.org/zap"
)

// Register onto KeyFile
func (k *KeyFile) Register(ctx context.Context) error {
	log := log.From(ctx)

	devices, err := u2fhid.Devices()
	if err != nil {
		log.Error("searching devices", zap.Error(err))
		return err
	}
	if len(devices) == 0 {
		log.Error("no U2F tokens found")
		return errors.New("no U2F tokens found")
	}

	d := devices[0]
	log.Debug("device info",
		zap.String("manufacturer", d.Manufacturer),
		zap.String("product", d.Product),
		zap.Uint16("productID", d.ProductID),
		zap.Uint16("vendorID", d.VendorID),
	)

	dev, err := u2fhid.Open(d)
	if err != nil {
		log.Error("opening device", zap.Error(err))
		return err
	}
	t := u2ftoken.NewToken(dev)

	version, err := t.Version()
	if err != nil {
		log.Error("device version", zap.Error(err))
		return err
	}
	log.Debug("device version", zap.String("version", version))

	challenge := make([]byte, 32)
	io.ReadFull(rand.Reader, challenge)

	var res []byte
	log.Info("registering, provide user presence")
	for {
		res, err = t.Register(u2ftoken.RegisterRequest{Challenge: challenge, Application: app})
		if err == u2ftoken.ErrPresenceRequired {
			time.Sleep(200 * time.Millisecond)
			continue
		} else if err != nil {
			log.Error("registering", zap.Error(err))
			return err
		}
		break
	}

	log.Info("successful", zap.ByteString("result", res))
	res = res[66:]
	khLen := int(res[0])
	res = res[1:]
	keyHandle := res[:khLen]
	log.Debug("registration data", zap.ByteString("keyHandle", keyHandle))

	dev.Close()

	k.Tokens = append(k.Tokens, Token{Handle: keyHandle})

	log.Debug("updated keyFile", zap.String("data", spew.Sprint(k)))

	return nil
}
