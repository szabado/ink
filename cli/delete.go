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
		err = deleteList(args[0])
	case 2:
		err = deleteItem(args[0], args[1])
	default:
		panic("Do you mean nuke?")
	}

	return err
}

func deleteList(list string) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(list))
	})
}

func deleteItem(list, itemName string) error {
	return db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(list))
		if err != nil {
			return err
		}

		l, err := unmarshalItem(item)
		if err != nil {
			return err
		}

		for key := range l.Values {
			if key == itemName {
				delete(l.Values, key)
			}
		}

		b, err := json.Marshal(l)
		if err != nil {
			return err
		}
		return txn.Set([]byte(list), b)
	})
}
