package ops

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var MarkUnappliedFlags struct {
	IDs *[]string
	All *bool
}

var markUnapplied = &cobra.Command{
	Use:     "mark-unapplied",
	Aliases: []string{"remove", "rm", "delete"},
	Short:   "mark migrations as having NOT been applied by removing the records that said they were",
	Example: shared.CLIExample(`
# Mark 123_example.sql as unapplied by removing the record showing that it was
# applied.
pgmigrate ops mark-unapplied 123_example
pgmigrate ops mark-unapplied --id 123_example

# Mark 123_example.sql and 456_another.sql as unapplied by removing the records
# showing that they were applied.
pgmigrate ops mark-unapplied 123_example 456_another
pgmigrate ops mark-unapplied --id 123_example --id 456_another

# Mark all migrations as unapplied by removing all records of migrations having
# been applied.
pgmigrate ops mark-unapplied --all
	`),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// Argument parsing
		if len(args) != 0 {
			*MarkUnappliedFlags.IDs = append(*MarkUnappliedFlags.IDs, args...)
		}
		if len(*MarkUnappliedFlags.IDs) != 0 && *MarkUnappliedFlags.All {
			return fmt.Errorf("--all and --id are mutually exclusive")
		}
		if len(*MarkUnappliedFlags.IDs) == 0 && !*MarkUnappliedFlags.All {
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
		m, err := newMigrator(dir, shared.State.TableName().Value(), mlogger)
		if err != nil {
			return err
		}

		// Execution
		var removed []pgmigrate.AppliedMigration
		if *MarkUnappliedFlags.All {
			removed, err = m.MarkAllUnapplied(ctx, db)
		} else {
			removed, err = m.MarkUnapplied(ctx, db, *MarkUnappliedFlags.IDs...)
		}
		if err != nil {
			return err
		}
		slogger.Info("finished removing migrations", "count", len(removed))
		for _, m := range removed {
			slogger.Info("removed",
				"id", m.ID,
				"checksum", m.Checksum,
				"applied_at", m.AppliedAt,
			)
		}
		return nil
	},
}

func init() {
	MarkUnappliedFlags.IDs = markUnapplied.Flags().StringArrayP("id", "i", nil, "migration ids of records to remove")
	MarkUnappliedFlags.All = markUnapplied.Flags().BoolP("all", "a", false, "if true, remove all migration records")
	markUnapplied.MarkFlagsMutuallyExclusive("id", "all")
}
