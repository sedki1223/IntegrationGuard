package batch

import (
	"fmt"
	"reflect"
	"sort"

	"integration-guard/internal/validator"
)

type MatchKey struct {
	SourceField      string `json:"source_field"`
	DestinationField string `json:"destination_field"`
}

type RecordIssue struct {
	RecordKey any               `json:"record_key,omitempty"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   []validator.Issue `json:"details,omitempty"`
}

type Result struct {
	Valid              bool          `json:"valid"`
	SourceRecords      int           `json:"source_records"`
	DestinationRecords int           `json:"destination_records"`
	MatchedRecords     int           `json:"matched_records"`
	PassedRecords      int           `json:"passed_records"`
	FailedRecords      int           `json:"failed_records"`
	Issues             []RecordIssue `json:"issues"`
}

func Compare(
	sourceRecords []validator.Record,
	destinationRecords []validator.Record,
	key MatchKey,
	rules []validator.FieldRule,
) Result {
	result := Result{
		Valid:              true,
		Issues:             []RecordIssue{},
		SourceRecords:      len(sourceRecords),
		DestinationRecords: len(destinationRecords),
	}

	if key.SourceField == "" || key.DestinationField == "" {
		result.addIssue(nil, "invalid_match_key", "source and destination match-key fields are required", nil)
		result.Valid = false
		return result
	}

	destByKey := make(map[string]validator.Record, len(destinationRecords))
	destOriginalKeys := make(map[string]any, len(destinationRecords))

	for _, record := range destinationRecords {
		value, ok := record[key.DestinationField]
		if !ok || value == nil {
			result.addIssue(nil, "missing_destination_key",
				fmt.Sprintf("destination record is missing match key %q", key.DestinationField), nil)
			continue
		}

		index := keyString(value)
		if _, exists := destByKey[index]; exists {
			result.addIssue(value, "duplicate_destination_key",
				fmt.Sprintf("multiple destination records use key %v", value), nil)
			continue
		}
		destByKey[index] = record
		destOriginalKeys[index] = value
	}

	seenSource := make(map[string]bool, len(sourceRecords))

	for _, source := range sourceRecords {
		value, ok := source[key.SourceField]
		if !ok || value == nil {
			result.FailedRecords++
			result.addIssue(nil, "missing_source_key",
				fmt.Sprintf("source record is missing match key %q", key.SourceField), nil)
			continue
		}

		index := keyString(value)
		if seenSource[index] {
			result.FailedRecords++
			result.addIssue(value, "duplicate_source_key",
				fmt.Sprintf("multiple source records use key %v", value), nil)
			continue
		}
		seenSource[index] = true

		destination, exists := destByKey[index]
		if !exists {
			result.FailedRecords++
			result.addIssue(value, "missing_destination_record",
				"source record has no matching destination record", nil)
			continue
		}

		result.MatchedRecords++
		comparison := validator.Compare(source, destination, rules)
		if comparison.Valid {
			result.PassedRecords++
		} else {
			result.FailedRecords++
			result.addIssue(value, "field_validation_failed",
				fmt.Sprintf("%d field validation issue(s)", len(comparison.Issues)),
				comparison.Issues)
		}
	}

	for index, value := range destOriginalKeys {
		if !seenSource[index] {
			result.addIssue(value, "unexpected_destination_record",
				"destination record has no matching source record", nil)
		}
	}

	sort.SliceStable(result.Issues, func(i, j int) bool {
		return result.Issues[i].Code < result.Issues[j].Code
	})

	result.Valid = len(result.Issues) == 0
	return result
}

func keyString(value any) string {
	return fmt.Sprintf("%T:%v", value, value)
}

func (r *Result) addIssue(key any, code, message string, details []validator.Issue) {
	r.Issues = append(r.Issues, RecordIssue{
		RecordKey: key,
		Code:      code,
		Message:   message,
		Details:   details,
	})
}

// Keep reflect imported to make key comparison behavior explicit if key handling
// is extended later; current keys are indexed by type and formatted value.
var _ = reflect.DeepEqual
