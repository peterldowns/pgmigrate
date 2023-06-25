package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

// Schema dumping is useful for the following flow:
//
// 1. create new migration file
// 2. generate stuff
// 3. dump schema to schema.sql to cause merge conflicts for conflicting migrations
// 4. use config to override/customize generated schema.sql if necessary
//
// user-controlled: migrations/
// automated: schema.sql
//
// But once schema parsing/dumping is implemented, could go all the way and implement
// the rest of migra/skeema. This could change the flow to:
//
// 1. update schema.sql file
// 2. generate migrations from the schema.sql compared to state of migrations/ dir
// 3. modify generated migration.sql file if necessary
//
// accomplishes the same goals, but! interface to editing the database is "the
// schema file" rather than "the migration file"? Allows for more natural
// definitions of things. This should probably be the end-goal.
//
// The operational flow, regardless, is to run migrations. Over time
// these migrations can be marked as "squashed" to prevent verification errors
// and ignore old contents. How to do that?
//
// 		migration_row.squashed_by => "schema_as_of_100003.sql"
//
// which is just a separate migration, which updates the existing migrations (if
// they exist?) to have "squashed_by" set to itself. After introduction of a squash
//
// - copy schema.sql -> squash.sql
// - append "update * from pgmigrate_migrations where id in (...) set squashed_by=squash.sql squashed_hash=....
//		- migration ids from migrate/*.sql
//		- this brings verification errors into the right state
// - delete migrate/*.sql
//
// planning/applying
// - replace earliest known instance of squashed_by with the squash
// - replace all subsequent existences with no-op
//
// instead of a "squash" concept, have a "base" concept?

// The goal of `pgmigrate dump --database $ORIGINAL > schema.sql` is for the resulting sql file to be:
//   - usable: can `psql $NEW -f schema.sql` to create a new database with the same schema.
//   - diffable: if there are migrations in different PRs/branches that will conflict with each other,
//       diffing the generated schema.sql files from each branch should result in a merge conflict that
//       cannot be automatically resolved.
//   - roundtrippable: dumping `pgmigrate dump --database $NEW > schema.sql` will result in 0 changes.
//   - customizable: you can include tables to dump values from (for enum tables) and you can explicitly
//       add dependencies between objects that will be respected during the dump, to work around faulty
//       dependency detection.

type ConfigDependency struct { // TODO: rename this?
	Name      string
	DependsOn []string
}

type ConfigData struct { // TODO: rename this?
	Name    string
	Columns []string
	OrderBy string
}
type Config struct {
	Schema       string
	Dependencies []ConfigDependency
	Data         []Data
}

type Result struct { // TODO: rename to Schema?
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
	// Metadata that isn't explicitly dumped.
	Config       Config
	Dependencies []*Dependency
	Data         []*Data // TODO: better name
}

func (r *Result) Sort() {
	r.Extensions = Sort[string](r.Extensions)
	r.Domains = Sort[string](r.Domains)
	r.CompoundTypes = Sort[string](r.CompoundTypes)
	r.Enums = Sort[string](r.Enums)
	r.Functions = Sort[string](r.Functions)
	r.Tables = Sort[string](r.Tables)
	r.Views = Sort[string](r.Views)
	r.Sequences = Sort[string](r.Sequences)
	r.Indexes = Sort[string](r.Indexes)
	r.Constraints = Sort[string](r.Constraints)
	r.Triggers = Sort[string](r.Triggers)
}

func (r *Result) Load(db *sql.DB) error {
	var err error
	// Objects
	if r.Extensions, err = LoadExtensions(r.Config, db); err != nil {
		return fmt.Errorf("extensions: %w", err)
	}
	if r.Domains, err = LoadDomains(r.Config, db); err != nil {
		return fmt.Errorf("domains: %w", err)
	}
	if r.CompoundTypes, err = LoadCompoundTypes(r.Config, db); err != nil {
		return fmt.Errorf("types: %w", err)
	}
	if r.Enums, err = LoadEnums(r.Config, db); err != nil {
		return fmt.Errorf("enums: %w", err)
	}
	if r.Functions, err = LoadFunctions(r.Config, db); err != nil {
		return fmt.Errorf("functions: %w", err)
	}
	if r.Tables, err = LoadTables(r.Config, db); err != nil {
		return fmt.Errorf("tables: %w", err)
	}
	if r.Views, err = LoadViews(r.Config, db); err != nil {
		return fmt.Errorf("views: %w", err)
	}
	if r.Sequences, err = LoadSequences(r.Config, db); err != nil {
		return fmt.Errorf("sequences: %w", err)
	}
	if r.Indexes, err = LoadIndexes(r.Config, db); err != nil {
		return fmt.Errorf("indexes: %w", err)
	}
	if r.Constraints, err = LoadConstraints(r.Config, db); err != nil {
		return fmt.Errorf("constraints: %w", err)
	}
	if r.Triggers, err = LoadTriggers(r.Config, db); err != nil {
		return fmt.Errorf("triggers: %w", err)
	}
	// Meta
	if r.Dependencies, err = LoadDependencies(r.Config, db); err != nil {
		return fmt.Errorf("dependencies: %w", err)
	}
	if r.Data, err = LoadData(r.Config, db); err != nil {
		return fmt.Errorf("data: %w", err)
	}
	return nil
}

