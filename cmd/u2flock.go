package main

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/flynn/hid"

	"go.uber.org/zap"

	"github.com/flynn/u2f/u2fhid"
	"github.com/kwiesmueller/u2flock/pkg/pid"
	"github.com/kwiesmueller/u2flock/pkg/u2flock"
	"github.com/seibert-media/golibs/log"
)

// base64 encoded "kwiesmueller/u2flock"
var app = []byte("b64-a3dpZXNtdWVsbGVyL3UyZmxvY2sK")
var done = make(chan bool)

var (
	reg         = flag.Bool("register", false, "register token")
	auth        = flag.Bool("auth", true, "run in auth mode (default: true)")
	watch       = flag.Bool("watch", false, "run in watch mode execuing lock action on token disconnect")
	lockCmd     = flag.String("lockCmd", "systemctl", "command to call when using watch")
	lockArgs    = flag.String("lockArgs", "suspend", "comma separated list of cmdline args for lockCmd")
	debug       = flag.Bool("debug", false, "enable debug logging")
	keyFilePath = flag.String("key", "/home/kwiesmueller/.secret/u2flock-key.json", "location of the key file for persisting tokens")
)

func main() {
	flag.Parse()

	ctx := log.WithLogger(
		context.Background(),
		log.New("", *debug),
	)

	keyFile := &u2flock.KeyFile{Done: done}
	err := keyFile.Open(ctx, *keyFilePath)
	if err != nil {
		log.From(ctx).Fatal("opening keyFile", zap.Error(err))
	}

	if *reg {
		err := keyFile.Register(ctx)
		if err != nil {
			log.From(ctx).Fatal("registering", zap.Error(err))
		}
		err = keyFile.Save(ctx)
		if err != nil {
			log.From(ctx).Fatal("saving keyFile", zap.Error(err))
		}
		return
	}

	err = keyFile.From(ctx)
	if err != nil {
		log.From(ctx).Fatal("fetching keyFile", zap.Error(err))
	}

	args := strings.Split(*lockArgs, ",")

	if *watch {
		go watchDevices(ctx, keyFile, keyFile.Lock(*lockCmd, args...))
	} else {

		PID, err := pid.GetPID(ctx, app)
		if err != nil {
			log.From(ctx).Debug("getting pid", zap.Error(err))
		}

		if PID != 0 {
			running, err := pid.CheckPID(ctx, PID)
			if err != nil {
				log.From(ctx).Fatal("checking pid", zap.Int("pid", PID), zap.Error(err))
			}

			if running {
				log.From(ctx).Error("exiting", zap.String("reason", "already running"), zap.Int("pid", PID))
				return
			}
		}

		PID, err = pid.CreatePID(ctx, app)
		if err != nil {
			log.From(ctx).Fatal("checking pid", zap.Int("pid", PID), zap.Error(err))
		}

		go watchDevices(ctx, keyFile, keyFile.Authenticate)
	}

	<-done
}

func watchDevices(
	ctx context.Context,
	keyFile *u2flock.KeyFile,
	action func(context.Context, *hid.DeviceInfo) error,
) {
	log.From(ctx).Info("starting auth loop")
	for {
		log.From(ctx).Debug("looking for device")

		devices, err := u2fhid.Devices()
		if err != nil {
			log.From(ctx).Error("searching devices", zap.Error(err))
			time.Sleep(time.Second * 1)
			continue
		}

		if len(devices) < 1 {
			log.From(ctx).Debug("no device found", zap.Int("sleeping", 1))
			time.Sleep(time.Second * 1)
			continue
		}

		for _, d := range devices {
			err := action(ctx, d)
			if err != nil {
				log.From(ctx).Error("authenticating", zap.Error(err))
			}
		}

		time.Sleep(time.Second * 1)
	}
}
