package u2flock

import (
	"context"
	"encoding/json"
	"os"

	"github.com/seibert-media/golibs/log"
	"go.uber.org/zap"
)

// base64 encoded "kwiesmueller/u2flock"
var app = []byte("b64-a3dpZXNtdWVsbGVyL3UyZmxvY2sK")

// Token represents a single registration token in the keyfile
type Token struct {
	Handle []byte `json:"handle"`
}

// KeyFile represents the json persistence
type KeyFile struct {
	File   *os.File  `json:"-"`
	Done   chan bool `json:"-"`
	Tokens []Token   `json:"tokens"`
}

// From loads the KeyFile
func (k *KeyFile) From(ctx context.Context) error {
	file := k.File
	defer file.Close()

	err := json.NewDecoder(file).Decode(&k)
	if err != nil {
		log.From(ctx).Error("decoding keyFile", zap.Error(err))
		return err
	}
	return nil
}

// Save the KeyFile
func (k *KeyFile) Save(ctx context.Context) error {
	file := k.File
	defer file.Close()

	err := json.NewEncoder(file).Encode(&k)
	if err != nil {
		log.From(ctx).Error("writing keyfile", zap.Error(err))
		return err
	}
	return nil
}

// Open or create the KeyFile
func (k *KeyFile) Open(ctx context.Context, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.From(ctx).Error("opening keyFile", zap.String("path", path), zap.Error(err))
		return err
	}
	k.File = file
	return nil
}
