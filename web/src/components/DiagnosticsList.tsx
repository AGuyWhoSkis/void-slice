import type { Diagnostic } from "../api";

export interface DiagnosticsListProps {
  diagnostics: Diagnostic[];
  onSelect: (line: number) => void;
}

export function DiagnosticsList({ diagnostics, onSelect }: DiagnosticsListProps) {
  return (
    <>
      {diagnostics.length === 0 ? (
        <div className="vs-diags-empty">
          <p>No diagnostics.</p>
        </div>
      ) : (
        <ul className="vs-diags-list">
          {diagnostics.map((d, i) => (
            <li key={i} className={`vs-diag vs-diag-${d.severity}`}>
              <button
                type="button"
                className="vs-diag-button"
                onClick={() => onSelect(d.line)}
              >
                <span className="vs-diag-pos">
                  {d.line}:{d.col}
                </span>
                <span className="vs-diag-code">{d.code}</span>
                <span className="vs-diag-message">{d.message}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <details className="vs-diags-limitations">
        <summary>Linter limitations</summary>
        <ul>
          <li>
            <strong>Small edits can produce many errors</strong> when they change the
            file's parse shape. A single quote-byte flip can extend an orphan string
            literal across multiple following statements — the diagnostic count
            reflects the structural reach of the edit, not the edit's byte count.
          </li>
          <li>
            <strong>Non-ASCII bytes in code positions</strong> trigger scan errors. A
            comment containing non-ASCII characters (e.g. accented letters in author
            notes) will produce errors if uncommented — the linter can't tell that the
            bytes were author-intended.
          </li>
        </ul>
      </details>
    </>
  );
}
