package cli

import (
	"encoding/json"

	"github.com/dgraph-io/badger"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "I delete things",
	RunE:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer db.Close()

	var err error
	switch len(args) {
	case 1:
		err = deleteNotebook(args[0])
	case 2:
		err = deleteEntry(args[0], args[1])
	default:
		panic("Do you mean nuke?")
	}

	return err
}

func deleteNotebook(notebook string) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(notebook))
	})
}

func deleteEntry(notebook, entry string) error {
	return db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(notebook))
		if err != nil {
			return err
		}

		n, err := unmarshalItem(item)
		if err != nil {
			return err
		}

		for key := range n.Entries {
			if key == entry {
				delete(n.Entries, key)
			}
		}

		b, err := json.Marshal(n)
		if err != nil {
			return err
		}
		return txn.Set([]byte(notebook), b)
	})
}
