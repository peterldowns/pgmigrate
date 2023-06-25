package shared

import "os"

var Flags struct { //nolint:gochecknoglobals
	LogFormat  *string // see logger.go
	Database   *string // see root.go
	Migrations *string // see root.go
}

func GetLogFormat() Variable[LogFormat] {
	return NewVariable(
		"log-format",
		LogFormat(*Flags.LogFormat),
		LogFormat(os.Getenv("PGM_LOG_FORMAT")),
		LogFormatText, // default
	)
}

func GetDatabase() Variable[string] {
	return NewVariable(
		"database",
		*Flags.Database,
		os.Getenv("PGM_DATABASE"),
	)
}

func GetMigrations() Variable[string] {
	return NewVariable(
		"migrations",
		*Flags.Migrations,
		os.Getenv("PGM_MIGRATIONS"),
	)
}
