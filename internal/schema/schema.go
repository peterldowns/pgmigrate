package schema

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

const DefaultSchema = "public"

type Config struct {
	// The names of the schemas whose contents should be dumped.
	Schemas []string `yaml:"names"`
	// The name of the file to which the dump should be written.
	Out string `yaml:"out"`
	// Any explicit dependencies between database objects.
	Dependencies map[string][]string `yaml:"dependencies"`
	// Rules for dumping table data in the form of INSERT statements.
	Data []Data `yaml:"data"`
}

type Schema struct {
	// Database objects that can be dumped.
	Extensions    []*Extension
	Domains       []*Domain
	CompoundTypes []*CompoundType
	Enums         []*Enum
	Functions     []*Function
	Tables        []*Table
	Views         []*View
	Sequences     []*Sequence
	Indexes       []*Index
	Constraints   []*Constraint
	Triggers      []*Trigger
	Data          []*Data
	// Metadata that isn't explicitly dumped.
	Config       Config
	Dependencies []*Dependency
}

func Parse(config Config, db *sql.DB) (*Schema, error) {
	if len(config.Schemas) == 0 {
		config.Schemas = []string{DefaultSchema}
	}
	schema := Schema{Config: config}
	// Load and parse each of the different types of object from the database for each schema.
	if err := schema.Load(db); err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	// Assign dependencies between objects.
	byName := schema.ObjectsByName()
	for _, dep := range schema.Dependencies {
		objName := pgtools.Identifier(dep.Object.Schema, dep.Object.Name)
		if obj, ok := byName[objName]; ok {
			obj.AddDependency(pgtools.Identifier(dep.DependsOn.Schema, dep.DependsOn.Name))
		}
	}
	for name, deps := range config.Dependencies {
		obj, ok := byName[name]
		if !ok {
			continue
		}
		for _, dep := range deps {
			obj.AddDependency(dep)
		}
	}

	// Add indexes to their owning table and remove them from schema.Index.
	tablesByName := asMap(schema.Tables)
	indexesByName := asMap(schema.Indexes)
	indexes := []*Index{}
	for _, index := range schema.Indexes {
		tableName := pgtools.Identifier(index.Schema, index.TableName)
		if table, ok := tablesByName[tableName]; ok {
			table.Indexes = append(table.Indexes, index)
		} else {
			indexes = append(indexes, index)
		}
	}
	schema.Indexes = indexes

	// Add constraints to their owning table and remove them from
	// schema.Constraints.
	constraints := []*Constraint{}
	for _, con := range schema.Constraints {
		tableName := pgtools.Identifier(con.Schema, con.TableName)
		if table, ok := tablesByName[tableName]; ok {
			table.Constraints = append(table.Constraints, con)
			continue
		}
		indexName := pgtools.Identifier(con.Schema, con.Index)
		if _, ok := indexesByName[indexName]; ok {
			continue
		}
		constraints = append(constraints, con)
	}
	schema.Constraints = constraints

	// Add sequences to their owning table and remove them from
	// schema.Sequences.
	sequences := []*Sequence{}
	for _, seq := range schema.Sequences {
		if seq.TableName.Valid {
			tableName := pgtools.Identifier(seq.Schema, seq.TableName.String)
			if table, ok := tablesByName[tableName]; ok {
				table.Sequences = append(table.Sequences, seq)
				if seq.ColumnName.Valid {
					colName := seq.ColumnName.String
					for _, col := range table.Columns {
						if col.Name == colName {
							col.Sequence = seq
							break
						}
					}
				}
				continue
			}
		}
		sequences = append(sequences, seq)
	}
	schema.Sequences = sequences

	// Add triggers to their owning table and remove them from schema.Triggers.
	remTriggers := []*Trigger{}
	for _, trig := range schema.Triggers {
		tableName := pgtools.Identifier(trig.Schema, trig.TableName)
		if table, ok := tablesByName[tableName]; ok {
			table.Triggers = append(table.Triggers, trig)
			continue
		}
		remTriggers = append(remTriggers, trig)
	}
	schema.Triggers = remTriggers

	// Inserting data can involve inserting foreign keys, which must respect
	// foreign key constraints at the time of insertion. So, make sure Data
	// inserts happen in the same order as the tables they're referencing — a
	// Data object's SortKey() / Identifier is the same as its underlying table
	// so we can just look up the table's dependencies.
	for _, data := range schema.Data {
		did := pgtools.Identifier(data.Schema, data.Name)
		if obj, ok := byName[did]; ok {
			for _, dep := range obj.DependsOn() {
				data.AddDependency(dep)
			}
		}
	}

	schema.Sort()
	return &schema, nil
}

// Sort orders each type of database objects into creation order. Does not
// perform a global ordering on the different types.
func (s *Schema) Sort() {
	s.Extensions = Sort(s.Extensions)
	s.Domains = Sort(s.Domains)
	s.CompoundTypes = Sort(s.CompoundTypes)
	s.Enums = Sort(s.Enums)
	s.Functions = Sort(s.Functions)
	s.Tables = Sort(s.Tables)
	s.Views = Sort(s.Views)
	s.Sequences = Sort(s.Sequences)
	s.Indexes = Sort(s.Indexes)
	s.Constraints = Sort(s.Constraints)
	s.Triggers = Sort(s.Triggers)
	s.Data = Sort(s.Data)
}

