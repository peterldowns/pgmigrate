package root

import (
	"database/sql"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var appliedCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:              "applied",
	Aliases:          []string{"list"},
	Short:            "Show all previously-applied migrations",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := shared.GetDatabase()
		if err := shared.Validate(database); err != nil {
			return err
		}

		slogger, mlogger := shared.NewLogger()
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()

		applied, err := pgmigrate.Applied(cmd.Context(), db, mlogger)
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

//nolint:gochecknoinits
func init() {
	Command.AddCommand(appliedCmd)
}
