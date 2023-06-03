package pgmigrate

// A VerificationError represents a warning of either of two types:
//
//   - a migration is marked as applied to the database but is not present in
//     the directory of migrations: this can happen if a migration is applied, but
//     the code containing that migration is later rolled back.
//   - a migration whose hash (when applied) doesn't match its current hash
//     (when calculated from its SQL contents): this can happen if someone edits a
//     migration after it was previously applied.
//
// These verification errors are worth looking into, but should not be treated
// the same as a failure to apply migrations. Typically these are warned or
// alerted on by the app using this migration library, and results in a human
// intervening in some way.
type VerificationError struct {
	Message string
	Fields  map[string]any
}