// Load queries the database and populates the slices of database objects. It
// does not assign any additional dependencies between the objects.
func (s *Schema) Load(db *sql.DB) error {
	var err error
	if s.Extensions, err = LoadExtensions(s.Config, db); err != nil {
		return fmt.Errorf("extensions: %w", err)
	}
	if s.Domains, err = LoadDomains(s.Config, db); err != nil {
		return fmt.Errorf("domains: %w", err)
	}
	if s.CompoundTypes, err = LoadCompoundTypes(s.Config, db); err != nil {
		return fmt.Errorf("types: %w", err)
	}
	if s.Enums, err = LoadEnums(s.Config, db); err != nil {
		return fmt.Errorf("enums: %w", err)
	}
	if s.Functions, err = LoadFunctions(s.Config, db); err != nil {
		return fmt.Errorf("functions: %w", err)
	}
	if s.Tables, err = LoadTables(s.Config, db); err != nil {
		return fmt.Errorf("tables: %w", err)
	}
	if s.Views, err = LoadViews(s.Config, db); err != nil {
		return fmt.Errorf("views: %w", err)
	}
	if s.Sequences, err = LoadSequences(s.Config, db); err != nil {
		return fmt.Errorf("sequences: %w", err)
	}
	if s.Indexes, err = LoadIndexes(s.Config, db); err != nil {
		return fmt.Errorf("indexes: %w", err)
	}
	if s.Constraints, err = LoadConstraints(s.Config, db); err != nil {
		return fmt.Errorf("constraints: %w", err)
	}
	if s.Triggers, err = LoadTriggers(s.Config, db); err != nil {
		return fmt.Errorf("triggers: %w", err)
	}
	// Meta
	if s.Dependencies, err = LoadDependencies(s.Config, db); err != nil {
		return fmt.Errorf("dependencies: %w", err)
	}
	if s.Data, err = LoadData(s.Config, db); err != nil {
		return fmt.Errorf("data: %w", err)
	}
	return nil
}

// ObjectsByName returns a map of all the database objects represented as the
// DBObject interface. This representation allows assigning dependencies between
// them, printing them, and sorting them.
func (s *Schema) ObjectsByName() map[string]DBObject {
	count := 0
	count += len(s.Extensions)
	count += len(s.Domains)
	count += len(s.CompoundTypes)
	count += len(s.Enums)
	count += len(s.Functions)
	count += len(s.Tables)
	count += len(s.Views)
	count += len(s.Sequences)
	count += len(s.Indexes)
	count += len(s.Constraints)
	count += len(s.Triggers)
	objects := make([]DBObject, 0, count)

	for _, obj := range s.Extensions {
		objects = append(objects, obj)
	}
	for _, obj := range s.Domains {
		objects = append(objects, obj)
	}
	for _, obj := range s.CompoundTypes {
		objects = append(objects, obj)
	}
	for _, obj := range s.Enums {
		objects = append(objects, obj)
	}
	for _, obj := range s.Functions {
		objects = append(objects, obj)
	}
	for _, obj := range s.Tables {
		objects = append(objects, obj)
	}
	for _, obj := range s.Views {
		objects = append(objects, obj)
	}
	for _, obj := range s.Sequences {
		objects = append(objects, obj)
	}
	for _, obj := range s.Indexes {
		objects = append(objects, obj)
	}
	for _, obj := range s.Constraints {
		objects = append(objects, obj)
	}
	for _, obj := range s.Triggers {
		objects = append(objects, obj)
	}

	return asMap(objects)
}

// String returns the contents of schema file that can be applied with `psql` to
// create a database with the same schema as the one that is parsed. Objects are
// grouped when possible, and ordered such that when an object is created all of
// its dependencies are guaranteed to exist.
//
// This schema file is
//
//   - usable: can `psql $NEW -f schema.sql` to create a new database with the
//     same schema.
//   - diffable: if there are migrations in different PRs/branches that will
//     conflict with each other, diffing the generated schema.sql files from each
//     branch should result in a merge conflict that cannot be automatically
//     resolved.
//   - roundtrippable: dumping `pgmigrate dump --database $NEW > schema.sql`
//     will result in 0 changes.
//   - customizable: you can include tables to dump values from (for enum
//     tables) and you can explicitly add dependencies between objects that will
//     be respected during the dump, to work around faulty dependency detection.
func (s *Schema) String() string {
	out := strings.Builder{}

	// These objects are always emitted first, and are not re-ordered to allow
	// dependencies. This means that, for instance, a Domain cannot depend on a
	// custom Function.
	//
	// - Extensions
	// - Schemas
	// - Domains
	// - Enums
	// - CompoundTypes
	// - Functions
	//
	// The upside is that all the other types of objects don't need to
	// explicitly say they depend on these.
	for _, obj := range s.Extensions {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}
	for _, schemaName := range s.Config.Schemas {
		out.WriteString(schemaDefinition(schemaName))
		out.WriteString("\n\n")
	}
	for _, obj := range s.Domains {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}
	for _, obj := range s.Enums {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}
	for _, obj := range s.CompoundTypes {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}
	for _, obj := range s.Functions {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}

	// These objects are allowed to depend on each other, and are re-ordered
	// to allow those dependencies.
	//
	// - Sequences
	// - Tables
	// - Views
	// - Indexes
	// - Constraints
	// - Triggers
	//
	var sortable []DBObject
	for _, obj := range s.Sequences {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range s.Tables {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range s.Views {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range s.Indexes {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range s.Constraints {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range s.Triggers {
		obj := obj
		sortable = append(sortable, obj)
	}
	sortable = Sort(sortable)
	for _, obj := range sortable {
		out.WriteString(obj.String())
		out.WriteString("\n\n")
	}

	// Add any data-inserting statements after all other database objects have
	// been created.
	for _, obj := range s.Data {
		statement := obj.String()
		if statement != "" {
			out.WriteString(obj.String())
			out.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(out.String())
}

func schemaDefinition(schemaName string) string {
	return fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", pgtools.Identifier(schemaName))
}
