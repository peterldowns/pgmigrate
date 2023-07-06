# example application

This example application is designed to help employees share json blobs with
each other. You can clone this application and use it as an example for playing
around with pgmigrate.

I recommend going through the cli quickstart and then checking out all the different
files.

### database

The application and the CLI will attempt to connect to a postgres database with this
connection string:

``` 
postgres://appuser:verysecret@localhost:5435/exampleapp
```

and tests will try to connect to a postgres database with this connection string:

```  
postgres://appuser:verysecret@localhost:5436/testdb
```

The example comes with a `docker-compose.yml` defining these two database servers,
you can start them with:

```shell
docker compose up -d appdb testdb
```

### application

The application is a dummy web server, it doesn't actually do anything.  The
`main.go` file is a good example of how to use pgmigrate as a library to run
migrations at application startup.

Run the application with:

```shell
go run .
```

Test the application with:

```shell
go test ./... -count=1 -race
```


### cli quickstart

Here's a bunch of commands to try to get a feel for the pgmigrate CLI tool.
You'll CRUD some migrations and use the most common commands. Please remember
that you can read docs and help for the commands with `pgmigrate help
<command>` or `pgmigrate <command> --help`. If you're still confused, or run
into any bugs, please open a Github issue.

```shell
# create the application database, which begins entirely empty
docker compose up -d appdb

# show the migrations that would be applied. this example directory
# has a `.pgmigrate.yaml` configuration file, but you can also
# specify the configuration with cli flags or environment variables.
#
# each of these commands is equivalent:
pgmigrate plan 
pgmigrate plan \
  --database "postgres://appuser:verysecret@localhost:5435/exampleapp" \
  --migrations ./migrations
pgmigrate \
  --database "postgres://appuser:verysecret@localhost:5435/exampleapp" \
  --migrations ./migrations \
  plan
PGM_DATABASE="postgres://appuser:verysecret@localhost:5435/exampleapp" PGM_MIGRATIONS="./migrations" pgmigrate plan

# show the current config, taking into account any cli flags or environment
# variables.
pgmigrate config

# show the currently-applied migrations. this should be empty
pgmigrate applied

# apply the migrations
pgmigrate migrate

# verify that migrations were applied correctly. this should be empty because
# there are no verification errors.
pgmigrate verify

# list all applied migrations
pgmigrate applied

# apply the migrations again, this time it should tell you
# that no migrations need to be applied
pgmigrate migrate

# create a new (empty) migration file
pgmigrate new --create --name silly_example

# show the plan, it should only contain the newly-created migration
pgmigrate plan

# apply migrations again
pgmigrate migrate

# edit the contents of the migration file to be something different
echo "-- this new comment changes the contents of the file and its hash" >> ./migrations/00004_silly_example.sql

# attempt to verify migrations. you should see a warning that the migration
# file's contents have changed
pgmigrate verify

# show the plan. even though 00004_silly_example.sql has changed since it was
# applied, pgmigrate will not plan to apply it again.
pgmigrate plan

# update the stored checksum in the migration table to silence the warning.
# verifying should emit no warnings.
pgmigrate ops recalculate-checksum --id 00004_silly_example
pgmigrate verify

# remove 00004_silly_example.sql and then verify migrations. you should
# see a warning that the migration has been removed.
rm migrations/00004_silly_example.sql
pgmigrate verify

# manually remove the record of the migration having been applied.
# verifying should emit no warnings.
pgmigrate ops mark-unapplied --id 00004_silly_example
pgmigrate verify

# dump the current schema from the `appdb` into the `schema.sql` file:
# (see the `.pgmigrate.yaml` for the dump configuration)
pgmigrate dump

# create a broken migration file. the first part is valid sql but the second
# part is riddled with syntax errors.  as a result, when we apply this
# migration neither part will # take effect.
cat > ./migrations/00004_broken.sql << EOF
-- valid
CREATE TABLE dogs (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  name text UNIQUE NOT NULL,
  very_good bool NOT NULL DEFAULT true
);
-- this is just straight up not valid SQL
CREATE TABLE this_has_syntax_errors;;; [
  id bigint
  foo bar
  baz blap
EOF
cat > ./migrations/00005_valid.sql << EOF
CREATE TABLE cats (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  name text UNIQUE NOT NULL,
  so_pretty_and_elegant bool NOT NULL DEFAULT true
);
INSERT INTO cats (name) VALUES
('daisy'),
('sunny'),
('charlie');
EOF

# show the plan, which should be to apply both migrations
pgmigrate plan

# apply migrations, this should fail on the broken one
pgmigrate migrate

# dump the schema again.
#
# you should see that the dogs table from the broken migration did not get
# created. this is because the migration is run within a transaction, so the
# syntax errors caused the entire transaction to roll back.
#
# you should also see that the cats table from the valid migration did not
# get created. this is because the broken migration prevented it from being
# run.
pgmigrate dump -o schema.sql
cat schema.sql

# show that both migrations are still in the plan, since neither
# was successfully applied
pgmigrate plan

# fix the broken migration
cat > ./migrations/00004_broken.sql << EOF
CREATE TABLE dogs (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  name text UNIQUE NOT NULL,
  very_good bool NOT NULL DEFAULT true
);
EOF

# apply the migrations and dump the schema. there should be no issues
# and both the dogs and cats tables should exist.
pgmigrate apply
pgmigrate dump -o schema.sql
```

