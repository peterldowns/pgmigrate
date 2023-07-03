| ⚠️   WARNING                      |
|:-------------------------------- |
| 🚧 This Is A Work In Progress 🚧 |

- [ ] Library
  - [ ] Docstrings on all public methods/functions
  - [ ] Tests for the schema handling stuff
  - [ ] Generally clean up the code
  - [ ] Improve error handling
    - [ ] wrap errors with constant messages
    - [ ] use an error library that includes stack traces
- [ ] CLI
  - some kind of tests!
  - Flake derivation
  - Homebrew install instructions
- [ ] Ops
  - [ ] cli versioning and releases
- [ ] Docs
  - [ ] comparisons to other migration frameworks
  - [ ] example of using pgtestdb
  - [ ] discussion of large/long-running migrations, migration ordering
  - [ ] how to check the dump for differences in CI
- [ ] Wishlist
  - [ ] make `*Result` diffable, allow generating migration from current state of database.
    - docs should say for now, just use [https://github.com/djrobstep/migra](https://github.com/djrobstep/migra)
  - [ ] some kind of linting
    - BEGIN/COMMIT/ROLLBACK
    - serial vs. identity
    - pks / fks with indexes
    - uppercase / mixed case
    - https://squawkhq.com/
    - https://github.com/sqlfluff/sqlfluff

# 🐽 pgmigrate

![Latest Version](https://badgers.space/badge/latest%20version/v0.0.3/blueviolet?corner_radius=m)
![Golang](https://badgers.space/badge/golang/1.18+/blue?corner_radius=m)


pgmigrate is a modern Postgres migrations CLI and golang library. It is designed
for use by high-velocity teams.

Major features:

- Applies any previously-unapplied migrations, in order &mdash; that's it.
- Each migration is applied within a transaction.
- Only "up" migrations, no "down" migrations (you don't want or need them.)
- Uses [Postgres advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) so it's safe to run in parallel.
- Compatible with [pgtestdb](https://github.com/peterldowns/pgtestdb) so database-backed tests are very fast.

# Documentation

- [This page, https://github.com/peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate)
- [The go.dev docs, pkg.go.dev/github.com/peterldowns/pgmigrate](https://pkg.go.dev/github.com/peterldowns/pgmigrate)

This page is the primary source for documentation. The code itself is supposed
to be well-organized, and each function has a meaningful docstring, so you
should be able to explore it quite easily using an LSP plugin, reading the
code, or clicking through the go.dev docs.

## Why do I want this?
You want pgmigrate because you just want migrations to be simpler and for them
to run successfully when you update your application. You want your whole team
to be able to understand a simple system with a few, well-documented invariants.

You don't want to worry about rebasing or updating your PR just to bump the
sequence number of your migration file. You don't want to break main because you
and a coworker both chose the same sequence number for your migration. You just
want your team to merge migrations and keep shipping.

You want to be able to fix migration failures and redeploy without any manual
intervention. When you merge the fix, pgmigrate will run the new migration
without complaining. (Believe it or not, but some migration frameworks set a
"dirty" bit when migrations fail, which means that you have to manually psql
into production and set `dirty = 'f'` before you can deploy again.)

You want to run migrations as an init step as part of your modern,
containerized, deployment workflow.

You want to use the same migrations logic as an embedded golang library, as a
standalone cli, or as a pre-built container.

Finally, you never write or think about down migrations again in your life! You
aren't using them, they aren't useful, it's 2023 we do not need them!
## How does it work?

pgmigrate has relatively simple invariants and behavior compared to other
migration libraries:

- A migration is a file whose name ends in `.sql`. The part before the extension
is its unique ID.
- All migrations are "up" migrations, there is no such thing as a "down"
migration.
- The migrations table is a table that pgmigrate uses to track which migrations
have been applied. It has the following schema:
  - `id (text not null)`: the ID of the migration
  - `checksum (text not null)`: the MD5() hash of the contents of the migration when it was applied.
  - `execution_time_in_millis (integer not null)`: how long it took to apply the migration, in milliseconds.
  - `applied_at (timestamp with time zone not null)`: the time at which the migration was finished applying and this row was inserted.
- A plan is an ordered list of previously-unapplied migrations. The migrations
are sorted by their IDs, in ascending lexicographical/alphabetical order. This
is the same order that you get when you use `ls` or `sort`.
- Each time migrations are applied, pgmigrate calculates the plan, then attempts
to apply each migration one at a time.
- To apply a migration, pgmigrate:
  - Begins a transaction.
  - Runs the migration SQL.
  - Creates and inserts a new row in the migrations table.
  - Commits the transaction.
- Because each migration is applied in an explicit transaction, you **must not**
use `BEGIN`/`COMMIT`/`ROLLBACK` within your migration files.
- Any error when applying a migration will result in an immediate failure. If
there are other migrations later in the plan, they will not be applied.
- If and only if a migration is applied successfully, there will be a row in the
`migrations` table containing its ID.
- pgmigrate uses [Postgres advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) to ensure that only once instance
is attempting to run migrations at any point in time. It is safe to run
migrations as part of an init container, when your binary starts, or any other
parallel way.
- After a migration has been applied you should never edit the file's contents.
If you do, pgmigrate will warn you that the hash of the migration differs from
the hash of the migration when it was applied.
- After a migration has been applied you should never delete the migration. If
you do, pgmigrate will warn you that a migration that had previously been
applied is no longer present.

## How should my team work with it?
### the migrations directory
Your team's repository should include a `migrations/` directory containing all known migrations.

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

### deploying and applying migrations
You should run pgmigrate with the latest migrations directory each time you
deploy. Assuming you're running in a modern cloud environment, you're most
likely doing rolling deployments where new instances of your application are
brought up before old ones are terminated. Therefore, make sure any new
migrations will result in a database state that the previous version of your
application (which will still be running as migrations are applied) can handle.
For more on this, see [the FAQ below](#).

Because pgmigrate uses advisory locks, you can roll out as many new instances of
your application as you'd like. Even if multiple instance attempt to run the
migrations at once, only one will acquire the lock and apply the migrations.
After its done, the other instances should see that there are no more migrations
to apply, and continue successfully.

Successfully running migrations should be a prerequisite to the new version of
your application starting up and accepting requests. 

### create new migrations
Unlike other migration libraries, *it is totally fine for migrations to have the
same prefix*! pgmigrate uses the full name of the file to identify a migration. This makes life a lot easier on a distributed, high-velocity team. Assuming you are using modern
git and integration tests, you can rely on those processes to prevent conflicting merges.

For instance, assuming the same migrations as above, you can create a new migration `0005_create_squirrels.sql`:

```sql
CREATE TABLE squirrels (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  species TEXT
);
```

One of your coworkers may also create a new migration at this point, and use the same number,
but do something different &mdash; they call it `0005_new_table_cows.sql`

```sql
CREATE TABLE cows (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tippable BOOL,
  is_tipped BOOL
);
```

This is totally fine! You can each merge your PRs, in any order, because they do not conflict.
Any sequence of merging and deploying and applying migrations will result in the same state, with both tables having been created and both migration files present in the folder:

```
migrations
├── 0001_cats.sql
├── 0003_dogs.sql
├── 0003_empty.sql
├── 0004_rm_me.sql
├── 0005_create_squirrels.sql
├── 0005_new_table_cows.sql
```

### preventing conflicts
You may have just asked yourself, wait, how could that be true? What if two PRs
merge SQL that conflicts? For instance, let's say two new tables get created,
one which deletes the `users` table and one which creates a new `houses` table
with a foreign key pointing to `users`. There's no way both of these migrations
could be safely applied, and the resulting database state could be different
depending on the order!

```
├── ...
├── 0006_aaaa_delete_users_table.sql
├── 0006_bbbb_new_table_with_foreign_key_to_users_table.sql
```

Or let's say two new PRs get merged, each of which renames the `user.name`
field. One which changes `user.name` to `user.full_name`, one which changes
`user.name` to `user.legal_name`.

```
├── ...
├── 0007_rename_user_name_to_fullname.sql
├── 0007_rename_user_name_to_legal_name.sql
```

Oh my god, isn't this a huge problem? Uh, no:

1. Once the second (conflicting) migration is merged, your CI tests should fail
because the migrations cannot be applied. Breaking main is annoying, but...
2. You will never intentionally do something like this. Even in distributed
teams, people coordinate work and it is exceedingly unlikely to have multiple
changes happening at once that have conflicting migrations.

Unconvinced? What if I pinky promised? OK, let's say you're still worried. Use
git to guarantee that you never have a problem by turning schema conflicts into
unmergeable file conflicts:

- Add a `schema.sql` file somewhere in your repo that contains a dump of your database schema generated with `pgmigrate dump -o schema.sql`
- Make an easy script for applying migrations and then dumping the resulting schema to `schema.sql` so developers can do it as they work.
- In CI, run the migrations and use that script to make sure `schema.sql` is up
to date. If running the script in CI causes any changes to the file, fail, and
ask the developer to redump the schema.

This will mean that `schema.sql` stays up to date as developers write new
migrations. If two developers write migrations that cause conflicting schema
updates, they won't be able to merge because it will be a git conflict.

## How to squash your migrations

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


# Library

## Install
* requires golang 1.18+ because it uses generics. 
* only depends on stdlib; all dependencies in the go.mod are for tests.
```bash
# library
go get github.com/peterldowns/pgmigrate@latest
```

## Usage

# CLI

## Install

```bash
go install github.com/peterldowns/pgmigrate/cli@latest
```

## Usage

# FAQ

# Acknowledgements

I'd like to thank and acknowledge:

- All existing migration libraries for inspiration.
- [djrobstep](https://github.com/djrobstep)'s
  [schemainspect](https://github.com/djrobstep/schemainspect) and
  [migra](https://github.com/djrobstep/migra) projects, for the queries used to
  implement `pgmigrate dump`.
- The backend team at Pipe for helping test and validate this project's
  assumptions, utility, and implementation.

# Notes

## `pgmigrate dump`
The goal of `pgmigrate dump --database $ORIGINAL > schema.sql` is for the resulting sql file to be:
  - usable: can `psql $NEW -f schema.sql` to create a new database with the same schema.
  - diffable: if there are migrations in different PRs/branches that will conflict with each other,
      diffing the generated schema.sql files from each branch should result in a merge conflict that
      cannot be automatically resolved.
  - roundtrippable: dumping `pgmigrate dump --database $NEW > schema.sql` will result in 0 changes.
  - customizable: you can include tables to dump values from (for enum tables) and you can explicitly
      add dependencies between objects that will be respected during the dump, to work around faulty
      dependency detection.


