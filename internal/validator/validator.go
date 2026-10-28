package validator

import (
	"fmt"
	"reflect"
	"strings"
)

type Record map[string]any

type Normalization string

const (
	NormalizeNone          Normalization = "none"
	NormalizeTrimLowercase Normalization = "trim_lowercase"
)

type FieldRule struct {
	SourceField      string        `json:"source_field"`
	DestinationField string        `json:"destination_field"`
	Required         bool          `json:"required"`
	Normalization    Normalization `json:"normalization"`
}

type Issue struct {
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Expected any    `json:"expected,omitempty"`
	Actual   any    `json:"actual,omitempty"`
}

type Result struct {
	Valid   bool    `json:"valid"`
	Checked int     `json:"checked"`
	Issues  []Issue `json:"issues"`
}

func Compare(source, destination Record, rules []FieldRule) Result {
	result := Result{
		Valid:  true,
		Issues: []Issue{},
	}

	for _, rule := range rules {
		if strings.TrimSpace(rule.SourceField) == "" ||
			strings.TrimSpace(rule.DestinationField) == "" {
			result.addIssue("", "invalid_rule", "source and destination field names are required", nil, nil)
			continue
		}

		sourceValue, sourceExists := source[rule.SourceField]
		destinationValue, destinationExists := destination[rule.DestinationField]

		if !sourceExists {
			if rule.Required {
				result.addIssue(rule.SourceField, "missing_source", "required field is missing from source record", "present", nil)
			}
			continue
		}

		if !destinationExists {
			if rule.Required {
				result.addIssue(rule.DestinationField, "missing_destination", "required field is missing from destination record", sourceValue, nil)
			}
			continue
		}

		result.Checked++

		normalizedSource := normalize(sourceValue, rule.Normalization)
		normalizedDestination := normalize(destinationValue, rule.Normalization)

		if reflect.TypeOf(normalizedSource) != reflect.TypeOf(normalizedDestination) {
			result.addIssue(
				rule.DestinationField,
				"type_mismatch",
				fmt.Sprintf("source is %T, destination is %T", normalizedSource, normalizedDestination),
				normalizedSource,
				normalizedDestination,
			)
			continue
		}

		if !reflect.DeepEqual(normalizedSource, normalizedDestination) {
			result.addIssue(
				rule.DestinationField,
				"value_mismatch",
				"source and destination values differ",
				normalizedSource,
				normalizedDestination,
			)
		}
	}

	return result
}

func normalize(value any, mode Normalization) any {
	if text, ok := value.(string); ok && mode == NormalizeTrimLowercase {
		return strings.ToLower(strings.TrimSpace(text))
	}
	return value
}

func (r *Result) addIssue(field, code, message string, expected, actual any) {
	r.Valid = false
	r.Issues = append(r.Issues, Issue{
		Field:    field,
		Code:     code,
		Message:  message,
		Expected: expected,
		Actual:   actual,
	})
}
