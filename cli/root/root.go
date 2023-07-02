package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/root/ops"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var Command = &cobra.Command{ //nolint:gochecknoglobals
	Version: shared.VersionString(),
	Use:     "pgmigrate",
	Short:   "migrate postgres databases",
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

	shared.State.Flags.LogFormat = Command.PersistentFlags().StringP(
		"log-format",
		"l",
		string(shared.LogFormatText),
		fmt.Sprintf("[PGM_LOGFORMAT] '%s' or '%s', the log line format", shared.LogFormatText, shared.LogFormatJSON),
	)
	shared.State.Flags.Database = Command.PersistentFlags().StringP(
		"database",
		"d",
		"",
		"[PGM_DATABASE] a 'postgres://...' connection string",
	)
	shared.State.Flags.Migrations = Command.PersistentFlags().StringP(
		"migrations",
		"m",
		"",
		"[PGM_MIGRATIONS] a path to a directory containing *.sql migrations",
	)
	shared.State.Flags.Configfile = Command.PersistentFlags().StringP(
		"configfile",
		"f",
		"",
		"[PGM_CONFIGFILE] a path to a configuration file",
	)
	_ = Command.MarkPersistentFlagDirname("migrations")

	Command.AddGroup(
		&cobra.Group{
			ID:    "migrating",
			Title: "Migrating:",
		},
		&cobra.Group{
			ID:    "ops",
			Title: "Operations:",
		},
		&cobra.Group{
			ID:    "dev",
			Title: "Development:",
		},
	)

	// migrating
	Command.AddCommand(appliedCmd)
	Command.AddCommand(planCmd)
	Command.AddCommand(verifyCmd)
	Command.AddCommand(migrateCmd)

	// ops
	Command.AddCommand(ops.Command)
	Command.AddCommand(versionCmd)

	// dev
	Command.AddCommand(debugCmd)
	Command.AddCommand(dumpCmd)
	Command.SetHelpCommandGroupID("dev")
}
