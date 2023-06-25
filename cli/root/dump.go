package root

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate/cli/shared"
	"github.com/peterldowns/pgmigrate/internal/schema"
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump the current schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		database := shared.GetDatabase()
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()

		config := schema.Config{
			Schema: "public",
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		fmt.Println(schema.Dump(result))
		return err
	},
}

func init() {
	Command.AddCommand(dumpCmd)
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "debug the current schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		database := shared.GetDatabase()
		db, err := sql.Open("pgx", database.Value())
		if err != nil {
			return err
		}
		defer db.Close()

		config := schema.Config{
			Schema: "public",
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		name := "lists_tradeable_contracts"
		for _, view := range result.Views {
			if view.Name == name {
				fmt.Println(view.DependsOn())
				fmt.Println(view.String())
			}
		}
		return err
	},
}

func init() {
	Command.AddCommand(debugCmd)
}
