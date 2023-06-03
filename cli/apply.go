package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
)

var applyFlags = struct { //nolint:gochecknoglobals
	Migrations *string
	Database   *string
}{}

var applyCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "apply",
	Short: "Apply any previously-unapplied migrations",
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		slogger, mlogger := newLogger()
		dir := os.DirFS(*applyFlags.Migrations)
		db, err := sql.Open("pgx", *applyFlags.Database)
		if err != nil {
			slogger.With("database", *applyFlags.Database, "error", err).Error("could not connect to postgres")
			return nil
		}
		defer db.Close()
		verrs, err := pgmigrate.Migrate(ctx, db, dir, mlogger)
		if err != nil {
			slogger.With("error", err).Error("failed to apply migrations")
			return nil
		}
		for _, verr := range verrs {
			var attrs []any
			for key, val := range verr.Fields {
				attrs = append(attrs, key, val)
			}
			slogger.With(attrs...).Warn(verr.Message)
		}
		slogger.Info("complete")
		return nil
	},
}

//nolint:gochecknoinits
func init() {
	applyFlags.Migrations = applyCmd.Flags().String("migrations", "", "directory of *.sql migration files")
	_ = applyCmd.MarkFlagRequired("migrations")
	applyFlags.Database = applyCmd.Flags().String("database", "", "postgres://... connection string of the database to be migrated")
	_ = applyCmd.MarkFlagRequired("database")
	root.AddCommand(applyCmd)
}
