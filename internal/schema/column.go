package schema

import (
	"database/sql"
	"fmt"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

// See Tables, Views

type Column struct {
	BelongsTo        int
	Number           int
	Name             string
	NotNull          bool
	DataType         string
	IsIdentity       bool
	IsIdentityAlways bool
	IsGenerated      bool
	Collation        sql.NullString
	DefaultDef       sql.NullString
	Comment          sql.NullString
	Sequence         *Sequence
}

func (c Column) SortKey() int {
	return c.Number
}

func (Column) DependsOn() []string {
	return nil
}

// TODO: extract to helper
func (c Column) ToString(tableName string, primaryKey bool, unique bool, references string) string { //nolint:revive // stupid fucking control flag bullshit
	def := fmt.Sprintf("%s %s", pgtools.QuoteIdentifier(c.Name), c.DataType)
	if primaryKey {
		def = fmt.Sprintf("%s PRIMARY KEY", def)
	} else if unique {
		def = fmt.Sprintf("%s UNIQUE", def)
	}
	if c.NotNull {
		def = fmt.Sprintf("%s NOT NULL", def)
	}
	defaultDef := ""
	if c.DefaultDef.Valid {
		defaultDef = c.DefaultDef.String
	}
	// if defaultDef == "" && c.Sequence != nil && (c.Sequence.IsIdentity || c.Sequence.IsIdentityAlways) {
	// 	defaultDef = fmt.Sprintf("nextval(%s::regclass)", pgtools.QuoteIdentifier(c.Sequence.Name))
	// }
	if c.IsIdentity {
		var identityType string
		if c.IsIdentityAlways {
			identityType = "ALWAYS"
		} else {
			identityType = "BY DEFAULT"
		}
		def = fmt.Sprintf("%s GENERATED %s AS IDENTITY", def, identityType)
	}
	if c.IsGenerated { // IsIdentity and IsGenerated are never both true
		def = fmt.Sprintf("%s GENERATED ALWAYS AS (%s) STORED", def, defaultDef)
	} else if defaultDef != "" {
		def = fmt.Sprintf("%s DEFAULT %s", def, defaultDef)
	}
	if references != "" {
		def = fmt.Sprintf("%s %s", def, references)
	}
	return def
}

func (c Column) String() string {
	def := fmt.Sprintf("%s %s", pgtools.QuoteIdentifier(c.Name), c.DataType)
	if c.IsIdentity {
		var identityType string
		if c.IsIdentityAlways {
			identityType = "ALWAYS"
		} else {
			identityType = "BY DEFAULT"
		}
		def = fmt.Sprintf("%s GENERATED %s AS IDENTITY", def, identityType)
	}
	if c.NotNull {
		def = fmt.Sprintf("%s NOT NULL", def)
	}
	if c.DefaultDef.Valid {
		if c.IsGenerated {
			def = fmt.Sprintf("%s GENERATED ALWAYS AS (%s) STORED", def, c.DefaultDef.String)
		} else {
			def = fmt.Sprintf("%s DEFAULT %s", def, c.DefaultDef.String)
		}
	}
	return def
}
