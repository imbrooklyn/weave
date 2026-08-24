package compilertest

import (
	"slices"
	"strings"
	"testing"
)

func TestRecordsAreStableAndIndependent(t *testing.T) {
	first := Records()
	second := Records()
	if len(first) != 6 || len(second) != 6 {
		t.Fatalf("fixture lengths = (%d, %d), want (6, 6)", len(first), len(second))
	}

	wantIDs := []string{"r01", "r02", "r03", "r04", "r05", "r06"}
	gotIDs := make([]string, len(first))
	for index := range first {
		gotIDs[index] = first[index].ID
	}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("record IDs = %v, want %v", gotIDs, wantIDs)
	}

	if first[0].NullableNumber == second[0].NullableNumber ||
		first[0].NullableText == second[0].NullableText {
		t.Fatal("Records() shared nullable-value pointers between calls")
	}
	*first[0].NullableNumber = 99
	*first[0].NullableText = "changed"
	if *second[0].NullableNumber != 1 || *second[0].NullableText != "plain-start" {
		t.Fatal("mutating one fixture call changed another call")
	}
}

func TestRecordsCoverValueNullMissingAndLiteralText(t *testing.T) {
	records := Records()
	if records[0].NullableNumber == nil || !records[0].NullableNumberPresent {
		t.Fatal("r01 does not encode a present value")
	}
	if records[2].NullableNumber != nil || !records[2].NullableNumberPresent {
		t.Fatal("r03 does not encode explicit null")
	}
	if records[3].NullableNumberPresent {
		t.Fatal("r04 does not encode missing")
	}
	if !strings.Contains(records[1].Text, LiteralSpecialText) {
		t.Fatal("r02 does not contain LiteralSpecialText")
	}
}

func TestCanonicalIDsUseSetSemantics(t *testing.T) {
	got := canonicalIDs([]string{"r02", "r01", "r02"})
	want := []string{"r01", "r02"}
	if !slices.Equal(got, want) {
		t.Fatalf("canonicalIDs() = %v, want %v", got, want)
	}
}
