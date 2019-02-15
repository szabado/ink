package datastore

import (
	"os"
	"path"
	"path/filepath"

	"github.com/dgraph-io/badger"
	log "github.com/sirupsen/logrus"
)

func DefaultLocation() string {
	home := os.ExpandEnv("${HOME}")

	return path.Join(home, ".ink")
}

func Connect(location string) (*badger.DB, error) {
	location = filepath.Clean(location)

	if _, err := os.Stat(location); err != nil {
		err := os.Mkdir(location, 0660)
		if err != nil {
			return nil, err
		}
	}

	opts := badger.DefaultOptions
	opts.ValueDir = location
	opts.Dir = location
	opts.Logger = log.StandardLogger()

	return badger.Open(opts)
}
