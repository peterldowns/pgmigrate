package root

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/shared"
	"github.com/peterldowns/pgmigrate/internal/schema"
)

var DumpFlags struct {
	File *string
}

var dumpCmd = &cobra.Command{
	Use:     "dump",
	Short:   "Dump the current schema",
	GroupID: "dev",
	RunE: func(cmd *cobra.Command, args []string) error {
		shared.State.Parse()
		database := shared.State.Database()
		if err := shared.Validate(database); err != nil {
			return err
		}
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()

		config := shared.State.Config
		parsed, err := schema.Parse(config.Schema, db)
		if err != nil {
			return err
		}
		contents := parsed.String()

		fout := *DumpFlags.File
		if fout == "-" || fout == "" {
			fmt.Println(contents)
		} else {
			file, err := os.OpenFile(fout, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			defer file.Close()
			fmt.Fprintln(file, contents)
		}
		return nil
	},
}

func init() {
	DumpFlags.File = dumpCmd.Flags().StringP("out", "o", "-", "path to write the schema to")
}
