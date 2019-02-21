package cli

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	errEntryNotFound  = errors.New("entry not present in any notebook")
	errDuplicateEntry = errors.New("multiple entries with the same name found")
	errTerminate      = errors.New("already printed user error, shutting down cli")
)

func createList(txn *badger.Txn, key []byte) error {
	log.WithField("key", string(key)).Debug("Creating new list")

	b, err := json.Marshal(newList())
	if err != nil {
		return err
	}

	return txn.Set(key, b)
}

func unmarshalItem(item *badger.Item) (*list, error) {
	l := newList()
	err := item.Value(func(val []byte) error {
		return json.Unmarshal(val, l)
	})

	if err != nil {
		return nil, err
	}
	return l, nil
}

func findItem(txn *badger.Txn, needle string) (notebook string, value string, err error) {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	found := false
	for it.Rewind(); it.Valid(); it.Next() {
		logger := log.WithField("list", fmt.Sprintf("%s", it.Item().Key()))

		l, err := unmarshalItem(it.Item())
		if err != nil {
			logger.WithError(err).Error("Couldn't unmarshal list")
			continue
		}

		for e, v := range l.Values {
			if e != needle {
				continue
			} else if found {
				log.WithField("entry name", e).Warn("Same entry name in multiple notebooks")
				return "", "", errDuplicateEntry
			}

			notebook = string(it.Item().Key())
			value = v

			found = true
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"item": e, "value": value}).
					Error("Error writing item to clipboard")
				continue
			}
		}
	}

	if !found {
		return "", "", errEntryNotFound
	}

	return notebook, value, nil
}

// notifyUser always notifies the user of the error and always returns an error
func notifyUser(err error, notebook, entry, value string) error {
	logger := log.WithError(err).WithFields(log.Fields{
		"notebook": notebook,
		"entry": entry,
		"value": value})

	switch err {
	case errEntryNotFound:
		fmt.Printf("%s not found in %s", entry, notebook)
	case errDuplicateEntry:
		fmt.Printf("%s found in multiple notebooks", entry)
	case errTerminate:
		// Do nothing
	default:
		logger.Error("Unknown error")
		fmt.Printf("ink encountered an unknown error: %v\n", err)
	}
	return errTerminate
}

// handleTerminate returns an error if it's not errTerminate
func handleTerminate(err error) error {
	if err != errTerminate {
		return err
	}
	return nil
}

// handle wraps notifyUser and handleTerminate
func handle(err error, notebook, entry, value string) error {

}
