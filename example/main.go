package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	_ "github.com/lib/pq"

	"github.com/peterldowns/pgmigrate"
)

// This is a simplified example of an application that will run a web server.
// Like any application using pgmigrate, it starts by connecting to the
// database and running pgmigrate.Migrate. If this fails, it exits. If it
// succeeds, it continues to running the server.
//
// You do not need to run migrations directly in your application -- for
// instance, you could use a kubernetes init container, or some other kind of
// initialization step to run the migrations via Docker or CLI before starting
// your web server.  This is just one way to do it.
func main() {
	ctx := context.Background()
	logger := log.NewWithOptions(os.Stdout, log.Options{Formatter: log.TextFormatter})
	logger.Info("connecting to the database")
	db, err := sql.Open("postgres", "postgres://appuser:verysecret@localhost:5435/exampleapp?sslmode=disable")
	if err != nil {
		panic(err)
	}
	logger.Info("applying migrations")
	err = applyMigrations(ctx, db, logger)
	if err != nil {
		panic(err)
	}

	logger.Info("running the web server")
	runServer(ctx, db, logger)
}

// The migrations directory will be embedded into the application
// at build time. You can also ship your migration files next to the
// application and have it read them from disk. For more information,
// read the docs for pgmigrate.Load.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Does what it says!
func applyMigrations(ctx context.Context, db *sql.DB, logger *log.Logger) error {
	verrs, err := pgmigrate.Migrate(ctx, db, migrationsFS, logAdapter{logger})
	if err != nil {
		return err
	}
	for _, verr := range verrs {
		var vals []any
		for key, val := range verr.Fields {
			vals = append(vals, key, val)
		}
		logger.Warn(verr.Message, vals...)
	}
	return nil
}

// This is a fake, it just pretends to start a web server. It actually does
// nothing because this is just an example application to show off how
// migrations work.
func runServer(_ context.Context, _ *sql.DB, logger *log.Logger) {
	fmt.Println("hello, world")
	fmt.Println("(this isn't actually a working application but please pretend it is)")
	for { // infinite loop, cancellable with ctrl-c
		time.Sleep(5 * time.Second)
		logger.Info("tick")
	}
}

// This is an unavoidable annoyance -- in order to make pgmigrate work with
// various different logging libraries (zap, slog, logrus, etc.) it requires
// you to adapt your logger to its interface. This wraps the charm/log logger
// so that we can see the pgmigrate logs when the app starts up.
type logAdapter struct {
	*log.Logger
}

func (l logAdapter) Log(
	_ context.Context,
	level pgmigrate.LogLevel,
	msg string,
	fields ...pgmigrate.LogField,
) {
	args := make([]any, 0, 2*len(fields))
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	switch level {
	case pgmigrate.LogLevelDebug:
		l.Logger.Debug(msg, args...)
	case pgmigrate.LogLevelInfo:
		l.Logger.Info(msg, args...)
	case pgmigrate.LogLevelError:
		l.Logger.Error(msg, args...)
	case pgmigrate.LogLevelWarning:
		l.Logger.Warn(msg, args...)
	}
}
