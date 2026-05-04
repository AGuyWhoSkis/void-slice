package validate

import "void-slice/internal/scan"

var Codes = struct {
	ARRAY_INDEX_OOB scan.DiagnosticCode
	ARRAY_DUP_INDEX scan.DiagnosticCode
}{
	ARRAY_INDEX_OOB: "VALIDATE_ARRAY_INDEX_OOB",
	ARRAY_DUP_INDEX: "VALIDATE_ARRAY_DUP_INDEX",
}
