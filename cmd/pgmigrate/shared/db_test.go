package shared

import (
	"testing"

	"github.com/peterldowns/testy/check"
)

func TestSetDefaultStatementCachingParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input    string
		expected string
	}{
		{
			// when default_query_exec_mode is omitted, it gets added
			input:    "postgresql://user:pass@host.provider.com:6543/postgres",
			expected: "postgresql://user:pass@host.provider.com:6543/postgres?default_query_exec_mode=exec",
		},
		{
			// when default_query_exec_mode is already included, it is left alone
			input:    "postgresql://user:pass@host.provider.com:6543/postgres?default_query_exec_mode=describe_exec",
			expected: "postgresql://user:pass@host.provider.com:6543/postgres?default_query_exec_mode=describe_exec",
		},
		{
			// other query parameters are left unchanged (but they are re-ordered alphabetically, and url-escaped)
			input:    "postgresql://user:pass@host.provider.com:6543/postgres?foo=bar,baz&age=12&sslmode=disable",
			expected: "postgresql://user:pass@host.provider.com:6543/postgres?age=12&default_query_exec_mode=exec&foo=bar%2Cbaz&sslmode=disable",
		},
	} {
		output, err := setDefaultStatementCachingParameter(tc.input)
		check.Nil(t, err)
		check.Equal(t, tc.expected, output)
	}
}
