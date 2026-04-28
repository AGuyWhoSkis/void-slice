package validate

import "void-slice/internal/scan"

var Codes = struct {
	ARRAY_COUNT_MISMATCH scan.DiagnosticCode
	ARRAY_INDEX_OOB      scan.DiagnosticCode
	ARRAY_DUP_INDEX      scan.DiagnosticCode
	ARRAY_MISSING_NUM    scan.DiagnosticCode
}{
	ARRAY_COUNT_MISMATCH: "VALIDATE_ARRAY_COUNT_MISMATCH",
	ARRAY_INDEX_OOB:      "VALIDATE_ARRAY_INDEX_OOB",
	ARRAY_DUP_INDEX:      "VALIDATE_ARRAY_DUP_INDEX",
	ARRAY_MISSING_NUM:    "VALIDATE_ARRAY_MISSING_NUM",
}
