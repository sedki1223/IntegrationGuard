package batch

import (
	"testing"

	"integration-guard/internal/validator"
)

func TestBatchComparison(t *testing.T) {
	rules := []validator.FieldRule{
		{SourceField: "employee_id", DestinationField: "staffCode", Required: true},
		{SourceField: "department", DestinationField: "departmentCode", Required: true, Normalization: validator.NormalizeTrimLowercase},
	}
	key := MatchKey{SourceField: "employee_id", DestinationField: "staffCode"}

	tests := []struct {
		name          string
		source        []validator.Record
		destination   []validator.Record
		wantValid     bool
		wantMatched   int
		wantPassed    int
		wantFailed    int
		wantIssueCode string
	}{
		{
			name: "all records match",
			source: []validator.Record{
				{"employee_id": "1", "department": "FIN"},
				{"employee_id": "2", "department": "HR"},
			},
			destination: []validator.Record{
				{"staffCode": "1", "departmentCode": "fin"},
				{"staffCode": "2", "departmentCode": "hr"},
			},
			wantValid: true, wantMatched: 2, wantPassed: 2,
		},
		{
			name:        "field mismatch",
			source:      []validator.Record{{"employee_id": "1", "department": "FIN"}},
			destination: []validator.Record{{"staffCode": "1", "departmentCode": "OPS"}},
			wantMatched: 1, wantFailed: 1, wantIssueCode: "field_validation_failed",
		},
		{
			name:        "missing destination record",
			source:      []validator.Record{{"employee_id": "1", "department": "FIN"}},
			destination: []validator.Record{},
			wantFailed:  1, wantIssueCode: "missing_destination_record",
		},
		{
			name:          "unexpected destination record",
			source:        []validator.Record{},
			destination:   []validator.Record{{"staffCode": "9", "departmentCode": "OPS"}},
			wantIssueCode: "unexpected_destination_record",
		},
		{
			name: "duplicate source key",
			source: []validator.Record{
				{"employee_id": "1", "department": "FIN"},
				{"employee_id": "1", "department": "HR"},
			},
			destination: []validator.Record{{"staffCode": "1", "departmentCode": "fin"}},
			wantMatched: 1, wantPassed: 1, wantFailed: 1, wantIssueCode: "duplicate_source_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.source, tt.destination, key, rules)

			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; issues: %+v", got.Valid, tt.wantValid, got.Issues)
			}
			if got.MatchedRecords != tt.wantMatched {
				t.Errorf("MatchedRecords = %d, want %d", got.MatchedRecords, tt.wantMatched)
			}
			if got.PassedRecords != tt.wantPassed {
				t.Errorf("PassedRecords = %d, want %d", got.PassedRecords, tt.wantPassed)
			}
			if got.FailedRecords != tt.wantFailed {
				t.Errorf("FailedRecords = %d, want %d", got.FailedRecords, tt.wantFailed)
			}
			if tt.wantIssueCode != "" {
				found := false
				for _, issue := range got.Issues {
					if issue.Code == tt.wantIssueCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue code %q, got %+v", tt.wantIssueCode, got.Issues)
				}
			}
		})
	}
}
