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
	Out    *string
	Schema *string
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump the database schema as a single migration file",
	Long: shared.CLIHelp(`
Dumps the current database schema as a single migration file that can be applied
with psql. The result will be stable, and can be checked in to your git
repository. You can also use this command to generate a "squash" migration.

The dump command will parse your database schema and attempt to infer
dependencies between objects. For instance, if a view "active_users" is defined
by querying the "users" table, the dump will create the "users" table before it
creates the "active_users" view.

You can explicitly define dependencies between objects in a configuration file
if pgmigrate is unable to infer them for you:

    # .pgmigrate.yaml
    schema:
      name: "public"
      dependencies:
        active_users: # depends on
          - users
      data:
      - name: "%_enum"

You can include data from tables as part of the generated dump by creating a
configuration file. For instance, to include every row in every table that ends
in "_enum", you can use a configuration like this:
   
    # .pgmigrate.yaml
    schema:
      name: "public"
      data:
      - name: "%_enum"

	`),
	Example: shared.CLIExample(`
# Apply migrations
pgmigrate apply
# Dump the resulting schema as a single migration file
pgmigrate dump --out schema.sql
# Apply the schema to a new database
psql $ANOTHER_DB -f ./schema.sql
# See that there is no difference between the two schemas
pgmigrate --database $ANOTHER_DB dump --out another.sql
diff schema.sql another.sql # should show no differences
	`),
	GroupID:          "dev",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && *DumpFlags.Out == "" {
			*DumpFlags.Out = args[0]
		}
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
		if *DumpFlags.Schema != "" {
			config.Schema.Schema = *DumpFlags.Schema
		}
		parsed, err := schema.Parse(config.Schema, db)
		if err != nil {
			return err
		}
		contents := parsed.String()

		if *DumpFlags.Out != "" {
			config.Schema.Out = *DumpFlags.Out
		}
		if config.Schema.Out == "" {
			config.Schema.Out = "-"
		}

		fout := config.Schema.Out
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
	DumpFlags.Out = dumpCmd.Flags().StringP("out", "o", "", "path to write the schema to, '-' means stdout")
	DumpFlags.Schema = dumpCmd.Flags().StringP(
		"schema",
		"s",
		"",
		fmt.Sprintf(
			`the name of the database schema to dump (default "%s")`,
			schema.DefaultSchema,
		),
	)
}
