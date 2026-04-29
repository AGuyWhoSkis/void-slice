import { useEffect, useRef } from "react";
import { EditorState, Compartment } from "@codemirror/state";
import { EditorView, lineNumbers, gutter, GutterMarker, keymap } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import type { Diagnostic } from "../api";

class DiagnosticMarker extends GutterMarker {
  constructor(private severity: "error" | "warning") {
    super();
  }
  toDOM() {
    const el = document.createElement("div");
    el.className = `vs-gutter vs-gutter-${this.severity}`;
    el.title = this.severity;
    return el;
  }
}

function diagnosticGutter(diagnostics: Diagnostic[]) {
  const byLine = new Map<number, "error" | "warning">();
  for (const d of diagnostics) {
    const existing = byLine.get(d.line);
    if (existing === "error") continue;
    byLine.set(d.line, d.severity);
  }
  return gutter({
    class: "vs-diag-gutter",
    lineMarker(view, line) {
      const lineNum = view.state.doc.lineAt(line.from).number;
      const sev = byLine.get(lineNum);
      return sev ? new DiagnosticMarker(sev) : null;
    },
  });
}

export interface EditorProps {
  value: string;
  diagnostics: Diagnostic[];
  scrollToLine?: { line: number; nonce: number } | null;
  onChange: (value: string) => void;
}

export function Editor({ value, diagnostics, scrollToLine, onChange }: EditorProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const viewRef = useRef<EditorView | null>(null);
  const diagCompRef = useRef(new Compartment());
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    if (!hostRef.current) return;

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        diagCompRef.current.of(diagnosticGutter(diagnostics)),
        EditorView.theme({
          "&": { height: "100%", fontSize: "13px" },
          ".cm-scroller": { fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" },
        }),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) {
            onChangeRef.current(u.state.doc.toString());
          }
        }),
      ],
    });

    const view = new EditorView({ state, parent: hostRef.current });
    viewRef.current = view;
    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    if (view.state.doc.toString() !== value) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value } });
    }
  }, [value]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({ effects: diagCompRef.current.reconfigure(diagnosticGutter(diagnostics)) });
  }, [diagnostics]);

  useEffect(() => {
    if (!scrollToLine) return;
    const view = viewRef.current;
    if (!view) return;
    const lineNum = Math.max(1, Math.min(view.state.doc.lines, scrollToLine.line));
    const line = view.state.doc.line(lineNum);
    view.dispatch({
      selection: { anchor: line.from },
      effects: EditorView.scrollIntoView(line.from, { y: "center" }),
    });
    view.focus();
  }, [scrollToLine]);

  return <div ref={hostRef} className="vs-editor" />;
}
