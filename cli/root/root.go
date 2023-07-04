package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/root/ops"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var Command = &cobra.Command{ //nolint:gochecknoglobals
	Version: shared.VersionString(),
	Use:     "pgmigrate",
	Short:   shared.DocsLink,
	Example: shared.CLIExample(`
# Preview and then apply migrations
pgmigrate plan     # Preview which migrations would be applied
pgmigrate migrate  # Apply any previously-unapplied migrations
pgmigrate verify   # Verify that migrations have been applied correctly
pgmigrate applied  # Show all previously-applied migrations

# Dump the current schema to a file
pgmigrate dump --out schema.sql
	`),
	TraverseChildren: true,
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
	shared.State.Flags.ConfigFile = Command.PersistentFlags().String(
		"configfile",
		"",
		"[PGM_CONFIGFILE] a path to a configuration file",
	)
	shared.State.Flags.LogFormat = Command.PersistentFlags().String(
		"log-format",
		"",
		fmt.Sprintf(
			"[PGM_LOGFORMAT] '%s' or '%s', the log line format (default '%s')",
			shared.LogFormatText, shared.LogFormatJSON, shared.LogFormatText,
		),
	)
	shared.State.Flags.TableName = Command.PersistentFlags().String(
		"table-name",
		"",
		fmt.Sprintf(
			"[PGM_TABLENAME] the table name to use to store migration records (default '%s')",
			pgmigrate.DefaultTableName,
		),
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
	Command.AddCommand(configCmd)
	Command.AddCommand(dumpCmd)
	Command.AddCommand(newCmd)
	Command.SetHelpCommandGroupID("dev")
}
