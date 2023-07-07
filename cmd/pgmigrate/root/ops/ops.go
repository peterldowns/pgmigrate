package ops

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "ops",
	Aliases: []string{"op", "admin"},
	Short:   "Perform manual operations on migration records",
	GroupID: "ops",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf(`invalid command: "%s"`, args[0])
		}
		return cmd.Help()
	},
}

func init() {
	Command.AddCommand(setChecksum)
	Command.AddCommand(recalculateChecksum)
	Command.AddCommand(markUnapplied)
	Command.AddCommand(markApplied)
}
