// Package when provides typed inclusion predicates for weave Builder and Group
// methods. An inclusion predicate that reports false omits its associated query
// node; it does not create a false node. Helpers in this package do not validate
// query values or produce errors.
//
// Builder and Group methods evaluate predicates from left to right, stop after
// the first false result, and call each evaluated predicate once. A nil
// predicate is invalid and is reported by weave when the containing Predicate
// is built. Use [All], [Any], and [Not] to compose inclusion rules without
// changing query Boolean logic.
package when
