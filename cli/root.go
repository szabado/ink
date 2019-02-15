package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/dgraph-io/badger"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/szabado/ink/datastore"
)

var (
	db    *badger.DB
	debug bool
)

func init() {
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Print debug logs.")
}

var RootCmd = cobra.Command{
	Use:  "ink [stuff]",
	Long: "Your digital notepad to write down your stray thoughts.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.FatalLevel)
		if debug {
			log.SetLevel(log.TraceLevel)
		}

		var err error
		db, err = datastore.Connect(datastore.DefaultLocation())
		if err != nil {
			log.WithError(err).Fatalf("Could not open ink datastore")
		}
	},
	Run: runRoot,
}

type list struct {
	Values map[string]string `json:"value"`
}

func newList() *list {
	return &list{
		Values: make(map[string]string),
	}
}

func runRoot(cmd *cobra.Command, args []string) {
	defer db.Close()

	var err error
	switch len(args) {
	case 0:
		err = zeroArg(args)
	case 1:
		err = oneArg(args)
	case 2:
		err = twoArg(args)
	default:
		args[2] = strings.Join(args[2:], " ")
		args = args[0:3]
		fallthrough
	case 3:
		err = threeArg(args)
	}

	if err != nil {
		log.WithError(err).Fatalf("Error querying database")
	}
}

func zeroArg(_ []string) error {
	err := db.View(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		it := txn.NewIterator(opt)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			l, err := unmarshalItem(it.Item())
			if err != nil {
				fmt.Printf("%s [data corrupted]\n", it.Item().Key())
			} else {
				fmt.Printf("%s (%v)\n", it.Item().Key(), len(l.Values))
			}
		}

		return nil
	})

	return err
}

func oneArg(args []string) error {
	key := []byte(args[0])
	err := db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			log.WithField("key", string(key)).Debug("No list found, looking for matching item")
			if !findItem(txn, string(key)) {
				log.WithField("key", string(key)).Debug("No item found")
				if err := createList(txn, key); err != nil {
					return err
				}
			}
			return nil
		} else if err != nil {
			return err
		}

		l, err := unmarshalItem(item)
		if err != nil {
			return err
		}

		for itemName := range l.Values {
			fmt.Println(itemName)
		}
		return nil
	})

	return err
}

func findItem(txn *badger.Txn, item string) bool {
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

		// TODO: aggregate all duplicates/warn if there are any
		for itemName, itemValue := range l.Values {
			if itemName != item {
				continue
			}

			found = true
			err := clipboard.WriteAll(itemValue)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"item": itemName, "value": itemValue}).
					Error("Error writing item to clipboard")
				continue
			}
		}
	}

	return found
}

func twoArg(args []string) error {
	key := []byte(args[0])
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		l, err := unmarshalItem(item)
		if err != nil {
			return err
		}
		return clipboard.WriteAll(l.Values[args[1]])
	})

	return err
}

func threeArg(args []string) error {
	// TODO: Support vim/other editors
	key := []byte(args[0])

	err := db.Update(func(txn *badger.Txn) error {
		l := newList()

		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		} else {
			l, err = unmarshalItem(item)

			if err != nil {
				return err
			}
		}

		l.Values[args[1]] = args[2]

		b, err := json.Marshal(l)
		if err != nil {
			return err
		}

		return txn.Set(key, b)
	})

	return err
}

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
