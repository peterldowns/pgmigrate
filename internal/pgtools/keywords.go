package pgtools

// https://www.postgresql.org/docs/current/sql-keywords-appendix.html
//
// These are any keywords which are labeled as:
//
//   - reserved
//   - non-reserved (cannot be function or type)
//   - requires AS
//
// If any of these conditions are true, then the identifier requires quoting in
// at least some context (when used as a table, column, type, or function name)
// and so we choose to always quote them.
var postgresKeywords = map[string]struct{}{
	"all":               {},
	"analyse":           {},
	"analyze":           {},
	"and":               {},
	"any":               {},
	"array":             {},
	"as":                {},
	"asc":               {},
	"asymmetric":        {},
	"authorization":     {},
	"between":           {},
	"bigint":            {},
	"binary":            {},
	"bit":               {},
	"boolean":           {},
	"both":              {},
	"case":              {},
	"cast":              {},
	"char":              {},
	"character":         {},
	"characteristics":   {},
	"check":             {},
	"coalesce":          {},
	"collate":           {},
	"collation":         {},
	"column":            {},
	"concurrently":      {},
	"constraint":        {},
	"create":            {},
	"cross":             {},
	"current_catalog":   {},
	"current_date":      {},
	"current_role":      {},
	"current_schema":    {},
	"current_time":      {},
	"current_timestamp": {},
	"current_user":      {},
	"day":               {},
	"dec":               {},
	"decimal":           {},
	"default":           {},
	"deferrable":        {},
	"desc":              {},
	"distinct":          {},
	"do":                {},
	"else":              {},
	"end":               {},
	"except":            {},
	"exists":            {},
	"fetch":             {},
	"filter":            {},
	"float":             {},
	"for":               {},
	"foreign":           {},
	"freeze":            {},
	"from":              {},
	"full":              {},
	"grant":             {},
	"greatest":          {},
	"group":             {},
	"grouping":          {},
	"having":            {},
	"hour":              {},
	"ilike":             {},
	"in":                {},
	"initially":         {},
	"inner":             {},
	"inout":             {},
	"instead":           {},
	"int":               {},
	"integer":           {},
	"intersect":         {},
	"interval":          {},
	"into":              {},
	"is":                {},
	"isnull":            {},
	"join":              {},
	"json":              {},
	"json_array":        {},
	"json_arrayagg":     {},
	"json_exists":       {},
	"json_object":       {},
	"json_objectagg":    {},
	"json_query":        {},
	"json_scalar":       {},
	"json_serialize":    {},
	"json_table":        {},
	"json_value":        {},
	"lateral":           {},
	"leading":           {},
	"least":             {},
	"left":              {},
	"like":              {},
	"limit":             {},
	"localtime":         {},
	"localtimestamp":    {},
	"merge_action":      {},
	"minute":            {},
	"month":             {},
	"national":          {},
	"natural":           {},
	"nchar":             {},
	"none":              {},
	"normalize":         {},
	"not":               {},
	"notnull":           {},
	"null":              {},
	"nullif":            {},
	"numeric":           {},
	"offset":            {},
	"on":                {},
	"only":              {},
	"or":                {},
	"order":             {},
	"out":               {},
	"outer":             {},
	"over":              {},
	"overlaps":          {},
	"overlay":           {},
	"placing":           {},
	"position":          {},
	"precision":         {},
	"primary":           {},
	"real":              {},
	"references":        {},
	"reserved":          {},
	"returning":         {},
	"right":             {},
	"row":               {},
	"second":            {},
	"session_user":      {},
	"setof":             {},
	"similar":           {},
	"smallint":          {},
	"some":              {},
	"substring":         {},
	"symmetric":         {},
	"system_user":       {},
	"table":             {},
	"tablesample":       {},
	"then":              {},
	"time":              {},
	"timestamp":         {},
	"to":                {},
	"trailing":          {},
	"treat":             {},
	"true":              {},
	"union":             {},
	"unique":            {},
	"user":              {},
	"using":             {},
	"values":            {},
	"varchar":           {},
	"variadic":          {},
	"varying":           {},
	"verbose":           {},
	"when":              {},
	"where":             {},
	"window":            {},
	"with":              {},
	"within":            {},
	"without":           {},
	"xmlattributes":     {},
	"xmlconcat":         {},
	"xmlelement":        {},
	"xmlexists":         {},
	"xmlforest":         {},
	"xmlnamespaces":     {},
	"xmlparse":          {},
	"xmlpi":             {},
	"xmlroot":           {},
	"xmlserialize":      {},
	"xmltable":          {},
	"year":              {},
}
