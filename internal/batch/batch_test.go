package batch

import (
	"testing"

	"integration-guard/internal/validator"
)

func TestUnsupportedMatchKeyTypes(t *testing.T) {
	rules := []validator.FieldRule{
		{
			SourceField:      "department",
			DestinationField: "departmentCode",
			Required:         true,
		},
	}

	key := MatchKey{
		SourceField:      "employee_id",
		DestinationField: "staffCode",
	}

	tests := []struct {
		name          string
		source        []validator.Record
		destination   []validator.Record
		wantIssueCode string
		wantFailed    int
	}{
		{
			name: "source object key is rejected",
			source: []validator.Record{
				{
					"employee_id": map[string]any{"id": "1"},
					"department":  "FIN",
				},
			},
			wantIssueCode: "invalid_source_key_type",
			wantFailed:    1,
		},
		{
			name: "source array key is rejected",
			source: []validator.Record{
				{
					"employee_id": []any{"1"},
					"department":  "FIN",
				},
			},
			wantIssueCode: "invalid_source_key_type",
			wantFailed:    1,
		},
		{
			name: "destination object key is rejected",
			source: []validator.Record{
				{"employee_id": "1", "department": "FIN"},
			},
			destination: []validator.Record{
				{
					"staffCode":      map[string]any{"id": "1"},
					"departmentCode": "fin",
				},
			},
			wantIssueCode: "invalid_destination_key_type",
			wantFailed:    1,
		},
		{
			name: "boolean keys are supported",
			source: []validator.Record{
				{"employee_id": true, "department": "FIN"},
			},
			destination: []validator.Record{
				{"staffCode": true, "departmentCode": "FIN"},
			},
			wantIssueCode: "",
			wantFailed:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.source, tt.destination, key, rules)

			if tt.wantIssueCode == "" {
				if !got.Valid {
					t.Fatalf("expected valid result, got issues: %+v", got.Issues)
				}
			} else if !hasIssueCode(got.Issues, tt.wantIssueCode) {
				t.Errorf(
					"expected issue code %q, got %+v",
					tt.wantIssueCode,
					got.Issues,
				)
			}

			if got.FailedRecords != tt.wantFailed {
				t.Errorf(
					"FailedRecords = %d, want %d",
					got.FailedRecords,
					tt.wantFailed,
				)
			}
		})
	}
}

func hasIssueCode(issues []RecordIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
