export type Severity = "error" | "warning";

export interface Diagnostic {
  line: number;
  col: number;
  severity: Severity;
  code: string;
  message: string;
}

export interface LintResponse {
  file: string;
  diagnostics: Diagnostic[];
}

const API_URL: string =
  (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8080";

export async function lintFile(
  filename: string,
  src: string,
  signal?: AbortSignal,
): Promise<LintResponse> {
  const url = `${API_URL}/lint?filename=${encodeURIComponent(filename)}`;
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: src,
    signal,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`lint failed (${res.status}): ${text || res.statusText}`);
  }
  return (await res.json()) as LintResponse;
}
