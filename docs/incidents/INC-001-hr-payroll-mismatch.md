# Incident Case Study: HR-to-Payroll Data Mismatch

**Incident ID:** INC-001
**System:** HR → Payroll
**Category:** Data transformation / integration validation
**Environment:** Simulated test environment
**Status:** Investigation scenario documented; remediation not yet verified
**Tool:** Integration Guard

---

## 1. Executive Summary

Integration Guard is used to compare records transferred between an HR source system and a Payroll destination system. The purpose is to detect data defects that could otherwise pass unnoticed between systems.

In this simulated incident, employee `00427` has department `FIN` in the source payload, but the destination payload contains `sales` for the corresponding department field. The configured normalization rule handles whitespace and letter case; it does not convert one department code into another. The values therefore remain inconsistent after normalization.

The same failure fixture also contains:
- A source employee record that has no matching destination record.
- A destination employee record that has no matching source record.

These three defects demonstrate why integration validation must check both field-level correctness and record-level consistency.

**Important:** This case study documents a reproducible test scenario. It does not represent a verified production incident, confirmed business impact, or completed root-cause investigation.

## 2. Business Context

HR is the authoritative source for employee information. Payroll consumes that information to maintain its own employee records.

A typical integration transfers fields such as:

| HR source field | Payroll destination field |
|---|---|
| `employee_id` | `staffCode` |
| `email` | `emailAddress` |
| `department` | `departmentCode` |

The systems may use different field names, formats, or casing conventions. A valid integration must distinguish harmless formatting differences from meaningful data changes.

For example, ` USER@EXAMPLE.COM ` and `user@example.com` can be treated as equivalent under the configured `trim_lowercase` normalization rule. By contrast, `FIN` and `SALES` are different department codes and must not be treated as equivalent.

## 3. Incident Symptoms and Evidence

### 3.1 Incorrect department value

The source record contains:

```json
{
  "employee_id": "00427",
  "email": " USER@EXAMPLE.COM ",
  "department": "FIN"
}
```

The corresponding destination record contains:

```json
{
  "staffCode": "00427",
  "emailAddress": "user@example.com",
  "departmentCode": "sales"
}
```

The configured mapping compares `department` with `departmentCode`. Normalization does not explain the discrepancy: trimming whitespace and converting letter case cannot turn `FIN` into `SALES`.

**Confirmed by the fixture:** the source and destination department values differ after normalization.

**Not confirmed by the fixture:** where or why the incorrect value was introduced.

### 3.2 Missing destination record

Source employee `00428` exists in the source payload, but there is no corresponding destination record in the failure fixture.

Potential explanations include a failed transformation, a rejected message, a filtering rule, a delivery failure, or an incomplete destination export. These are hypotheses to investigate, not established causes.

### 3.3 Unexpected destination record

Destination record `00999` exists without a corresponding source record in the failure fixture.

Possible explanations include a stale destination record, an incorrect source extract, a duplicate or replayed message, or an identifier-mapping defect. The available test data alone cannot distinguish these possibilities.

## 4. Investigation Procedure

### Step 1 — Confirm the source record

Retrieve the source payload used for the affected run and verify that employee `00427` has department `FIN`.

Record the source file, extraction timestamp, relevant identifier, and any available version or correlation ID.

**Expected evidence:** the exact source record used by the validation run.

### Step 2 — Inspect the destination record

Retrieve the destination payload and confirm that `staffCode` `00427` has `departmentCode` `sales`.

Check whether the destination record belongs to the same integration run and whether the source and destination snapshots are comparable.

**Expected evidence:** the destination record and its relationship to the source record.

### Step 3 — Check the mapping configuration

Inspect the deployed mapping rules for the HR-to-Payroll integration.

Verify:
- The mapping from `department` to `departmentCode`.
- Any lookup tables that translate department codes.
- Default values or fallback behavior.
- Conditional transformations and filtering rules.
- The configuration version deployed when the data was transferred.

Do not assume that the field mapping itself is defective merely because the resulting values differ. A downstream lookup, stale configuration, or incorrect input could also explain the result.

**Expected evidence:** the applicable configuration and the deployed version.

### Step 4 — Trace the transformation

Follow the affected record through the integration pipeline using available logs, message history, correlation IDs, or audit records.

Compare the department value at each available stage:
1. Source extraction.
2. Transformation input.
3. Transformation output.
4. Destination submission.
5. Persisted destination record.

Identify the earliest stage where the value changes from `FIN` to `SALES`.

For employee `00428`, determine whether the message was extracted, transformed, submitted, rejected, or delivered but not persisted.

For employee `00999`, establish whether the record originated from an earlier run, a different source extract, a replay, or another legitimate process.

**Expected evidence:** a traceable sequence of events sufficient to support a root-cause conclusion.

If logs or message history are unavailable, record that limitation. Do not invent a root cause.

