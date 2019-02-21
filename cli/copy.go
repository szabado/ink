package cli

import (
	"github.com/atotto/clipboard"
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(copyCmd)
}

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "I copy things to your clipboard",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("I'm supposed to have two args")
		}
		return nil
	},
	RunE: runCopy,
}

func runCopy(cmd *cobra.Command, args []string) error {
	return db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(args[0]))
		if err == badger.ErrKeyNotFound {
			// TODO: Handle gracefully

			return err
		} else if err != nil {
			return err
		}

		n, err := unmarshalItem(item)
		if err != nil {
			return err
		}

		s, ok := n.Entries[args[1]]
		if !ok {
			// TODO: handle
		}

		return clipboard.WriteAll(s)
	})
}
