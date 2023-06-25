package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/shared"
)

var Command = &cobra.Command{ //nolint:gochecknoglobals
	Version: shared.VersionString(),
	Use:     "pgmigrate",
	Short:   "migrate postgres databases",
	Example: shared.CLIExample(``),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf(`invalid command: "%s"`, args[0])
		}
		return cmd.Help()
	},
}

func init() { //nolint:gochecknoinits
	Command.CompletionOptions.HiddenDefaultCmd = true
	Command.TraverseChildren = true
	Command.SilenceErrors = true
	Command.SilenceUsage = false
	Command.SetVersionTemplate("{{.Version}}\n")

	shared.Flags.LogFormat = Command.PersistentFlags().StringP(
		"log-format",
		"l",
		string(shared.LogFormatText),
		fmt.Sprintf("[PGM_LOGFORMAT] '%s' or '%s', the log line format", shared.LogFormatText, shared.LogFormatJSON),
	)
	shared.Flags.Database = Command.PersistentFlags().StringP(
		"database",
		"d",
		"",
		"[PGM_DATABASE] a 'postgres://...' connection string",
	)
	shared.Flags.Migrations = Command.PersistentFlags().StringP(
		"migrations",
		"m",
		"",
		"[PGM_MIGRATIONS] a path to a directory containing *.sql migrations",
	)
	_ = Command.MarkPersistentFlagDirname("migrations")
}
