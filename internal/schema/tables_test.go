package schema_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/peterldowns/pgmigrate/internal/schema"
)

func TestExpressionBasedUniqueIndexes(t *testing.T) {
	t.Parallel()
	dbtest(t, query(`--sql
CREATE TABLE public.tab_folders (
    id_folder UUID PRIMARY KEY NOT NULL,
    id_parent_folder UUID REFERENCES public.tab_folders(id_folder),
    name jsonb NOT NULL
);

CREATE UNIQUE INDEX idx_tab_folders_id_parent_folder_name_cs
    ON public.tab_folders (id_parent_folder, (name->>'cs'))
    WHERE name ? 'cs';

CREATE UNIQUE INDEX idx_tab_folders_id_parent_folder_name_en
    ON public.tab_folders (id_parent_folder, (name->>'en'))
    WHERE name ? 'en';
	`), func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}

		// Check that id_parent_folder is NOT marked as unique
		// since it only participates in expression-based unique indexes
		tableSQL := result.Tables[0].String()
		// The column should NOT have UNIQUE keyword since there's no actual
		// UNIQUE constraint, only unique expression-based indexes
		if strings.Contains(tableSQL, "id_parent_folder uuid UNIQUE") {
			t.Errorf("Column id_parent_folder incorrectly marked as UNIQUE.\nGenerated SQL:\n%s", tableSQL)
		}

		return nil
	})
}
