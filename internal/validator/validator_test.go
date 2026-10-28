package validator

import "testing"

func TestValidationEngine(t *testing.T) {
	tests := []struct {
		name           string
		source         Record
		destination    Record
		rules          []FieldRule
		wantValid      bool
		wantChecked    int
		wantIssueCodes []string
	}{
		{
			name:        "matching records",
			source:      Record{"id": "00427", "department": "FIN"},
			destination: Record{"id": "00427", "department": "FIN"},
			rules: []FieldRule{
				{SourceField: "id", DestinationField: "id", Required: true},
				{SourceField: "department", DestinationField: "department", Required: true},
			},
			wantValid:   true,
			wantChecked: 2,
		},
		{
			name:        "changed identifier",
			source:      Record{"id": "00427"},
			destination: Record{"id": "427"},
			rules: []FieldRule{
				{SourceField: "id", DestinationField: "id", Required: true},
			},
			wantChecked:    1,
			wantIssueCodes: []string{"value_mismatch"},
		},
		{
			name:        "missing required destination field",
			source:      Record{"department": "FIN"},
			destination: Record{},
			rules: []FieldRule{
				{SourceField: "department", DestinationField: "department", Required: true},
			},
			wantIssueCodes: []string{"missing_destination"},
		},
		{
			name:        "type mismatch",
			source:      Record{"amount": 1250},
			destination: Record{"amount": "1250"},
			rules: []FieldRule{
				{SourceField: "amount", DestinationField: "amount", Required: true},
			},
			wantChecked:    1,
			wantIssueCodes: []string{"type_mismatch"},
		},
		{
			name:        "different field names",
			source:      Record{"employee_id": "00427"},
			destination: Record{"staffCode": "00427"},
			rules: []FieldRule{
				{SourceField: "employee_id", DestinationField: "staffCode", Required: true},
			},
			wantValid:   true,
			wantChecked: 1,
		},
		{
			name:        "explicit string normalization",
			source:      Record{"email": " USER@EXAMPLE.COM "},
			destination: Record{"emailAddress": "user@example.com"},
			rules: []FieldRule{
				{
					SourceField:      "email",
					DestinationField: "emailAddress",
					Required:         true,
					Normalization:    NormalizeTrimLowercase,
				},
			},
			wantValid:   true,
			wantChecked: 1,
		},
		{
			name:        "missing optional fields",
			source:      Record{},
			destination: Record{},
			rules: []FieldRule{
				{SourceField: "nickname", DestinationField: "displayName"},
			},
			wantValid: true,
		},
		{
			name:        "invalid empty field mapping",
			source:      Record{},
			destination: Record{},
			rules: []FieldRule{
				{SourceField: "", DestinationField: "id", Required: true},
			},
			wantIssueCodes: []string{"invalid_rule"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.source, tt.destination, tt.rules)

			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; issues: %+v", got.Valid, tt.wantValid, got.Issues)
			}

			if got.Checked != tt.wantChecked {
				t.Errorf("Checked = %d, want %d", got.Checked, tt.wantChecked)
			}

			if len(got.Issues) != len(tt.wantIssueCodes) {
				t.Fatalf("got %d issues, want %d: %+v",
					len(got.Issues), len(tt.wantIssueCodes), got.Issues)
			}

			for i, wantCode := range tt.wantIssueCodes {
				if got.Issues[i].Code != wantCode {
					t.Errorf("issue[%d].Code = %q, want %q",
						i, got.Issues[i].Code, wantCode)
				}
			}
		})
	}
}
