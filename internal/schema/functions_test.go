package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestLoadFunctionsWithoutAny(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		functions, err := schema.LoadFunctions(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, []*schema.Function{}, functions)
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadFunctionResultIsStable(t *testing.T) {
	t.Parallel()
	original := query(`--sql
create function public."MixedCaseExample"(i int)
returns integer as $$
select i + 1
$$ language SQL;
	`)
	result := query(`--sql
CREATE OR REPLACE FUNCTION public."MixedCaseExample"(i integer)
 RETURNS integer
 LANGUAGE sql
AS $function$
select i + 1
$function$
;
	`)
	checkFunction(t, original, result)
	checkFunction(t, result, result)
}

func TestLoadFunctionLowerCaseNotQuoted(t *testing.T) {
	t.Parallel()
	def := query(`--sql
CREATE OR REPLACE FUNCTION public.dummy_len_plus_one(t text)
 RETURNS integer
 LANGUAGE sql
AS $function$
select bit_length(t) + 1
$function$
;
	`)
	checkFunction(t, def, def)
}

func TestLoadFunctionParsesAllAttributes(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	def := query(`--sql

CREATE OR REPLACE FUNCTION public.b58_encode(input bigint, suffix text)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE STRICT
 PARALLEL SAFE
AS $function$
DECLARE
    base constant INT := 58;
    alphabet constant TEXT[] := string_to_array('123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz', null);

    result TEXT := '';
    remainder INT;
    temp BIGINT;
begin
    IF input < 0 THEN
      RAISE EXCEPTION 'negative bigints are not supported';
    END IF;

    temp := input;

    LOOP
      remainder := temp % base;
      temp := temp / base;
      result := alphabet[remainder + 1] || result;

      exit when temp <= 0;
    END LOOP;

    RETURN result || suffix;
end;
$function$
;

	`)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.Exec(def); err != nil {
			return err
		}
		functions, err := schema.LoadFunctions(config, db)
		if err != nil {
			return err
		}
		if !check.Equal(t, 1, len(functions)) {
			return nil
		}
		parsed := functions[0]
		check.Equal(t, "public", parsed.Schema)
		check.Equal(t, "b58_encode", parsed.Name)
		check.Equal(t, "plpgsql", parsed.Language)
		check.Equal(t, "func", parsed.Kind)
		check.Equal(t, "immutable", parsed.Volatility)
		check.Equal(t, "safe", parsed.Parallel)
		check.Equal(t, "invoker", parsed.Security)
		check.Equal(t, "text", parsed.ResultType)
		check.Equal(t, "input bigint, suffix text", parsed.ArgumentTypes)
		return nil
	})
	assert.Nil(t, err)
}

func checkFunction(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, definition); err != nil {
			return err
		}
		functions, err := schema.LoadFunctions(config, db)
		if err != nil {
			return err
		}
		if check.Equal(t, 1, len(functions)) {
			parsed := functions[0].String()
			if !check.Equal(t, result, parsed) {
				t.Logf("expected\n%s", result)
				t.Logf(" received\n%s", parsed)
			}
		}
		return nil
	})
	assert.Nil(t, err)
}
