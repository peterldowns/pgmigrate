package pgtools_test

import (
	"testing"

	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

func TestLiteral(t *testing.T) {
	t.Parallel()
	check.Equal(t, `'hello'`, pgtools.Literal(`hello`))
	check.Equal(t, `'''hello'''`, pgtools.Literal(`'hello'`))
	check.Equal(t, `'"hello"'`, pgtools.Literal(`"hello"`))
	check.Equal(t, ` E'abc\\def'`, pgtools.Literal(`abc\def`)) // literal \, not an escape character
	check.Equal(t, `'schema.table'`, pgtools.Literal(`schema.table`))
}

// These tests are designed to show how the Identifier function behaves with
// various inputs that expect to represent typical use cases.
func TestIdentifierExpectedInputs(t *testing.T) {
	t.Parallel()
	// a plain identifier without any qualification
	check.Equal(t, `hello`, pgtools.Identifier(`hello`))
	// a fully-dotted identifier, for use in CREATE TABLE or other DDL.
	check.Equal(t, `someschema.sometable`, pgtools.Identifier(`someschema.sometable`))
	// the identifier "table" is a reserved keyword and requires quoting.
	check.Equal(t, `schema."table"`, pgtools.Identifier(`schema.table`))
	check.Equal(t, `schema."table"`, pgtools.Identifier(`schema`, `table`))
	// Cats contains upper-case and therefore requires quoting.
	check.Equal(t, `schema."Cats"`, pgtools.Identifier(`schema.Cats`))
	check.Equal(t, `schema."Cats"`, pgtools.Identifier(`schema`, `Cats`))
	// CATS is completely upper-case and therefore requires quoting.
	check.Equal(t, `schema."CATS"`, pgtools.Identifier(`schema.CATS`))
	check.Equal(t, `schema."CATS"`, pgtools.Identifier(`schema`, `CATS`))
	// user is reserved and thefore requires quoting.
	check.Equal(t, `"user"."user"`, pgtools.Identifier(`user.user`))
	check.Equal(t, `"user"."user"`, pgtools.Identifier(`user`, `user`))
}

// These tests are designed to show how the Identifier function behaves with
// various inputs that are not typical, but could theoretically be passed to the
// function.
func TestIdentifierGarbageInputs(t *testing.T) {
	t.Parallel()
	// any literal single quote ' is not escaped
	check.Equal(t, `some'ide'ntifier`, pgtools.Identifier(`some'ide'ntifier`))
	// any literal double quote " gets escaped by doubling `"` -> `""`, which
	// requires surrounding the part with quotes as well.
	check.Equal(t, `"""schema"""."""tablename"""`, pgtools.Identifier(`"schema"."tablename"`))
	check.Equal(t, `"""schema"."tablename"""`, pgtools.Identifier(`"schema.tablename"`))
}

func TestParseTableName(t *testing.T) {
	t.Parallel()
	schema, tablename := pgtools.ParseTableName("users")
	check.Equal(t, "public", schema)
	check.Equal(t, "users", tablename)

	schema, tablename = pgtools.ParseTableName("custom.users")
	check.Equal(t, "custom", schema)
	check.Equal(t, "users", tablename)

	schema, tablename = pgtools.ParseTableName(".users")
	check.Equal(t, "", schema)
	check.Equal(t, "users", tablename)

	schema, tablename = pgtools.ParseTableName("a.b.c")
	check.Equal(t, "a", schema)
	check.Equal(t, "b.c", tablename)
}
