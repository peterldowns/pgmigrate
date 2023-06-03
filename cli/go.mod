module github.com/peterldowns/pgmigrate/cli

go 1.18

replace github.com/peterldowns/pgmigrate => ../

require (
	github.com/jackc/pgx/v5 v5.3.1
	github.com/peterldowns/pgmigrate v0.0.1
	github.com/spf13/cobra v1.7.0
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)
