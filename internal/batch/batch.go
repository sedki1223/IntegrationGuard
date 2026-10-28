package batch

import (
	"encoding/json"
	"fmt"
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
		result.addIssue(
			nil,
			"invalid_match_key",
			"source and destination match-key fields are required",
			nil,
		)
		result.Valid = false
		return result
	}

	destByKey := make(map[string]validator.Record, len(destinationRecords))
	originalDestKeys := make(map[string]any, len(destinationRecords))
	ambiguousDestKeys := make(map[string]bool)

	for _, record := range destinationRecords {
		value, ok := record[key.DestinationField]
		if !ok || value == nil {
			result.addIssue(
				nil,
				"missing_destination_key",
				fmt.Sprintf("destination record is missing match key %q", key.DestinationField),
				nil,
			)
			continue
		}

		index, valid := keyString(value)
		if !valid {
			result.addIssue(
				nil,
				"invalid_destination_key_type",
				fmt.Sprintf(
					"destination match key %q must be a string, number, or boolean; got %T",
					key.DestinationField,
					value,
				),
				nil,
			)
			continue
		}

		if ambiguousDestKeys[index] {
			continue
		}

		if _, exists := destByKey[index]; exists {
			result.addIssue(
				value,
				"duplicate_destination_key",
				fmt.Sprintf("multiple destination records use key %v; matching is ambiguous", value),
				nil,
			)

			delete(destByKey, index)
			delete(originalDestKeys, index)
			ambiguousDestKeys[index] = true
			continue
		}

		destByKey[index] = record
		originalDestKeys[index] = value
	}

	seenSource := make(map[string]bool, len(sourceRecords))

	for _, source := range sourceRecords {
		value, ok := source[key.SourceField]
		if !ok || value == nil {
			result.FailedRecords++
			result.addIssue(
				nil,
				"missing_source_key",
				fmt.Sprintf("source record is missing match key %q", key.SourceField),
				nil,
			)
			continue
		}

		index, valid := keyString(value)
		if !valid {
			result.FailedRecords++
			result.addIssue(
				nil,
				"invalid_source_key_type",
				fmt.Sprintf(
					"source match key %q must be a string, number, or boolean; got %T",
					key.SourceField,
					value,
				),
				nil,
			)
			continue
		}

		if seenSource[index] {
			result.FailedRecords++
			result.addIssue(
				value,
				"duplicate_source_key",
				fmt.Sprintf("multiple source records use key %v", value),
				nil,
			)
			continue
		}
		seenSource[index] = true

		if ambiguousDestKeys[index] {
			result.FailedRecords++
			result.addIssue(
				value,
				"ambiguous_destination_match",
				"multiple destination records use this key; field validation was skipped",
				nil,
			)
			continue
		}

		destination, exists := destByKey[index]
		if !exists {
			result.FailedRecords++
			result.addIssue(
				value,
				"missing_destination_record",
				"source record has no matching destination record",
				nil,
			)
			continue
		}

		result.MatchedRecords++

		comparison := validator.Compare(source, destination, rules)
		if comparison.Valid {
			result.PassedRecords++
		} else {
			result.FailedRecords++
			result.addIssue(
				value,
				"field_validation_failed",
				fmt.Sprintf("%d field validation issue(s)", len(comparison.Issues)),
				comparison.Issues,
			)
		}
	}

	for index, value := range originalDestKeys {
		if !seenSource[index] {
			result.addIssue(
				value,
				"unexpected_destination_record",
				"destination record has no matching source record",
				nil,
			)
		}
	}

	sort.SliceStable(result.Issues, func(i, j int) bool {
		if result.Issues[i].Code != result.Issues[j].Code {
			return result.Issues[i].Code < result.Issues[j].Code
		}
		return keyStringForSort(result.Issues[i].RecordKey) <
			keyStringForSort(result.Issues[j].RecordKey)
	})

	result.Valid = len(result.Issues) == 0
	return result
}

// keyString accepts only JSON scalar identifiers and preserves their types.
// For example, the string "427" and the number 427 are distinct keys.
func keyString(value any) (string, bool) {
	switch value.(type) {
	case string, bool,
		json.Number,
		float32, float64,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%T:%v", value, value), true
	default:
		return "", false
	}
}

func keyStringForSort(value any) string {
	if value == nil {
		return ""
	}

	key, valid := keyString(value)
	if !valid {
		return fmt.Sprintf("%T:%v", value, value)
	}
	return key
}

func (r *Result) addIssue(
	key any,
	code string,
	message string,
	details []validator.Issue,
) {
	r.Issues = append(r.Issues, RecordIssue{
		RecordKey: key,
		Code:      code,
		Message:   message,
		Details:   details,
	})
}
