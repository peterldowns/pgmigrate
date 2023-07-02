package schema

import (
	"database/sql"
)

// Column represents a column in a [Table] or a [View], and will never exist "on
// its own".
type Column struct {
	// These fields are read from the database
	BelongsTo        int // The name of the [Table] or [View] that owns this Column.
	Number           int // The position of this Column within its owners full set of columns. The first column is number 0.
	Name             string
	NotNull          bool
	DataType         string // The Postgres data type of this column
	IsIdentity       bool
	IsIdentityAlways bool           // If True, then IsIdentity is also True.
	IsGenerated      bool           // If True, then IsIdentity is False and IsIdentityAlways is False.
	Collation        sql.NullString // The collation rules for this column, if any.
	DefaultDef       sql.NullString // The default definition for this column, if any.
	Comment          sql.NullString // The comment on this column, if any.
	// These fields will be populated during Parse()
	Sequence *Sequence // If set, the sequence associated with this column. Usually set in the case of primary keys or IS IDENTITY GENERATED ALWAYS.
}
