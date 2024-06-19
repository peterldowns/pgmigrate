package root

import (
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var verifyCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "verify",
	Short: "Verify that migrations have been applied correctly",
	Long: shared.CLIHelp(`
Warns about any migrations that:
- are marked as applied in the database table but do not exist in the migrations
directory
- have a different checksum in the database than the current file hash

If there are any warnings, exits with status code 1.
Otherwise, succeeds without printing anything and exits with status code 0.
	`),
	GroupID:          "migrating",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		shared.State.Parse()
		database := shared.State.Database()
		migrations := shared.State.Migrations()
		if err := shared.Validate(database, migrations); err != nil {
			return err
		}

		slogger, mlogger := shared.State.Logger()
		dir := os.DirFS(migrations.Value())
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()

		verrs, err := pgmigrate.Verify(cmd.Context(), db, dir, mlogger)
		if err != nil {
			return err
		}
		for _, verr := range verrs {
			var attrs []any
			for key, val := range verr.Fields {
				attrs = append(attrs, key, val)
			}
			slogger.With(attrs...).Warn(verr.Message)
		}
		if len(verrs) != 0 {
			os.Exit(1)
		}
		return nil
	},
}
