package pgtools_test

import (
	"testing"

	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

func TestQuoteTableExpectedInput(t *testing.T) {
	t.Parallel()
	// Designed use case: quoting the schema and table name for use in CREATE TABLE.
	check.Equal(t, `"schema"."table"`, pgtools.QuoteTableAndSchema(`schema.table`))
	check.Equal(t, `"table"`, pgtools.QuoteTableAndSchema(`table`))
}

func TestQuoteTableUnexpectedInput(t *testing.T) {
	t.Parallel()
	// Not designed to do anything else, but here's how it behaves when the input
	// already has quotes in it
	//
	// each single quote `"` gets escaped by doubling -> `""`,
	// and the schema and table names are both quoted.
	// (this is almost certainly not useful.)
	check.Equal(t, `"""schema"""."""table"""`, pgtools.QuoteTableAndSchema(`"schema"."table"`))
	check.Equal(t, `"""schema"."table"""`, pgtools.QuoteTableAndSchema(`"schema.table"`))
}

func TestQuoteLiteral(t *testing.T) {
	t.Parallel()
	check.Equal(t, `'hello'`, pgtools.QuoteLiteral(`hello`))
	check.Equal(t, `'''hello'''`, pgtools.QuoteLiteral(`'hello'`))
	check.Equal(t, `'"hello"'`, pgtools.QuoteLiteral(`"hello"`))
	check.Equal(t, ` E'abc\\def'`, pgtools.QuoteLiteral(`abc\def`)) // literal \, not an escape character
	check.Equal(t, `'schema.table'`, pgtools.QuoteLiteral(`schema.table`))
}

func TestQuoteIdentifier(t *testing.T) {
	t.Parallel()
	check.Equal(t, `"hello"`, pgtools.QuoteIdentifier(`hello`))
	check.Equal(t, `"'hello'"`, pgtools.QuoteIdentifier(`'hello'`))
	check.Equal(t, "\"`hello`\"", pgtools.QuoteIdentifier("`hello`"))
	check.Equal(t, `"schema.table"`, pgtools.QuoteIdentifier(`schema.table`))
}

func TestReservedKeywords(t *testing.T) {
	t.Parallel()
	check.Equal(t, "public.hello", pgtools.QuoteTableAndSchema("hello"))
}
