package main

import (
	"context"
	"database/sql"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
)

var appliedFlags = struct { //nolint:gochecknoglobals
	Database *string
}{}

var appliedCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:              "applied",
	Short:            "Show all previously-applied migrations",
	TraverseChildren: true,
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		slogger, mlogger := newLogger()
		db, err := sql.Open("pgx", *appliedFlags.Database)
		if err != nil {
			slogger.With("database", *appliedFlags.Database, "error", err).Error("could not connect to postgres")
			return nil
		}
		defer db.Close()
		applied, err := pgmigrate.Applied(ctx, db, mlogger)
		if err != nil {
			slogger.With("error", err).Error("failed to get applied migrations")
			return nil
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
	appliedFlags.Database = appliedCmd.PersistentFlags().String("database", "", "postgres://... connection string of the database to be migrated")
	_ = appliedCmd.MarkFlagRequired("database")
	root.AddCommand(appliedCmd)
}
