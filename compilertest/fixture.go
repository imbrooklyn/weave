package compilertest

// LiteralSpecialText contains wildcard, escape, regular-expression, backslash,
// Unicode, and newline characters. Text operators must treat the complete
// value as literal text.
const LiteralSpecialText = "%_! .*+?[](){}^$|\\ \u4e16\u754c\nend"

// Record is the shared scalar fixture row used by Compiler semantic tests.
// Nullable fields distinguish a present value, explicit null, and missing by
// combining a pointer with a presence flag:
//
//   - Present is true and the pointer is non-nil for a value.
//   - Present is true and the pointer is nil for explicit null.
//   - Present is false for missing; the pointer is then ignored.
//
// Adapters whose storage model has no missing state may materialize missing
// nullable fields as null and set Harness.DistinguishesMissing to false.
type Record struct {
	// ID is the stable record identifier compared by the semantic suite.
	ID string
	// Number is an always-present scalar number.
	Number int64
	// Text is an always-present scalar string.
	Text string
	// NullableNumber holds a number when the field has a value.
	NullableNumber *int64
	// NullableNumberPresent distinguishes value or null from missing.
	NullableNumberPresent bool
	// NullableText holds text when the field has a value.
	NullableText *string
	// NullableTextPresent distinguishes value or null from missing.
	NullableTextPresent bool
}

// Records returns a fresh copy of the shared fixture. Every call allocates
// independent nullable-value pointers so a test cannot mutate another test's
// fixture through those pointers.
func Records() []Record {
	return []Record{
		valueRecord("r01", 1, "plain-start", 1, "plain-start"),
		valueRecord("r02", 2, "literal "+LiteralSpecialText, 2, "literal "+LiteralSpecialText),
		nullRecord("r03", 3, "prefix-middle-suffix"),
		missingRecord("r04", 4, "prefix %_ suffix"),
		valueRecord("r05", 5, "\u4e16\u754c-end", 5, "\u4e16\u754c-end"),
		valueRecord("r06", 6, ".*", 2, ".*"),
	}
}

func valueRecord(
	id string,
	number int64,
	text string,
	nullableNumber int64,
	nullableText string,
) Record {
	return Record{
		ID:                    id,
		Number:                number,
		Text:                  text,
		NullableNumber:        pointerTo(nullableNumber),
		NullableNumberPresent: true,
		NullableText:          pointerTo(nullableText),
		NullableTextPresent:   true,
	}
}

func nullRecord(id string, number int64, text string) Record {
	return Record{
		ID:                    id,
		Number:                number,
		Text:                  text,
		NullableNumberPresent: true,
		NullableTextPresent:   true,
	}
}

func missingRecord(id string, number int64, text string) Record {
	return Record{
		ID:     id,
		Number: number,
		Text:   text,
	}
}

func pointerTo[T any](value T) *T {
	return &value
}
