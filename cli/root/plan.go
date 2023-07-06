package root

import (
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var planCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "plan",
	Short: "Preview which migrations would be applied",
	Long: shared.CLIHelp(`
Plan shows which migrations, if any, would be applied, in the order that they
would be applied in.

The plan will be a list of any migrations that are present in the migrations
directory that have not yet been marked as applied in the migrations table.

The migrations in the plan will be ordered by their IDs, in ascending
lexicographical order. This is the same order that you see if you use "ls".
This is also the same order that they will be applied in.

The ID of a migration is its filename without the ".sql" suffix.

A migration will only ever be applied once. Editing the contents of the
migration file will NOT result in it being re-applied. Instead, you will see a
verification error warning that the contents of the migration differ from its
contents when it was previously applied.

Migrations can be applied "out of order". For instance, if there were three
migrations that had been applied:

  - 001_initial
  - 002_create_users
  - 003_create_viewers

And a new migration "002_create_companies" is merged:

  - 001_initial
  - 002_create_companies
  - 002_create_users
  - 003_create_viewers

Running "pgmigrate plan" will show:

  - 002_create_companies

Because the other migrations have already been applied. This is by design; most
of the time, when you're working with your coworkers, you will not write
migrations that conflict with each other. As long as you use a migration
name/number higher than that of any dependencies, you will not have any
problems.
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

		plan, err := pgmigrate.Plan(cmd.Context(), db, dir, mlogger)
		if err != nil {
			return err
		}
		for _, m := range plan {
			slogger.With("checksum", m.MD5()).Info(m.ID)
		}
		return nil
	},
}
