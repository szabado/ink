package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RootCmd = cobra.Command{
	Use: "ink",
	Long: "Your digital notepad to write down your stray thoughts.",
	Run: runRoot,
}

func runRoot(cmd *cobra.Command, args []string) {
	fmt.Println("Hello world")
}