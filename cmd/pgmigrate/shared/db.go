package shared

import (
	"database/sql"
	"fmt"
	"net/url"

	// pgx driver
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
)

func OpenDB() (*sql.DB, error) {
	dbVar := State.Database()
	if err := Validate(dbVar); err != nil {
		return nil, err
	}
	dbStr, err := setDefaultStatementCachingParameter(dbVar.Value())
	if err != nil {
		return nil, err
	}
	return sql.Open("pgx", dbStr)
}

// If the pgmigrate user has not explicitly specified a pgx statement caching
// parameter in their connection string, set it to "exec", which will work
// correctly even when connecting to bouncers/poolers like Pgbouncer. If we
// don't do this, the default value pgx chooses is "cache_statement", which
// BREAKS when you connect to a pooler. Why pgx has a default value that breaks
// with poolers (very common!) and poorly documented, I can't say.
func setDefaultStatementCachingParameter(connstr string) (string, error) {
	eurl, err := url.Parse(connstr)
	if err != nil {
		return "", fmt.Errorf("failed to parse 'database' URL: %w", err)
	}
	query := eurl.Query()
	// hardcoded query parameter name comes from the pgx code:
	// https://github.com/jackc/pgx/blob/672c4a3a24849b1f34857817e6ed76f6581bbe90/conn.go#L191
	queryModeParam := "default_query_exec_mode"
	// hardcoded value "exec" comes from the pgx code:
	// https://github.com/jackc/pgx/blob/fd0c65478e18be837b77c7ef24d7220f50540d49/conn.go#L200
	execModeValue := "exec"
	if !query.Has(queryModeParam) {
		// The meaning is described by the documentation:
		// https://pkg.go.dev/github.com/jackc/pgx/v5#QueryExecMode
		//
		// > Assume the PostgreSQL query parameter types based on the Go type of
		// > the arguments. This uses the extended protocol with text formatted
		// > parameters and results. Queries are executed in a single round trip.
		// > Type mappings can be registered with
		// > pgtype.Map.RegisterDefaultPgType. Queries will be rejected that have
		// > arguments that are unregistered or ambiguous. e.g. A
		// > map[string]string may have the PostgreSQL type json or hstore. Modes
		// > that know the PostgreSQL type can use a map[string]string directly as
		// > an argument. This mode cannot.
		//
		// Although it doesn't say it explicitly, other modes are described as
		// NOT working with connection bouncers/poolers, but this one does work.
		query.Add(queryModeParam, execModeValue)
	}
	eurl.RawQuery = query.Encode()
	return eurl.String(), nil
}
