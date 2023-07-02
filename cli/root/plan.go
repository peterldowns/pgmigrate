package root

import (
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cli/shared"
)

var planCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:     "plan",
	Short:   "Preview which migrations would be applied",
	GroupID: "migrating",
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
