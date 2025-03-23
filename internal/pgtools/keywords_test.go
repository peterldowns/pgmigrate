package pgtools

import (
	"fmt"
	"strings"
	"testing"

	"github.com/peterldowns/testy/check"
)

// Sanity-check our reserved keywords map to make sure that any capitalization
// of a reserved keyword is correctly quoted.
func TestReservedKeywordsAreAlwaysQuotedRegardlessOfCase(t *testing.T) {
	t.Parallel()
	for keyword := range postgresKeywords {
		check.True(t, requiresQuoting(keyword))
		check.Equal(t, fmt.Sprintf(`"%s"`, keyword), Identifier(keyword))
		uppered := strings.ToUpper(keyword)
		check.True(t, requiresQuoting(uppered))
		check.Equal(t, fmt.Sprintf(`"%s"`, uppered), Identifier(uppered))
		mixed := strings.ToUpper(string(keyword[0])) + keyword[1:]
		check.True(t, requiresQuoting(mixed))
		check.Equal(t, fmt.Sprintf(`"%s"`, mixed), Identifier(mixed))
	}
}
