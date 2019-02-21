package cli

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(dumpCmd)
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "I spew notebooks",
	RunE:  runDump,
}

func runDump(cmd *cobra.Command, args []string) error {
	defer db.Close()

	switch len(args) {
	case 0:
		return dumpEverything()
	default:
		args[0] = strings.Join(args[0:], " ")
		args = args[0:1]
		fallthrough
	case 1:
		return dumpNotebook(args[0])
	}
}

func dumpEverything() error {
	return db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := printNotebookEntries(it.Item())
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func dumpNotebook(notebook string) error {
	return db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(notebook))
		if err == badger.ErrKeyNotFound {
			// TODO: handle gracefully
			return err
		} else if err != nil {
			return err
		}
		return printNotebookEntries(item)
	})
}

func printNotebookEntries(item *badger.Item) error {
	n, err := unmarshalItem(item)
	if err != nil {
		return err
	}

	fmt.Printf("%s:\n", item.Key())
	for k, v := range n.Entries {
		fmt.Printf("  %s: %s\n", k, v)
	}

	return nil
}
