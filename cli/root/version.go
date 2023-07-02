package root

import (
	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/shared"
)

var versionCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "version",
	GroupID: "ops",
	Short:   "show the version of this binary",
	RunE: func(_ *cobra.Command, _ []string) error {
		logger, _ := shared.State.Logger()
		logger.Print(shared.VersionString())
		return nil
	},
}
