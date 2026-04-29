import type { Diagnostic } from "../api";

export interface DiagnosticsListProps {
  diagnostics: Diagnostic[];
  onSelect: (line: number) => void;
}

export function DiagnosticsList({ diagnostics, onSelect }: DiagnosticsListProps) {
  if (diagnostics.length === 0) {
    return (
      <div className="vs-diags-empty">
        <p>No diagnostics.</p>
      </div>
    );
  }
  return (
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
  );
}
