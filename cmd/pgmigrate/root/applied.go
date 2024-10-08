package root

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var appliedCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "applied",
	Aliases: []string{"list"},
	Short:   "Show all previously-applied migrations",
	Long: shared.CLIHelp(`
Prints the previously-applied migrations in the order that they were applied
(applied_at, id ASC).

If there are no applied migrations, or the specified table does not exist, this
command will print nothing and exit successfully.
	`),
	GroupID:          "migrating",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		shared.State.Parse()
		migrations := shared.State.Migrations()
		database := shared.State.Database()
		if err := shared.Validate(database, migrations); err != nil {
			return err
		}
		dir := os.DirFS(migrations.Value())

		slogger, mlogger := shared.State.Logger()
		db, err := shared.OpenDB()
		if err != nil {
			return err
		}
		defer db.Close()

		m, err := newMigrator(dir, shared.State.TableName().Value(), mlogger)
		if err != nil {
			return err
		}

		applied, err := m.Applied(cmd.Context(), db)
		if err != nil {
			return err
		}
		for _, m := range applied {
			slogger.With(
				"applied_at", m.AppliedAt,
				"checksum", m.Checksum,
				"execution_time_ms", m.ExecutionTimeInMillis,
			).Info(m.ID)
		}
		return nil
	},
}
