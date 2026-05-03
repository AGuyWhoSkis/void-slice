import { useCallback, useEffect, useState } from "react";
import { Editor } from "./Editor";
import { DiagnosticsList } from "./DiagnosticsList";
import { lintFile, type Diagnostic } from "../api";
import { SAMPLES } from "../samples";

type Status =
  | { kind: "empty" }
  | { kind: "loading" }
  | { kind: "ok" }
  | { kind: "error"; message: string };

export function Playground() {
  const [filename, setFilename] = useState<string>("");
  const [text, setText] = useState<string>("");
  const [diagnostics, setDiagnostics] = useState<Diagnostic[]>([]);
  const [status, setStatus] = useState<Status>({ kind: "empty" });
  const [scrollTarget, setScrollTarget] = useState<{
    line: number;
    nonce: number;
  } | null>(null);
  const [dragOver, setDragOver] = useState(false);

  const runLint = useCallback(
    async (name: string, body: string, signal: AbortSignal) => {
      setStatus({ kind: "loading" });
      try {
        const res = await lintFile(name, body, signal);
        setDiagnostics(res.diagnostics);
        setStatus({ kind: "ok" });
      } catch (err) {
        if (signal.aborted) return;
        const message = err instanceof Error ? err.message : "lint failed";
        setStatus({ kind: "error", message });
        setDiagnostics([]);
      }
    },
    [],
  );

  useEffect(() => {
    if (text === "") return;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      runLint(filename || "untitled", text, controller.signal);
    }, 500);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [filename, text, runLint]);

  const loadSample = useCallback((sampleFilename: string) => {
    const sample = SAMPLES.find((s) => s.filename === sampleFilename);
    if (!sample) return;
    setFilename(sample.filename);
    setText(sample.body);
  }, []);

  const handleFile = useCallback(async (file: File) => {
    const body = await file.text();
    setFilename(file.name);
    setText(body);
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setDragOver(false);
      const file = e.dataTransfer.files?.[0];
      if (file) handleFile(file);
    },
    [handleFile],
  );

  const onPickFile = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) handleFile(file);
      e.target.value = "";
    },
    [handleFile],
  );

  const selectDiagnostic = useCallback((line: number) => {
    setScrollTarget({ line, nonce: Date.now() });
  }, []);

  return (
    <section id="playground" className="vs-playground">
      <div className="vs-playground-header">
        <h2>Playground</h2>
        <div className="vs-sample-row">
          <span className="vs-sample-label">Try a sample:</span>
          {SAMPLES.map((s) => (
            <button
              key={s.filename}
              type="button"
              className="vs-sample-chip"
              onClick={() => loadSample(s.filename)}
              title={s.hint}
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>

      <div
        className={`vs-dropzone ${dragOver ? "vs-dropzone-active" : ""}`}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
      >
        <p>
          Drop a <code>.decl</code>, <code>.entities</code>, or{" "}
          <code>.entitydef</code> file here, or{" "}
          <label className="vs-file-trigger">
            choose a file
            <input type="file" onChange={onPickFile} hidden />
          </label>
          .
        </p>
      </div>

      <div className="vs-workbench">
        <div className="vs-workbench-main">
          <div className="vs-workbench-titlebar">
            <span className="vs-workbench-filename">
              {filename || "no file loaded"}
            </span>
            <span className={`vs-status vs-status-${status.kind}`}>
              {status.kind === "loading" && "linting…"}
              {status.kind === "error" && status.message}
              {status.kind === "ok" &&
                (diagnostics.length === 0
                  ? "clean"
                  : `${diagnostics.length} diagnostic${diagnostics.length === 1 ? "" : "s"}`)}
              {status.kind === "empty" && "drop a file or click a sample"}
            </span>
          </div>
          <Editor
            value={text}
            diagnostics={diagnostics}
            scrollToLine={scrollTarget}
            onChange={setText}
          />
        </div>
        <aside className="vs-workbench-side">
          <h3>Diagnostics</h3>
          <DiagnosticsList
            diagnostics={diagnostics}
            onSelect={selectDiagnostic}
          />
        </aside>
      </div>
    </section>
  );
}
