package root

import (
	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/shared"
)

var versionCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "version",
	Short: "show the version of this binary",
	RunE: func(_ *cobra.Command, _ []string) error {
		logger, _ := shared.NewLogger()
		logger.Print(shared.VersionString())
		return nil
	},
}

func init() { //nolint:gochecknoinits
	Command.AddCommand(versionCmd)
}
