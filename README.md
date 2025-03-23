# 🐽 pgmigrate

![Latest Version](https://badgers.space/badge/latest%20version/v0.2.1/blueviolet?corner_radius=m)
![Golang](https://badgers.space/badge/golang/1.18+/blue?corner_radius=m)

pgmigrate is a modern Postgres migrations CLI and golang library. It is
designed for use by high-velocity teams who practice continuous deployment. The
goal is to make migrations as simple and reliable as possible.

### Major features

- Applies any previously-unapplied migrations, in ascending filename order &mdash; that's it.
- Each migration is applied within a transaction.
- Only "up" migrations, no "down" migrations.
- Uses [Postgres advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) so it's safe to run in parallel.
- All functionality is available as a golang library, a docker container, and as a static cli binary
- Can dump your database schema and data from arbitrary tables to a single migration file
  - This lets you squash migrations
  - This lets you prevent schema conflicts in CI
  - The dumped sql is human readable
  - The dumping process is roundtrip-stable (*dumping > applying > dumping* gives you the same result)
- Supports a shared configuration file that you can commit to your git repo
- CLI contains "ops" commands for manually modifying migration state in your database, for those rare occasions when something goes wrong in prod.
- Compatible with [pgtestdb](https://github.com/peterldowns/pgtestdb) so database-backed tests are very fast.

# Documentation

- The primary documentation is [this Github README, https://github.com/peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate).
- The code itself is supposed to be well-organized, and each function has a
  meaningful docstring, so you should be able to explore it quite easily using
  an LSP plugin or by reading the code in Github or in your local editor.
- You may also refer to [the go.dev docs, pkg.go.dev/github.com/peterldowns/pgmigrate](https://pkg.go.dev/github.com/peterldowns/pgmigrate).

# Quickstart Example

[Please visit the `./example` directory](./example/) for a working example of
how to use pgmigrate. This example demonstrates:

- Using the CLI
- Creating and applying new migrations
- Dumping your schema to a file
- Using pgmigrate as an embedded library to run migrations on startup
- Writing extremely fast database-backed tests

# CLI

## Install

#### Homebrew:
```bash
# install it
brew install peterldowns/tap/pgmigrate
```

#### Download a binary:
Visit [the latest Github release](https://github.com/peterldowns/pgmigrate/releases/latest) and pick the appropriate binary. Or, click one of the shortcuts here:
- [darwin-amd64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-darwin-amd64)
- [darwin-arm64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-darwin-arm64)
- [linux-amd64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-linux-amd64)
- [linux-arm64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-linux-arm64)

#### Nix (flakes):
```bash
# run it
nix run github:peterldowns/pgmigrate -- --help
# install it
nix profile install --refresh github:peterldowns/pgmigrate
```

#### Docker:
The prebuilt docker container is `ghcr.io/peterldowns/pgmigrate` and each
version is properly tagged. You may reference this in a kubernetes config
as an init container.

To run the pgmigrate cli:

```bash
# The default CMD is "pgmigrate" which just shows the help screen.
docker run -it --rm ghcr.io/peterldowns/pgmigrate:latest
# To actually run migrations, you'll want to make sure the container can access
# your database and migrations directory and specify a command. To access a
# database running on the host, use `host.docker.internal` instead of
# `localhost` in the connection string:
docker run -it --rm \
  --volume $(pwd)//migrations:/migrations \
  --env PGM_MIGRATIONS=/migrations \
  --env PGM_DATABASE='postgresql://postgres:password@host.docker.internal:5433/postgres' \
  ghcr.io/peterldowns/pgmigrate:latest \
  pgmigrate plan
```

#### Golang:
I recommend installing a different way, since the installed binary will not
contain version information.

```bash
# run it
go run github.com/peterldowns/pgmigrate/cmd/pgmigrate@latest --help
# install it
go install github.com/peterldowns/pgmigrate/cmd/pgmigrate@latest
```

## Configuration

pgmigrate reads its configuration from cli flags, environment variables, and a
configuration file, in that order.

pgmigrate will look in the following locations for a configuration file:

- If you passed `--configfile <aaa>`, then it reads `<aaa>`
- If you defined `PGM_CONFIGFILE=<bbb>`, then it reads `<bbb>`
- If your current directory has a `.pgmigrate.yaml` file,
  it reads `$(pwd)/.pgmigrate.yaml`
- If the root of your current git repo has a `.pgmigrate.yaml` file,
  it reads `$(git_repo_root)/.pgmigrate.yaml`

Here's an example configuration file. All keys are optional, an empty file is
also a valid configuration.

```yaml
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
```
## Usage

The CLI ships with documentation and examples built in, please see `pgmigrate
help` and `pgmigrate help <command>` for more details.

```shell
# pgmigrate --help
Docs: https://github.com/peterldowns/pgmigrate

Usage:
  pgmigrate [flags]
  pgmigrate [command]

Examples:
  # Preview and then apply migrations
  pgmigrate plan     # Preview which migrations would be applied
  pgmigrate migrate  # Apply any previously-unapplied migrations
  pgmigrate verify   # Verify that migrations have been applied correctly
  pgmigrate applied  # Show all previously-applied migrations
  
  # Dump the current schema to a file
  pgmigrate dump --out schema.sql

Migrating:
  applied     Show all previously-applied migrations
  migrate     Apply any previously-unapplied migrations
  plan        Preview which migrations would be applied
  verify      Verify that migrations have been applied correctly

Operations:
  ops         Perform manual operations on migration records
  version     Print the version of this binary

Development:
  config      Print the current configuration / settings
  dump        Dump the database schema as a single migration file
  help        Help about any command
  new         generate the name of the next migration file based on the current sequence prefix

Flags:
      --configfile string   [PGM_CONFIGFILE] a path to a configuration file
  -d, --database string     [PGM_DATABASE] a 'postgres://...' connection string
  -h, --help                help for pgmigrate
      --log-format string   [PGM_LOGFORMAT] 'text' or 'json', the log line format (default 'text')
  -m, --migrations string   [PGM_MIGRATIONS] a path to a directory containing *.sql migrations
      --table-name string   [PGM_TABLENAME] the table name to use to store migration records (default 'public.pgmigrate_migrations')
  -v, --version             version for pgmigrate

Use "pgmigrate [command] --help" for more information about a command.
```  

# Library

## Install

* requires golang 1.18+ because it uses generics. 
* only depends on stdlib; all dependencies in the go.mod are for tests.

```bash
# library
go get github.com/peterldowns/pgmigrate@latest
```

## Usage

All of the methods available in the CLI are equivalently named and available in
the library. Please read the cli help with `pgmigrate help <command>` or read
the [the go.dev docs at pkg.go.dev/github.com/peterldowns/pgmigrate](https://pkg.go.dev/github.com/peterldowns/pgmigrate).

# FAQ

## How does it work?

pgmigrate has the following invariants, rules, and behavior:

- A migration is a file whose name ends in `.sql`. The part before the extension is its unique ID.
- All migrations are "up" migrations, there is no such thing as a "down" migration.
- The migrations table is a table that pgmigrate uses to track which migrations have been applied. It has the following schema:
  - `id (text not null)`: the ID of the migration
  - `checksum (text not null)`: the MD5() hash of the contents of the migration when it was applied.
  - `execution_time_in_millis (integer not null)`: how long it took to apply the migration, in milliseconds.
  - `applied_at (timestamp with time zone not null)`: the time at which the migration was finished applying and this row was inserted.
- A plan is an ordered list of previously-unapplied migrations. The migrations are sorted by their IDs, in ascending lexicographical/alphabetical order. This is the same order that you get when you use `ls` or `sort`.
- Each time migrations are applied, pgmigrate calculates the plan, then attempts to apply each migration one at a time.
- To apply a migration, pgmigrate:
  - Begins a transaction.
    - Runs the migration SQL.
    - Creates and inserts a new row in the migrations table.
  - Commits the transaction.
- Because each migration is applied in an explicit transaction, you **must not** use `BEGIN`/`COMMIT`/`ROLLBACK` within your migration files.
- Any error when applying a migration will result in an immediate failure. If there are other migrations later in the plan, they will not be applied.
- If and only if a migration is applied successfully, there will be a row in the `migrations` table containing its ID.
- pgmigrate uses [Postgres advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) to ensure that only once instance is attempting to run migrations at any point in time.
- It is safe to run migrations as part of an init container, when your binary starts, or any other parallel way.
- After a migration has been applied you should not edit the file's contents.
  - Editing its contents will not cause it to be re-applied.
  - Editing its contents will cause pgmigrate to show a warning that the hash of the migration differs from the hash of the migration when it was applied.
- After a migration has been applied you should never delete the migration. If you do, pgmigrate will warn you that a migration that had previously been applied is no longer present.

## Why use pgmigrate instead of the alternatives?

pgmigrate has the following features and benefits:

- your team can merge multiple migrations with the same sequence number (00123_create_a.sql, 00123_update_b.sql).
- your team can merge multiple migrations "out of order" (merge 00123_create_a.sql, then merge 00121_some_other.sql).
- your team can dump a human-readable version of your database schema to help with debugging and to prevent schema conflicts while merging PRs.
- your team can squash migration files to speed up new database creation and reduce complexity.
- you never need to think about down migrations ever again (you don't use them and they're not necessary).
- you can see exactly when each migration was applied, and the hash of the file
  contents of that migration, which helps with auditability and debugging.
- if a migration fails you can simply edit the file and then redeploy without
having to perform any manual operations.
- the full functionality of pgmigrate is available no matter how you choose to use it (cli, embedded library, docker container).

## How should my team work with it?

### the migrations directory
Your team repository should include a `migrations/` directory containing all known migrations.

```
migrations
├── 0001_cats.sql
├── 0003_dogs.sql
├── 0003_empty.sql
├── 0004_rm_me.sql
```

Because your migrations are applied in ascending lexicographical order, you
should use a consistent-length numerical prefix for your migration files. This
will mean that when you `ls` the directory, you see the migrations in the same
order that they will be applied.  Some teams use unix timestamps, others use
integers, it doesn't matter as long as you're consistent.

### creating a new migration
Add a new migration by creating a new file in your `migrations/` directory
ending in `.sql`. The usual work flow is:
- Create a new feature branch
- Create a new migration with a sequence number one greater than the most recent migration
- Edit the migration contents

It is OK for you and another coworker to use the same sequence number. If you
both choose the exact same filename, git will prevent you from merging both PRs.

### what's allowed in a migration
You can do anything you'd like in a migration except for the following limitations:

- migrations **must not** use transactions (`BEGIN/COMMIT/ROLLBACK`) as pgmigrate will
run each migration inside of a transaction.
- migrations **must not** use `CREATE INDEX CONCURRENTLY` as this is guaranteed to fail
inside of a transaction.

### preventing conflicts
You may be wondering, how is running "any previously unapplied migration" safe? 
What if there are two PRs that contain conflicting migrations?

For instance let's say two new migrations get created,

- `0006_aaa_delete_users.sql`, which deletes the `users` table
- `0006_bbb_create_houses.sql`, which creates a new `houses` table with a foreign key to `users`.

```
├── ...
├── 0006_aaa_delete_users.sql
├── 0006_bbb_create_houses.sql
```

There's no way both of these migrations could be safely applied, and the
resulting database state could be different depending on the order!

- If `0006_aaa_delete_users.sql` is merged and applied first, then
  `0006_bbb_create_houses.sql` is guaranteed to fail because there is no longer
  a `users` table to reference in the foreign key.
- If `0006_bbb_create_houses.sql` is merged and applied first, then
  `0006_aaa_delete_users.sql` will either fail (because it cannot delete the
  users table) or result in the deletion of the houses table as well (in the
  case of `ON DELETE CASCADE` on the foreign key).


You can prevent this conflict at CI-time by using pgmigrate to maintain an
up-to-date dump of your database schema. This schema dump will cause a git
merge conflict so that only one of the migrations can be merged, and the second
will force the developer to update the PR and the migration:

```bash
# schema.sql should be checked in to your repository, and CI should enforce that
# it is up to date. The easiest way to do this is to spin up a database, apply
# the migrations, and run the dump command.  Then, error if there are any
# changes detected:
pgmigrate dump -o schema.sql
```

You should also make sure to run a CI check on your main/dev branch that creates
a new database and applies all known migrations. This check should block
deploying until it succeeds.

Returning to the example of two conflicting migrations being merged, we can see
how these guards provide a good developer experience and prevent a broken
migration from being deployed:

1. One of the two migrations is merged. The second branch should not be able to be merged
because the dumped schema.sql will contain a merge conflict.
2. If for some reason both of the migrations are able to be merged, the check on
the main/dev branch will fail to apply migrations and block the deploy.
because the migrations cannot be applied. Breaking main is annoying, but...

Lastly, you should expect this situation to happen only rarely. Most teams, even
with large numbers of developers working in parallel, coordinate changes to
shared tables such that conflicting schema changes are a rare event.

### deploying and applying migrations
You should run pgmigrate with the latest migrations directory each time you
deploy. You can do this by:

- using pgmigrate as a golang library, and calling `pgmigrate.Migrate(...)` 
  when your application starts
- using pgmigrate as a cli or as a docker init container and applying
  migrations before your application starts.

Your application should fail to start if migrations fail for any reason.

Your application should start successfully if there are verification errors or
warnings, but you should treat those errors as a sign there is a difference
between the expected database state and the schema as defined by your migration
files.

Because pgmigrate uses advisory locks, you can roll out as many new instances of
your application as you'd like. Even if multiple instance attempt to run the
migrations at once, only one will acquire the lock and apply the migrations. The
other instances will wait for it to succeed and then no-op.

### backwards compatibility
Assuming you're running in a modern cloud environment, you're most
likely doing rolling deployments where new instances of your application are
brought up before old ones are terminated. Therefore, make sure any new
migrations will result in a database state that the previous version of your
application (which will still be running as migrations are applied) can handle.

### squashing migrations

At some point, if you have hundreds or thousands of migration files, you may
want to replace them with a single migration file that achieves the same thing.
You may want this because:

- creating a new dev or test database and applying migrations will be faster if
there are fewer migrations to run.
- having so many migration files makes it annoying to add new migrations
- having so many migration files gives lots of out-of-date results when
searching for sql tables/views/definitions.

This process will involve manually updating the migrations table of your
staging/production databases. Your coworkers will need to recreate their
development databases or manually update their migration state with the same
commands used in staging/production. Make sure to coordinate carefully with your
team and give plenty of heads up beforehand. This should be an infrequent
procedure.

Start by replacing your migrations with the output of `pgmigrate dump`.  This
can be done in a pull request just like any other change.

- Apply all current migrations to your dev/local database and verify that they were applied:
```bash
export PGM_MIGRATIONS="./migrations"
pgmigrate apply
pgmigrate verify
```
- Remove all existing migration files:
```bash
rm migrations/*.sql
```
- Dump the current schema as a new migration:
```bash
pgmigrate dump -o migrations/00001_squash_on_2023_07_02.sql
```

This "squash" migration does the exact same thing as all the migration files
that it replaced, which is the goal! But before you can deploy and run
migrations, you will need to manually mark this migration as having already been
applied. Otherwise, pgmigrate would attempt to apply it, and that almost
certainly wouldn't work. The commands below use `$PROD` to reference the
connection string for the database you are manually modifying, but you will need
to do this on every database for which you manage migrations.

- Double-check that the schema dumped from production is the exact same as the
squash migration file. If there are any differences in these two files, DO NOT
continue with the rest of this process. You will need to figure out why your
production database schema is different than that described by your migrations.
If necessary, please report a bug or issue on Github if pgmigrate is the reason
for the difference.
```bash
mkdir -p tmp
pgmigrate --database $PROD dump -o tmp/prod-schema.sql
# This should result in no differences being printed. If you see any
# differences, please abort this process.
diff migrations/00001_squash_on_2023_07_02.sql tmp/prod-schema.sql
rm tmp/prod-schema.sql
```
- Remove the records of all previous migrations having been applied.
```bash
# DANGER: Removes all migration records from the database
pgmigrate --database $PROD ops mark-unapplied --all
```
- Mark this migration as having been applied
```bash
# DANGER: marks all migrations in the directory (only our squash migration in
# this case) as having been applied without actually running the migrations.
pgmigrate --database $PROD ops mark-applied --all
```
- Check that the migration plan is empty, the result should show no migrations
need to be applied.
```bash
pgmigrate --database $PROD plan
```
- Verify the migrations state, should show no errors or problems.
```bash
pgmigrate --database $PROD verify
```

## ERROR: prepared statement "stmtcache_..." already exists (SQLSTATE 42P05)
If you're using the `pgmigrate` CLI and you see an error like this:

```
error: hasMigrationsTable: ERROR: prepared statement "stmtcache_19cfd54753d282685a62119ed71c7d6c9a2acfa4aa0d34ad" already exists (SQLSTATE 42P05)
```

you can fix the issue by adding a parameter to your database connection string to change how `pgmigrate` caches statements:

```yaml
# before
database: "postgresql://user:password@host.provider.com:6543/postgres"
# after
database: "postgresql://user:password@host.provider.com:6543/postgres?default_query_exec_mode=describe_exec"
```  

`pgmigrate` uses the on [`jackc/pgx`](https://github.com/jackc/pgx/) library to
connect to Postgres databases.  This library defaults to fairly aggressives
statement caching which is unfortunately not compatible with Pgbouncer or other
poolers. If you've seen the error above, you're most likely connecting through a
pooler like Pgbouncer.

The solution is to pass a `default_query_exec_mode=exec` connection string parameter, which
`jackc/pgx` will use to configure its statement caching behavior. [The documentation](https://pkg.go.dev/github.com/jackc/pgx/v5#QueryExecMode)
and [the connection parsing code](https://github.com/jackc/pgx/blob/fd0c65478e18be837b77c7ef24d7220f50540d49/conn.go#L194) describe the available
options, but `exec` should work by default. 

As of v0.1.0, the CLI will automatically add this query parameter for you if you
haven't already specified a statement caching mode.

## Configuring Postgres Timeouts for Locks, Statements, and Transactions

`pgmigrate` supports configuring various timeouts through the `postgres://...` connection string, just like any other Postgres client. I recommend you make sure to configure these options to prevent your migrations from hanging indefinitely, which potentially could impact your existing software also attempting to serve customer requests from the same database.

Depending on the version of Postgres your server is running, you may need to specify slightly different parameter names. No matter which version, you can do so in the URL passed via the `--database` CLI flag or via the `database:` value in the `.pgmigrate.yaml` config file.

```
postgres://user:password@host:port/dbname?statement_timeout=1000&lock_timeout=100&transaction_timeout=3000
```

[You can see the current list of supported timeout options here, at the Postgres docs](https://www.postgresql.org/docs/current/runtime-config-client.html). As of [Postgres 17](https://www.postgresql.org/docs/17/runtime-config-client.html), the options are:

- `statement_timeout` (integer):
  > Abort any statement that takes more than the specified amount of time. If log_min_error_statement is set to ERROR or lower, the statement that timed out will also be logged. If this value is specified without units, it is taken as milliseconds. A value of zero (the default) disables the timeout.
- `transaction_timeout` (integer):
  > Terminate any session that spans longer than the specified amount of time in a transaction. The limit applies both to explicit transactions (started with BEGIN) and to an implicitly started transaction corresponding to a single statement. If this value is specified without units, it is taken as milliseconds. A value of zero (the default) disables the timeout.
- `lock_timeout` (integer):
  > Abort any statement that waits longer than the specified amount of time while attempting to acquire a lock on a table, index, row, or other database object. The time limit applies separately to each lock acquisition attempt. The limit applies both to explicit locking requests (such as LOCK TABLE, or SELECT FOR UPDATE without NOWAIT) and to implicitly-acquired locks. If this value is specified without units, it is taken as milliseconds. A value of zero (the default) disables the timeout.
- `idle_in_transaction_session_timeout` (integer):
  > Terminate any session that has been idle (that is, waiting for a client query) within an open transaction for longer than the specified amount of time. If this value is specified without units, it is taken as milliseconds. A value of zero (the default) disables the timeout.
- `idle_session_timeout` (integer):
  > Terminate any session that has been idle (that is, waiting for a client query), but not within an open transaction, for longer than the specified amount of time. If this value is specified without units, it is taken as milliseconds. A value of zero (the default) disables the timeout.

Remember that when `pgmigrate` connects to your database, it will attempt to acquire a session lock (so that if you have multiple app servers starting up at once, only one of them runs the migrations), and then once it has a connection with that session lock, it will run each pending migration in its own transaction.

Most likely you'll want to set the `transaction_timeout` option in order to guard against the case where a migration takes an unexpectedly long amount of time and preventing Postgres from serving requests by your existing app servers.

Because each migration is run inside of its own transaction, you can always modify these timeouts for a specific migration by adding `SET LOCAL` commands to the beginning of the migration file. Be very careful to use `SET LOCAL` (which updates the configuration values for the current transaction) rather than `SET`, which updates the configuration values for the current connection. For more information, [see the Postgres docs on `SET`](https://www.postgresql.org/docs/current/sql-set.html).

# Acknowledgements

I'd like to thank and acknowledge:

- All existing migration libraries for inspiration.
- [djrobstep](https://github.com/djrobstep)'s
  [schemainspect](https://github.com/djrobstep/schemainspect) and
  [migra](https://github.com/djrobstep/migra) projects, for the queries used to
  implement `pgmigrate dump`.
- The backend team at Pipe for helping test and validate this project's
  assumptions, utility, and implementation.
