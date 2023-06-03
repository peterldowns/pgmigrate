package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
)

var planFlags = struct { //nolint:gochecknoglobals
	Database   *string
	Migrations *string
}{}

var planCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "plan",
	Short: "Preview which migrations would be applied",
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		slogger, mlogger := newLogger()
		dir := os.DirFS(*planFlags.Migrations)
		db, err := sql.Open("pgx", *planFlags.Database)
		if err != nil {
			slogger.With("database", *planFlags.Database, "error", err).Error("could not connect to postgres")
			return nil
		}
		defer db.Close()
		plan, err := pgmigrate.Plan(ctx, db, dir, mlogger)
		if err != nil {
			slogger.With("error", err).Error("failed to get migrations plan")
			return nil
		}
		for _, m := range plan {
			slogger.With("checksum", m.MD5()).Info(m.ID)
		}
		return nil
	},
}

//nolint:gochecknoinits
func init() {
	planFlags.Migrations = planCmd.Flags().String("migrations", "", "directory of *.sql migration files")
	_ = planCmd.MarkFlagRequired("migrations")
	planFlags.Database = planCmd.Flags().String("database", "", "postgres://... connection string of the database to be migrated")
	_ = planCmd.MarkFlagRequired("database")
	root.AddCommand(planCmd)
}
