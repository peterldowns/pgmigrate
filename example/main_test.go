package main

import (
	"database/sql"
	"testing"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/pgmigrator"
	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgmigrate"
)

// This is a helper function to open a connection to a unique, fully-isolated,
// fully-migrated database that will be deleted when the test is done.
//
// For more information, see https://github.com/peterldowns/pgtestdb
func newDB(t *testing.T) *sql.DB {
	t.Helper()
	logger := pgmigrate.NewTestLogger(t)
	pgm, err := pgmigrator.New(migrationsFS, pgmigrator.WithLogger(logger))
	assert.Nil(t, err)
	db := pgtestdb.New(t, pgtestdb.Config{
		DriverName: "postgres",
		Host:       "localhost",
		User:       "appuser",
		Database:   "testdb",
		Password:   "verysecret",
		Port:       "5436",
		Options:    "sslmode=disable",
	}, pgm)
	assert.NotEqual(t, nil, db)
	return db
}

// Tests that newDB() works and the new database is queryable.
func TestWithMigratedDatabase(t *testing.T) {
	t.Parallel()
	db := newDB(t)

	row := db.QueryRow("select 'hello world'")
	assert.Nil(t, row.Err())

	var message string
	err := row.Scan(&message)
	assert.Nil(t, err)

	assert.Equal(t, "hello world", message)
}

// Tests that newDB() works and the new database has the expected schema, which
// is the result of applying the migrations.
func TestApplicationHasNoDataButSchemaIsCorrect(t *testing.T) {
	t.Parallel()
	db := newDB(t)
	var count int

	// 0 companies
	row := db.QueryRow("select count(*) from companies")
	assert.Nil(t, row.Err())
	assert.Nil(t, row.Scan(&count))
	assert.Equal(t, 0, count)

	// 0 users
	row = db.QueryRow("select count(*) from companies")
	assert.Nil(t, row.Err())
	assert.Nil(t, row.Scan(&count))
	assert.Equal(t, 0, count)

	// 0 blobs
	row = db.QueryRow("select count(*) from blobs")
	assert.Nil(t, row.Err())
	assert.Nil(t, row.Scan(&count))
	assert.Equal(t, 0, count)

	// 3 review states
	rows, err := db.Query("select value from blob_type_enum")
	assert.Nil(t, err)
	var types []string
	for rows.Next() {
		var blobtype string
		assert.Nil(t, rows.Scan(&blobtype))
		types = append(types, blobtype)
	}
	assert.Equal(t, []string{"pending_review", "approved", "rejected"}, types)
}
