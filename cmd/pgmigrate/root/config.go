package root

import (
	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"debug"},
	Short:   "Print the current configuration / settings",
	Long: shared.CLIHelp(`
pgmigrate reads its configuration from cli flags, environment variables, and a
configuration file, in that order.

pgmigrate will look in the following locations for a configuration file:

- If you passed "--configfile <aaa>", then it reads "<aaa>"
- If you defined "PGM_CONFIGFILE=<bbb>", then it reads "<bbb>"
- If your current directory has a ".pgmigrate.yaml" file,
  it reads "$(pwd)/.pgmigrate.yaml"
- If the root of your current git repo has a ".pgmigrate.yaml" file,
  it reads "$(git_repo_root)/.pgmigrate.yaml"

Here's an example configuration file. All keys are optional, an empty file is
also a valid configuration.

    # connection string to a database to manage
    database: "postgres://postgres:password@localhost:5433/postgres"
    # path to the folder of migration files. if this is relative,
    # it is treated as relative to wherever the "pgmigrate" command
    # is invoked, NOT as relative to this config file.
    migrations: "./tmp/migrations"
    # the name of the table to use for storing migration records.  you can give
    # this in the form "table" to use your database's default schema, or you can
    # give this in the form "schema.table" to explicitly set the schema.
    table_name: "custom_schema.custom_table"
    # this key configures the "dump" command.
    schema:
      # the name of the schema to dump, defaults to "public"
      name: "public"
      # the file to which to write the dump, defaults to "-" (stdout)
      # if this is relative, it is treated as relative to wherever the
      # "pgmigrate" command is invoked, NOT as relative to this config file.
      file: "./schema.sql"
      # any explicit dependencies between database objects that are
      # necessary for the dumped schema to apply successfully.
      dependencies:
        some_view: # depends on
          - some_function
          - some_table
        some_table: # depends on
          - another_table
      # any tables for which the dump should contain INSERT statements to create
      # actual data/rows. this is useful for enums or other tables full of
      # ~constants.
      data:
        - name: "%_enum" # accepts wildcards using SQL query syntax
        - name: "my_example_table" # can also be a literal
          # if not specified, defaults to "*"
          columns:
            - "value"
            - "comment"
          # a valid SQL order clause to use to order the rows in the INSERT
          # statement.
          order_by: "value asc"
	`),
	GroupID:          "dev",
	TraverseChildren: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		logger, _ := shared.State.Logger()
		configfile := shared.State.Configfile()

		logger.Info(configfile.Name(), "is_set", configfile.IsSet(), "value", configfile.Value())

		shared.State.Parse()

		database := shared.State.Database()
		logformat := shared.State.LogFormat()
		migrations := shared.State.Migrations()
		tablename := shared.State.TableName()

		logger.Info(migrations.Name(), "is_set", migrations.IsSet(), "value", migrations.Value())
		logger.Info(database.Name(), "is_set", database.IsSet(), "value", database.Value())
		logger.Info(logformat.Name(), "is_set", logformat.IsSet(), "value", logformat.Value())
		logger.Info(tablename.Name(), "is_set", tablename.IsSet(), "value", tablename.Value())

		return nil
	},
}