func (r *Result) ObjectsByName() map[string]DBObject {
	count := 0
	count += len(r.Extensions)
	count += len(r.Domains)
	count += len(r.CompoundTypes)
	count += len(r.Enums)
	count += len(r.Functions)
	count += len(r.Tables)
	count += len(r.Views)
	count += len(r.Sequences)
	count += len(r.Indexes)
	count += len(r.Constraints)
	count += len(r.Triggers)
	objects := make([]DBObject, 0, count)

	// TODO: stop using asMap here, not necessary
	for _, obj := range r.Extensions {
		objects = append(objects, obj)
	}
	for _, obj := range r.Domains {
		objects = append(objects, obj)
	}
	for _, obj := range r.CompoundTypes {
		objects = append(objects, obj)
	}
	for _, obj := range r.Enums {
		objects = append(objects, obj)
	}
	for _, obj := range r.Functions {
		objects = append(objects, obj)
	}
	for _, obj := range r.Tables {
		objects = append(objects, obj)
	}
	for _, obj := range r.Views {
		objects = append(objects, obj)
	}
	for _, obj := range r.Sequences {
		objects = append(objects, obj)
	}
	for _, obj := range r.Indexes {
		objects = append(objects, obj)
	}
	for _, obj := range r.Constraints {
		objects = append(objects, obj)
	}
	for _, obj := range r.Triggers {
		objects = append(objects, obj)
	}

	return asMap[string](objects)
}

func Parse(config Config, db *sql.DB) (*Result, error) { // TODO: rename to New?
	result := Result{Config: config}
	if err := result.Load(db); err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	byName := result.ObjectsByName()

	tablesByName := asMap[string](result.Tables)
	indexesByName := asMap[string](result.Indexes)
	remIndexes := []*Index{}
	for _, index := range result.Indexes {
		if table, ok := tablesByName[RefTable(index.TableName)]; ok {
			table.Indexes = append(table.Indexes, index)
		} else {
			remIndexes = append(remIndexes, index)
		}
	}
	result.Indexes = remIndexes

	remConstraints := []*Constraint{}
	for _, con := range result.Constraints {
		if con.ForeignTableName == "" {
			if table, ok := tablesByName[RefTable(con.TableName)]; ok {
				table.Constraints = append(table.Constraints, con)
				continue
			}
		}
		if _, ok := indexesByName[RefIndex(con.Index)]; ok {
			continue
		}
		remConstraints = append(remConstraints, con)
	}
	result.Constraints = remConstraints

	remSequences := []*Sequence{}
	for _, seq := range result.Sequences {
		if seq.TableName.Valid {
			if table, ok := tablesByName[RefTable(seq.TableName.String)]; ok {
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
		remSequences = append(remSequences, seq)
	}
	result.Sequences = remSequences

	remTriggers := []*Trigger{}
	for _, trig := range result.Triggers {
		if table, ok := tablesByName[trig.TableName]; ok {
			table.Triggers = append(table.Triggers, trig)
			continue
		}
		remTriggers = append(remTriggers, trig)
	}
	result.Triggers = remTriggers

	for _, dep := range result.Dependencies {
		if obj, ok := byName[dep.Object.Name]; ok {
			obj.AddDependency(dep.DependsOn.Name)
		}
	}
	for _, dep := range config.Dependencies {
		leftObj, ok := byName[dep.Name]
		if !ok {
			continue
		}
		for _, right := range dep.DependsOn {
			leftObj.AddDependency(right)
		}
	}

	for _, tc := range []struct {
		x string
		y string
	}{} {
		if x, ok := byName[tc.x]; ok {
			x.AddDependency(tc.y)
		}
	}

	result.Sort()
	return &result, nil
}

func Dump(r *Result) string {
	// TODO: use a buf / string-builder approach here for, you know,
	// "efficiency" because that's sooooooooooooooooooooo important.
	var out string
	for _, obj := range r.Extensions {
		out += obj.String() + "\n\n"
	}
	for _, obj := range r.Domains {
		out += obj.String() + "\n\n"
	}
	for _, obj := range r.Enums {
		out += obj.String() + "\n\n"
	}
	for _, obj := range r.CompoundTypes {
		out += obj.String() + "\n\n"
	}
	for _, obj := range r.Functions {
		out += obj.String() + "\n\n"
	}
	var sortable []DBObject
	for _, obj := range r.Sequences {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range r.Tables {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range r.Views {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range r.Indexes {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range r.Constraints {
		obj := obj
		sortable = append(sortable, obj)
	}
	for _, obj := range r.Triggers {
		obj := obj
		sortable = append(sortable, obj)
	}
	sortable = Sort[string](sortable)
	for _, obj := range sortable {
		out += obj.String() + "\n\n"
	}
	for _, data := range r.Data {
		out += data.String() + "\n\n"
	}
	return strings.TrimSpace(out)
}
