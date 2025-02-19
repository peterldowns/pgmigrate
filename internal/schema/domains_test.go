package schema_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestLoadDomainsSucceedsWithoutAnyDomains(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schemas: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		domains, err := schema.LoadDomains(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, 0, len(domains))
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadDomainResultIsStable(t *testing.T) {
	t.Parallel()
	original := query(`--sql
create domain score double precision check (value >= 0 and value <= 100);
	`)
	result := query(`--sql
CREATE DOMAIN public.score AS double precision
CHECK (VALUE >= 0::double precision AND VALUE <= 100::double precision);
	`)
	// original should parse to result
	checkDomain(t, original, result)
	// result should parse to itself
	checkDomain(t, result, result)
}

func TestLoadDomainWithAllOptions(t *testing.T) {
	t.Parallel()
	definition := query(`--sql
CREATE DOMAIN public."US_Postal_Code" AS text
COLLATE "en_US"
DEFAULT 'missing'::text
CHECK (VALUE <> 'garbage'::text)
NOT NULL;
	`)
	checkDomain(t, definition, definition)
}

func TestDependentDomains(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	config := schema.Config{
		Schemas: []string{"public"},
		Dependencies: map[string][]string{
			"ddd": {"zzz"},
			"ccc": {"zzz"},
			"aaa": {"ccc"},
			"qqq": {"vvv"},
		},
	}
	definition := query(`--sql
CREATE DOMAIN public.zzz AS text DEFAULT 'zzz'::text;
CREATE DOMAIN public.vvv AS text;
CREATE DOMAIN public.ttt AS text;
CREATE DOMAIN public.xxx AS text;
CREATE DOMAIN public.ddd AS zzz DEFAULT 'ddd'::text;
CREATE DOMAIN public.ccc AS zzz DEFAULT 'ccc'::text;
CREATE DOMAIN public.aaa AS ccc CHECK (value <> 'garbage'::text) NOT NULL;
CREATE DOMAIN public.qqq AS vvv DEFAULT 'qqq'::text;
	`)
	var result *schema.Schema
	assert.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		_, err := db.ExecContext(ctx, definition)
		if err != nil {
			return fmt.Errorf("failed to create: %w", err)
		}
		result, err = schema.Parse(config, db)
		return err
	}))
	orderedNames := []string{}
	for _, domain := range result.Domains {
		orderedNames = append(orderedNames, domain.Name)
	}
	assert.Equal(t, []string{
		"zzz", // <- ccc <- aaa
		"ccc", // <- aaa
		"aaa", // -> ccc -> zzz
		"ddd", // -> zzz
		"vvv", // <- qqq
		"qqq", // -> vvv
		"ttt", //
		"xxx", //
	}, orderedNames)
}

func checkDomain(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.Config{Schemas: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, definition); err != nil {
			return err
		}
		domains, err := schema.LoadDomains(config, db)
		if err != nil {
			return err
		}
		if check.Equal(t, 1, len(domains)) {
			parsed := domains[0].String()
			if !check.Equal(t, result, parsed) {
				t.Logf("expected\n%s", result)
				t.Logf(" received\n%s", parsed)
			}
		}
		return nil
	})
	assert.Nil(t, err)
}
