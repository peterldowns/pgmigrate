// migrations contains example migration data that is used in tests.
package migrations

import "embed"

// FS is an embedded filesystem that contains the example migrations at its
// root.
//
//go:embed *.sql
var FS embed.FS
