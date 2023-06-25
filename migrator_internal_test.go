package pgmigrate

import (
	"context"
	"fmt"
	"testing"
)

func TestLoggingSucceedsWithNilLogger(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migrator := NewMigrator(nil)

	migrator.log(ctx, LogLevelInfo, "hello", LogField{Key: "location", Value: "world"})
	migrator.log(ctx, LogLevelDebug, "hello", LogField{Key: "location", Value: "world"})
	migrator.log(ctx, LogLevelError, "hello", LogField{Key: "location", Value: "world"})

	migrator.debug(ctx, "hello", LogField{Key: "location", Value: "world"})
	migrator.info(ctx, "hello", LogField{Key: "location", Value: "world"})
	migrator.error(ctx, fmt.Errorf("new error"), "hello", LogField{Key: "location", Value: "world"})
}
