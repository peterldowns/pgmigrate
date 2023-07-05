# 🐽 pgmigrate

![Latest Version](https://badgers.space/badge/latest%20version/v0.0.4/blueviolet?corner_radius=m)
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

# CLI

## Install

#### Homebrew:
```bash
# TODO: not yet published
# install it
# brew install peterldowns/tap/pgmigrate
```

#### Docker:
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
#### Nix (flakes):
```bash
# run it
nix run github:peterldowns/pgmigrate -- --help
# install it
nix profile install --refresh github:peterldowns/pgmigrate
```

#### Manually download binaries
Visit [the latest Github release](https://github.com/peterldowns/pgmigrate/releases/latest) and pick the appropriate binary. Or, click one of the shortcuts here:
- [darwin-amd64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-darwin-amd64)
- [darwin-arm64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-darwin-arm64)
- [linux-amd64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-linux-amd64)
- [linux-arm64](https://github.com/peterldowns/pgmigrate/releases/latest/download/pgmigrate-linux-arm64)


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
      --table-name string   [PGM_TABLENAME] the table name to use to store migration records (default 'pgmigrate_migrations')
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
the library. Please read the cli help with `pgmigrate help <command>` and read
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

For instance let's say two new tables get created,

- one which deletes the `users` table
- one which creates a new `houses` table with a foreign key to `users`.

There's no way both of these migrations could be safely applied, and the
resulting database state could be different depending on the order!

```
├── ...
├── 0006_aaaa_delete_users_table.sql
├── 0006_bbbb_new_table_with_foreign_key_to_users_table.sql
```

You prevent these conflicts during CI by using pgmigrate to maintain an
up-to-date dump of your database schema:

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

# Acknowledgements

I'd like to thank and acknowledge:

- All existing migration libraries for inspiration.
- [djrobstep](https://github.com/djrobstep)'s
  [schemainspect](https://github.com/djrobstep/schemainspect) and
  [migra](https://github.com/djrobstep/migra) projects, for the queries used to
  implement `pgmigrate dump`.
- The backend team at Pipe for helping test and validate this project's
  assumptions, utility, and implementation.

# Future Work / TODOs

- [ ] Library
  - [ ] More tests for the schema handling stuff
  - [ ] Generally clean up the code
- [ ] Readme
  - [ ] example of using pgtestdb
  - [ ] discussion of large/long-running migrations
- [ ] Wishlist
  - [ ] make `*Result` diffable, allow generating migration from current state of database.
    - for now, just use [https://github.com/djrobstep/migra](https://github.com/djrobstep/migra)
  - [ ] some kind of built-inlinting
    - maybe using https://github.com/auxten/postgresql-parser
    - BEGIN/COMMIT/ROLLBACK
    - serial vs. identity
    - pks / fks with indexes
    - uppercase / mixed case
    - https://squawkhq.com/
    - https://github.com/sqlfluff/sqlfluff