### Step 5 — Validate the fix

After identifying and correcting the cause:

1. Prepare a corrected source/destination fixture or a controlled integration run.
2. Re-run Integration Guard using the same applicable rules.
3. Confirm that employee `00427` has matching normalized department values.
4. Confirm that employee `00428` is present in the destination when required.
5. Investigate and resolve employee `00999`, or document why its presence is legitimate.
6. Run the complete automated test suite.
7. Add a regression test for the confirmed defect.
8. Preserve the validation report and relevant configuration version as evidence.

**Expected evidence:** a passing validation result, test output, and a documented explanation for all previously detected record-level discrepancies.

A successful test alone is not sufficient if the test data was changed to hide the defect or if unexplained records remain.

## 5. Root-Cause Analysis

The root cause remains **undetermined** until the investigation produces supporting evidence.

| Hypothesis | Evidence needed |
|---|---|
| Incorrect department lookup | Lookup-table entries and transformation output |
| Stale mapping configuration | Deployed version and configuration history |
| Incorrect source data | Authoritative HR record and source extract |
| Default or fallback value applied | Transformation conditions and execution logs |
| Partial message delivery | Message status, retry history, and destination audit records |
| Stale or unrelated destination record | Record creation history and source-system comparison |

The investigation should identify the earliest stage at which the observed defect appears. The remediation should target that cause rather than merely changing the destination data to make the comparison pass.

## 6. Remediation Plan

The exact corrective action depends on the evidence gathered during investigation.

Potential corrective actions include:
- Fixing an incorrect department lookup or transformation rule.
- Restoring the intended configuration version.
- Correcting an upstream source-data defect through the appropriate data owner.
- Reprocessing a failed message after confirming that retrying is safe.
- Reconciling stale or unexpected destination records through an authorized procedure.

After remediation, rerun the affected integration and compare the results against the authoritative source.

Any manual correction must be auditable and must not conceal an unresolved transformation defect.

## 7. Regression Testing Strategy

The regression suite should cover both successful and unsuccessful integration scenarios.

| Test scenario | Expected outcome |
|---|---|
| All mapped fields match | Validation passes |
| Whitespace and letter-case differences under `trim_lowercase` | Validation passes |
| Department value changes from `FIN` to `SALES` | Field validation fails |
| Source record has no destination match | Missing destination record is reported |
| Destination record has no source match | Unexpected destination record is reported |
| Duplicate source identifier | Duplicate source key is reported |
| Duplicate destination identifier | Duplicate destination key is reported |
| Required mapped field is absent | Validation fails |
| Malformed JSON input | Operational error is reported |
| Corrected integration dataset | Validation passes with no unresolved issues |

Regression tests should verify both the reported defect and the command's exit status. Invalid integration data must not be mistaken for a successful run.

## 8. Validation Evidence from Integration Guard

The existing failure fixture is designed to exercise three record-level conditions:

1. Employee `00427`: department value mismatch.
2. Employee `00428`: missing destination record.
3. Employee `00999`: unexpected destination record.

The expected behavior is a failed validation result with a structured report containing the relevant issue codes. The CLI should return a nonzero exit status for validation failure.

The existing successful fixture provides the contrasting case: three source records, three destination records, three matched records, and no validation issues.

These are test-fixture expectations and previously observed local test outcomes. They do not establish that a real external HR or Payroll system has been corrected.

## 9. Limitations

Integration Guard currently validates supplied JSON payloads against configured field mappings and rules. It does not, by itself:

- Connect to a live HR or Payroll system.
- Inspect production logs or message brokers.
- Automatically identify the root cause of a mismatch.
- Correct destination data.
- Verify that a deployment or business process is safe to resume.
- Establish the financial or operational impact of an incident.

Those activities require system access, operational evidence, and appropriate authorization. The tool supplies structured validation evidence to support that work.

## 10. Incident Closure Criteria

The incident can be marked resolved only when all applicable conditions are satisfied:

- [ ] The source and destination evidence has been captured.
- [ ] The root cause is supported by evidence, or the remaining uncertainty is explicitly documented.
- [ ] The corrective action has been implemented and reviewed.
- [ ] The corrected dataset passes validation.
- [ ] Missing and unexpected records have been reconciled or explicitly accepted with documented justification.
- [ ] Regression tests cover the confirmed failure mode.
- [ ] The automated test suite passes.
- [ ] The validation report and relevant configuration version are retained.

**Current status:** simulated incident documented; root cause not established; remediation and closure not yet verified.

## Conclusion

This case demonstrates how Integration Guard can detect field-level transformation defects and record-level discrepancies before integration results are accepted.

Its value is not that it automatically diagnoses every integration failure. Its value is that it produces repeatable, structured evidence that an engineer can use to investigate, correct, and test data flowing between systems.
