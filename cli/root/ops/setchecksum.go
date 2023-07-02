package ops

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var SetChecksumFlags struct {
	ID       *string
	All      *bool
	Checksum *string
}

var setChecksum = &cobra.Command{
	Use:     "set-checksum",
	Aliases: []string{"checksum", "set-hash", "hash", "update"},
	Short:   "set the checksum value of a record of an applied migration",
	Example: shared.CLIExample(`
# Mark migration 123_example.sql as having been applied with checksum 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
pgmigrate ops set-checksum 123_example aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
pgmigrate ops set-checksum --id 123_example --checksum aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
	`),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// Argument parsing
		if len(args) == 2 {
			*SetChecksumFlags.ID = args[0]
			*SetChecksumFlags.Checksum = args[1]
		} else if len(args) != 0 {
			return fmt.Errorf("unexpected arguments: ['%s']", strings.Join(args, "', '"))
		}
		var missing []string
		if *SetChecksumFlags.ID == "" {
			missing = append(missing, "--id")
		}
		if *SetChecksumFlags.Checksum == "" {
			missing = append(missing, "--checksum")
		}
		if len(missing) == 1 {
			return fmt.Errorf(`required flag "%s" not set`, missing[0])
		}
		if len(missing) > 1 {
			return fmt.Errorf(`required flags "%s" not set`, strings.Join(missing, `", "`))
		}
		shared.State.Parse()
		migrationsDir := shared.State.Migrations()
		database := shared.State.Database()
		if err := shared.Validate(database, migrationsDir); err != nil {
			return err
		}
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()
		dir := os.DirFS(migrationsDir.Value())
		slogger, mlogger := shared.State.Logger()

		updated, err := pgmigrate.SetChecksums(ctx, db, dir, mlogger, pgmigrate.ChecksumUpdate{
			MigrationID: *SetChecksumFlags.ID,
			NewChecksum: *SetChecksumFlags.Checksum,
		})
		if err != nil {
			return err
		}
		slogger.Info("set migration checksum", "count", len(updated))
		for _, m := range updated {
			slogger.Info("set checksum",
				"id", m.ID,
				"checksum", m.Checksum,
				"applied_at", m.AppliedAt,
			)
		}
		return nil
	},
}

func init() {
	SetChecksumFlags.ID = setChecksum.Flags().StringP("id", "i", "", "migration ids of records to update checksums")
	SetChecksumFlags.Checksum = setChecksum.Flags().StringP("checksum", "c", "", "if true, update the checksum of all migration records")
}
