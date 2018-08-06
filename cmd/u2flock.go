package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/flynn/hid"

	"go.uber.org/zap"

	"github.com/flynn/u2f/u2fhid"
	u2flock "github.com/kwiesmueller/u2flock/pkg"
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

var errNoPID = errors.New("no pid file found")

// GetPID for current process
func GetPID(ctx context.Context) (int, error) {
	filePath := fmt.Sprintf("%s.pid", app)

	// check for existing instance
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		// file exists

		file, err := os.Open(filePath)
		if err != nil {
			log.From(ctx).Error("opening pidfile", zap.String("path", filePath), zap.Error(err))
			return 0, err
		}

		pidBytes, err := ioutil.ReadAll(file)
		if err != nil {
			log.From(ctx).Error("reading pidfile", zap.String("path", filePath), zap.Error(err))
			return 0, err
		}

		pid, err := strconv.ParseInt(string(pidBytes), 10, 32)
		if err != nil {
			log.From(ctx).Error("parsing pid", zap.Error(err))
			return 0, err
		}

		return int(pid), nil
	}
	return 0, errNoPID
}

// CreatePID for current process
func CreatePID(ctx context.Context) (int, error) {
	filePath := fmt.Sprintf("%s.pid", app)

	log.From(ctx).Debug("creating pid", zap.String("path", filePath))

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.From(ctx).Error("opening pidfile", zap.String("path", filePath), zap.Error(err))
		return 0, err
	}

	pid := os.Getpid()
	_, err = file.Write([]byte(strconv.Itoa(pid)))
	if err != nil {
		log.From(ctx).Error("writing pid", zap.Int("pid", pid), zap.String("path", filePath), zap.Error(err))
		return pid, err
	}

	return pid, nil
}

// CheckPID to determine if a process is active
func CheckPID(ctx context.Context, pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		log.From(ctx).Error("searching process", zap.Error(err))
		return false, err
	}

	if err := p.Signal(os.Signal(syscall.Signal(0))); err != nil {
		if err.Error() == "os: process already finished" {
			return false, nil
		}
		log.From(ctx).Error("sending signal", zap.Error(err))
		return false, err
	}

	return true, nil
}

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

		pid, err := GetPID(ctx)
		if err != nil || pid == 0 {
			log.From(ctx).Debug("getting pid", zap.Error(err))
		} else {
			running, err := CheckPID(ctx, pid)
			if err != nil {
				log.From(ctx).Fatal("checking pid", zap.Int("pid", pid), zap.Error(err))
			}

			if running {
				log.From(ctx).Error("exiting", zap.String("reason", "already running"), zap.Int("pid", pid))
				return
			}

			pid, err = CreatePID(ctx)
			if err != nil {
				log.From(ctx).Fatal("checking pid", zap.Int("pid", pid), zap.Error(err))
			}
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
