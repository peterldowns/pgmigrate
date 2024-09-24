package ops

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var RecalculateChecksumFlags struct {
	IDs *[]string
	All *bool
}

var recalculateChecksum = &cobra.Command{
	Use:     "recalculate-checksum",
	Aliases: []string{"recalculate", "update", "refresh", "reset"},
	Short:   "recalculate and update the checksum value of a record of an applied migration",
	Example: shared.CLIExample(`
# Recalculate the checksum of migration 123_example.sql from the migration's
# current contents, and update its record of application if the database has a
# different stored checksum.
pgmigrate ops recalculate-checksum 123_example
pgmigrate ops recalculate-checksum --id 123_example

# Recalculate the checksums for 123_example.sql and 456_another.sql and update
# the database if there were any differences.
pgmigrate ops recalculate-checksum 123_example 456_example
pgmigrate ops recalculate-checksum --id 123_example --id 456_example

# Recalculate the checksums for all migrations and update the database if there
# were any differences.
pgmigrate ops recalculate-checksum --all
	`),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// Argument parsing
		if len(args) != 0 {
			*RecalculateChecksumFlags.IDs = append(*RecalculateChecksumFlags.IDs, args...)
		}
		if len(*RecalculateChecksumFlags.IDs) != 0 && *RecalculateChecksumFlags.All {
			return fmt.Errorf("--all and --id are mutually exclusive")
		}
		if len(*RecalculateChecksumFlags.IDs) == 0 && !*RecalculateChecksumFlags.All {
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
		var updated []pgmigrate.AppliedMigration
		if *RecalculateChecksumFlags.All {
			updated, err = pgmigrate.RecalculateAllChecksums(ctx, db, dir, mlogger)
		} else {
			updated, err = pgmigrate.RecalculateChecksums(ctx, db, dir, mlogger, *RecalculateChecksumFlags.IDs...)
		}
		if err != nil {
			return err
		}
		slogger.Info("recalculated checksums", "count", len(updated))
		for _, m := range updated {
			slogger.Info("recalculated",
				"id", m.ID,
				"checksum", m.Checksum,
				"applied_at", m.AppliedAt,
			)
		}
		return nil
	},
}

func init() {
	RecalculateChecksumFlags.IDs = recalculateChecksum.Flags().StringArrayP("id", "i", nil, "migration ids of records to update checksums")
	RecalculateChecksumFlags.All = recalculateChecksum.Flags().BoolP("all", "a", false, "if true, update the checksum of all migration records")
	recalculateChecksum.MarkFlagsMutuallyExclusive("all", "id")
}
