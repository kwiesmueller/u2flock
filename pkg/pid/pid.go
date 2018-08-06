package pid

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	"github.com/seibert-media/golibs/log"
	"go.uber.org/zap"
)

var errNoPID = errors.New("no pid file found")

// GetPID for current process
func GetPID(ctx context.Context, app []byte) (int, error) {
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
func CreatePID(ctx context.Context, app []byte) (int, error) {
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
