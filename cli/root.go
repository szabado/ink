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
	Use:          "ink [stuff]",
	Long:         "Your digital notepad to write down your stray thoughts.",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
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
	RunE: runRoot,
}

type list struct {
	Values map[string]string `json:"value"`
	//Ctime uint64 `json:"ctime"`
	//Mtime uint64 `json:"mtime"`
}

func newList() *list {
	return &list{
		Values: make(map[string]string),
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
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

	return err
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

			_, v, err := findItem(txn, string(key))
			if err == errEntryNotFound {
				log.WithField("key", string(key)).Debug("No item found")

				// The third possible behaviour here is creating a notebook
				if err := createList(txn, key); err != nil {
					return err
				}
			} else if err != errDuplicateEntry {
				// TODO: handle gracefully
				panic(err)
			} else if err != nil {
				// TODO: handle gracefully
				panic(err)
			}

			err = clipboard.WriteAll(v)
			if err != nil {
				// TODO: handle gracefully
				panic(err)
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

func twoArg(args []string) error {
	key := []byte(args[0])
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			// TODO: handle missing keys
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
