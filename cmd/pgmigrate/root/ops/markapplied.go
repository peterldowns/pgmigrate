package ops

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var MarkAppliedFlags struct {
	IDs *[]string
	All *bool
}

var markApplied = &cobra.Command{
	Use:     "mark-applied",
	Aliases: []string{"create"},
	Short:   "mark migrations as having been applied without actually running them",
	Example: shared.CLIExample(`
# Mark 123_example.sql as applied without running the migration
pgmigrate ops mark-applied 123_example
pgmigrate ops mark-applied --id 123_example

# Mark 123_example.sql and 456_another.sql as applied without running them
pgmigrate ops mark-applied 123_example 456_another
pgmigrate ops mark-applied --id 123_example --id 456_another

# Mark all migrations as having been applied
pgmigrate ops mark-applied --all
	`),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// Argument parsing
		if len(args) != 0 {
			*MarkAppliedFlags.IDs = append(*MarkAppliedFlags.IDs, args...)
		}
		if len(*MarkAppliedFlags.IDs) != 0 && *MarkAppliedFlags.All {
			return fmt.Errorf("--all and --id are mutually exclusive")
		}
		if len(*MarkAppliedFlags.IDs) == 0 && !*MarkAppliedFlags.All {
			return fmt.Errorf("must pass at least one migration ID with --id or --all")
		}
		shared.State.Parse()
		migrationsDir := shared.State.Migrations()
		database := shared.State.Database()
		if err := shared.Validate(database, migrationsDir); err != nil {
			return err
		}
		db, err := shared.OpenDB()
		if err != nil {
			return err
		}
		defer db.Close()
		dir := os.DirFS(migrationsDir.Value())
		slogger, mlogger := shared.State.Logger()

		// Execution
		var applied []pgmigrate.AppliedMigration
		if *MarkAppliedFlags.All {
			slogger.Info("marking ALL as applied")
			applied, err = pgmigrate.MarkAllApplied(ctx, db, dir, mlogger)
		} else {
			applied, err = pgmigrate.MarkApplied(ctx, db, dir, mlogger, *MarkAppliedFlags.IDs...)
		}
		if err != nil {
			return err
		}
		slogger.Info("marked migrations as applied", "count", len(applied))
		for _, m := range applied {
			slogger.Info("marked as applied",
				"id", m.ID,
				"checksum", m.Checksum,
				"applied_at", m.AppliedAt,
			)
		}
		return nil
	},
}

func init() {
	MarkAppliedFlags.IDs = markApplied.Flags().StringArrayP("id", "i", nil, "migration ids of records to mark as applied")
	MarkAppliedFlags.All = markApplied.Flags().BoolP("all", "a", false, "if true, mark all migrations as applied")
	markApplied.MarkFlagsMutuallyExclusive("id", "all")
}
