package schema

import (
	"testing"

	"github.com/peterldowns/testy/check"
)

func TestIdentifiersDoesntQuote(t *testing.T) {
	t.Parallel()
	cases := []string{
		`hello`,
		`hello_world`,
		`hello-world`,
		`1-2-3-4`,
		`∆_unicode`,
		`single'quotes'`,
	}
	for _, tc := range cases {
		check.Equal(t, tc, identifier(tc))
	}
}

func TestIdentifiersQuotes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expected string
		parts    []string
	}{
		{`public."hello_WORLD"`, []string{"public", `hello_WORLD`}},
		{`∆."foo""bar"`, []string{"∆", `foo"bar`}},
	}
	for _, tc := range cases {
		check.Equal(t, tc.expected, identifier(tc.parts...))
	}
}
