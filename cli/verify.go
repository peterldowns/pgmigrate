package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
)

var verifyFlags = struct { //nolint:gochecknoglobals
	Migrations *string
	Database   *string
}{}

var verifyCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "verify",
	Short: "Verify that migrations have been applied correctly",
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		slogger, mlogger := newLogger()
		dir := os.DirFS(*verifyFlags.Migrations)
		db, err := sql.Open("postgres", *verifyFlags.Database)
		if err != nil {
			slogger.With("database", *verifyFlags.Database, "error", err).Error("could not connect to postgres")
		}
		defer db.Close()
		verrs, err := pgmigrate.Verify(ctx, db, dir, mlogger)
		if err != nil {
			slogger.With("error", err).Error("failed to verify migrations")
			return nil
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

//nolint:gochecknoinits
func init() {
	verifyFlags.Migrations = verifyCmd.Flags().String("migrations", "", "directory of *.sql migration files")
	_ = verifyCmd.MarkFlagRequired("migrations")
	verifyFlags.Database = verifyCmd.Flags().String("database", "", "postgres://... connection string of the database to be migrated")
	_ = verifyCmd.MarkFlagRequired("database")
	root.AddCommand(verifyCmd)
}
