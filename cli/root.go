package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	//Ctime time.Time `json:"ctime"`
	//Mtime time.Time `json:"mtime"`
}

func newList() *list {
	return &list{
		Values: make(map[string]string),
		//Ctime: time.Now(),
		//Mtime: time.Now(),
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	defer db.Close()

	var err error
	switch len(args) {
	case 0:
		err = listAll(args)
	case 1:
		err = oneArg(args)
	case 2:
		err = getEntry(args)
	default:
		args[2] = strings.Join(args[2:], " ")
		args = args[0:3]
		fallthrough
	case 3:
		err = newEntry(args[0], args[1], args[2])
	}

	return err
}

func listAll(_ []string) error {
	return handleTerminate(db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
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
	}))
}

func oneArg(args []string) error {
	key := []byte(args[0])
	return handleTerminate(db.Update(func(txn *badger.Txn) error {
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
	}))
}

func getEntry(notebook, entry string) error {
	key := []byte(notebook)
	return handleTerminate(db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return notifyUser(err, notebook, entry, "n/a")
		} else if err != nil {
			return notifyUser(err, notebook, entry, "n/a")
		}

		l, err := unmarshalItem(item)
		if err != nil {
			return err
		}
		return clipboard.WriteAll(l.Values[entry])
	}))
}

func newEntry(notebook, entry, content string) error {
	key := []byte(notebook)

	err := db.Update(func(txn *badger.Txn) error {
		l := newList()

		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			l, err = unmarshalItem(item)

			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		l.Values[entry] = content

		b, err := json.Marshal(l)
		if err != nil {
			return err
		}

		return txn.Set(key, b)
	})

	return handle(err, notebook, entry, content)
}
