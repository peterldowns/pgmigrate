package pgmigrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/peterldowns/pgmigrate/logging"
)

func TestLoggingSucceedsWithNilLogger(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migrator := NewMigrator(nil)

	migrator.log(ctx, logging.LevelInfo, "hello", logging.Field{Key: "location", Value: "world"})
	migrator.log(ctx, logging.LevelDebug, "hello", logging.Field{Key: "location", Value: "world"})
	migrator.log(ctx, logging.LevelError, "hello", logging.Field{Key: "location", Value: "world"})

	migrator.debug(ctx, "hello", logging.Field{Key: "location", Value: "world"})
	migrator.info(ctx, "hello", logging.Field{Key: "location", Value: "world"})
	migrator.error(ctx, fmt.Errorf("new error"), "hello", logging.Field{Key: "location", Value: "world"})
}
