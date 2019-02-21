package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/dgraph-io/badger"
	"github.com/dustin/go-humanize"
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
		db, err = datastore.Connect(datastore.DefaultDataFolder())
		if err != nil {
			log.WithError(err).Fatalf("Could not open ink datastore")
		}
	},
	RunE: runRoot,
}

type notebook struct {
	Entries map[string]string `json:"value"`
	Ctime   time.Time         `json:"ctime"`
	Mtime   time.Time         `json:"mtime"`
}

func newNotebook() *notebook {
	now := time.Now()
	return &notebook{
		Entries: make(map[string]string),
		Ctime:   now,
		Mtime:   now,
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	defer db.Close()

	var err error
	switch len(args) {
	case 0:
		err = listAllNotebooks()
	case 1:
		err = oneArg(args[0])
	case 2:
		err = getEntry(args[0], args[1])
	default:
		args[2] = strings.Join(args[2:], " ")
		args = args[0:3]
		fallthrough
	case 3:
		err = newEntry(args[0], args[1], args[2])
	}

	return err
}

func listAllNotebooks() error {
	return handleTerminate(db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			n, err := unmarshalItem(it.Item())
			nName := string(it.Item().Key())
			if err != nil {
				fmt.Printf("  %s [data corrupted]\n", nName)
			} else {
				fmt.Printf("  %s (%v)\n", nName, len(n.Entries))
			}
		}

		return nil
	}))
}

func oneArg(key string) error {
	var (
		notebook = key
		entry    = ""
		value    = "n/a"
	)

	err := db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(notebook))
		if err == nil {
			// We found the notebook we were looking for
			n, err := unmarshalItem(item)
			if err != nil {
				return err
			}

			for entry, content := range n.Entries {
				fmt.Printf("    %s (%s)\n", entry, humanize.Bytes(uint64(len(content))))
			}
			return nil
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		// It wasn't a notebook.  Try to find an entry
		entry = key
		notebook = ""

		_, v, err := findEntry(txn, entry)
		if err == nil {
			// We found the entry we were looking for
			value = v
			return clipboard.WriteAll(v)
		} else if err != errEntryNotFound {
			return err
		}

		notebook = key
		entry = ""
		// Create a notebook with the specified name
		return createList(txn, notebook)
	})

	return handle(err, notebook, entry, value)
}

func getEntry(notebook, entry string) error {
	key := []byte(notebook)
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return notifyUser(err, notebook, entry, "n/a")
		} else if err != nil {
			return notifyUser(err, notebook, entry, "n/a")
		}

		n, err := unmarshalItem(item)
		if err != nil {
			return err
		}
		return clipboard.WriteAll(n.Entries[entry])
	})

	return handle(err, notebook, entry, "n/a")
}

func newEntry(notebook, entry, content string) error {
	key := []byte(notebook)

	err := db.Update(func(txn *badger.Txn) error {
		n := newNotebook()

		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		} else if err == nil {
			n, err = unmarshalItem(item)

			if err != nil {
				return err
			}
		}

		n.Entries[entry] = content

		b, err := json.Marshal(n)
		if err != nil {
			return err
		}

		return txn.Set(key, b)
	})

	return handle(err, notebook, entry, content)
}
