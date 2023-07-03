package root

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var migrateCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "migrate",
	Aliases: []string{"apply"},
	Short:   "Apply any previously-unapplied migrations",
	Long: shared.CLIHelp(`
Applies any previously un-applied migrations. It stores metadata in the database
migrations table, with the following schema:

  - id: text not null
  - checksum: text not null
  - execution_time_in_millis: integer not null
  - applied_at: timestamp with time zone not null

First, it acquires an advisory lock to prevent conflicts with other instances
that may be running in parallel. This way only one migrator will attempt to run
the migrations at any point in time. This makes it safe to use "migrate" on
app/container startup even if you run multiple copies of the application.

Second, calculate a plan of migrations to apply. The plan will be a list of
migrations that have not yet been marked as applied in the migrations table.
The migrations in the plan will be ordered by their IDs, in ascending
lexicographical order. For more information, see "pgmigrate help plan".

Third, for each migration in the plan,

  - Begin a transaction
  - Run the migration
  - Create a record in the migrations table saying that the migration was applied
  - Commit the transaction

If a migration fails at any point, the transaction will roll back. A failed
migration results in NO record for that migration in the migrations table,
which means that future attempts to run the migrations will include it in
their plan.

Migrate will immediately return the error related to a failed migration,
and will NOT attempt to run any further migrations. Any migrations applied
before the failure will remain applied. Any migrations not yet applied will
not be attempted.

Fourth, if all the migrations in the plan are applied successfully, it calls
"pgmigrate verify" to double-check that all known migrations have been marked as
applied in the migrations table.

Finally, the advisory lock is released. 
	`),
	GroupID:          "migrating",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		verrs, err := pgmigrate.Migrate(cmd.Context(), db, dir, mlogger)
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
		return nil
	},
}
