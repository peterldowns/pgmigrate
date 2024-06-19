package root

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared"
)

var NewFlags struct {
	Name   *string
	Bare   *bool
	Create *bool
}

var newCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "new",
	Short: "generate the name of the next migration file based on the current sequence prefix",
	Long: shared.CLIHelp(`
Most teams use an integer prefix in the names of their migration files, to make
it easier to understand the order in which they'll be applied. For instance,

  00001_initial.sql
  00002_create_users.sql
  00003_another.sql
  ...
  01039_most_recently.sql

pgmigrate doesn't have any requirement for you to do this, but it's a good idea
because it will make pgmigrate's ordering match your expectations. (for more
information on migration ordering, see "pgmigrate help plan").

This command is a helper for generating a new migration file with a sequence
number one greater than the most recent migration, which is almost always what
you want to do.

Example:
  * your most recent migration is "00139_something.sql"
  * the next number in the sequence is "00140"
  * you run "pgmigrate new my_example"
  * the generated migration id is "00140_my_example" and the filename is
    "00140_my_example.sql"

If your sequence has reached its maximum (all "9"'s) the command will fail and
warn that the sequence has overflowed. In this case you should probably squash
your migrations (see the web documentation for more information).
	`),
	Example: shared.CLIExample(`
# Just come up with the filename, don't create it
pgmigrate new
# Use a specific name => "0001_my_example.sql"
pgmigrate new my_example
pgmigrate new --name my_example
# Only print the file name, suitable for passing to other programs
pgmigrate new --bare 
# Create the migration file as well as printing its name
pgmigrate new --create

# Create a new migration file and send it to another program
pgmigrate new vim_user_example --create --bare | xargs vim 
	`),
	GroupID:          "dev",
	TraverseChildren: true,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 1 && *NewFlags.Name == "" {
			*NewFlags.Name = args[0]
		}
		shared.State.Parse()
		migrationsDir := shared.State.Migrations()
		if err := shared.Validate(migrationsDir); err != nil {
			return err
		}
		slogger, _ := shared.State.Logger()
		migrations, err := pgmigrate.Load(os.DirFS(migrationsDir.Value()))
		if err != nil {
			return err
		}

		prefix := ""
		suffix := *NewFlags.Name
		if suffix == "" {
			suffix = "generated"
		}
		if len(migrations) == 0 {
			prefix = "00001"
			suffix = "initial"
		} else {
			lastMig := migrations[len(migrations)-1]
			id := lastMig.ID
			parts := strings.SplitN(id, "_", 2)
			if len(parts) == 0 {
				return fmt.Errorf("could not infer prefix from %s", id)
			}
			prefix = parts[0]
			size := len(prefix)
			prefix = strings.TrimLeft(prefix, "0")
			i, err := strconv.Atoi(prefix)
			if err != nil {
				return fmt.Errorf("could not parse prefix as an integer: %s", parts[0])
			}
			i++
			prefix = fmt.Sprintf("%d", i)
			if len(prefix) > size {
				return fmt.Errorf(
					"sequence overflow: next prefix '%s' has more characters (%d) than the sequence allows (%d)",
					prefix, len(prefix), size,
				)
			}
			prefix = strings.Repeat("0", size-len(prefix)) + prefix
		}

		dir := migrationsDir.Value()
		id := fmt.Sprintf("%s_%s", prefix, suffix)
		filename := fmt.Sprintf("%s.sql", id)
		fp := path.Join(dir, filename)
		if *NewFlags.Create {
			if err := os.WriteFile(fp, []byte(`-- write your migration here`), 0o660); err != nil {
				return err
			}
		}
		if *NewFlags.Bare {
			fmt.Println(fp)
		} else {
			slogger.Info("created", "id", id, "path", fp)
		}
		return nil
	},
}

func init() {
	NewFlags.Bare = newCmd.Flags().BoolP("bare", "b", false, "if true, only print the created migration file path")
	NewFlags.Create = newCmd.Flags().BoolP("create", "c", false, "if true, create the migration file")
	NewFlags.Name = newCmd.Flags().StringP("name", "n", "", "the name of the new migration (default 'generated')")
}
