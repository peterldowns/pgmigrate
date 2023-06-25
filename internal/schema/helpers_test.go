package schema

import (
	"testing"

	"github.com/peterldowns/testy/check"
)

func TestIdentifiersDoesntQuoteLowered(t *testing.T) {
	t.Parallel()
	cases := []string{
		"hello",
		"hello_world",
		"hello-world",
		"1-2-3-4",
		"∆_unicode",
	}
	for _, tc := range cases {
		check.Equal(t, tc, identifiers(tc))
	}
}

func TestIdentifiersQuotesMixedCase(t *testing.T) {
	t.Parallel()
	check.Equal(t, `public."hello_WORLD"`, identifiers("public", `hello_WORLD`))
}
