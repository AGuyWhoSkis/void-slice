import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { lintFile, type LintResponse } from "./api";

type FetchCapture = { url: string; init: RequestInit };

function installFetch(
  responder: (input: RequestInfo | URL, init: RequestInit) => Response | Promise<Response>,
): { capture: FetchCapture | null } {
  const ref: { capture: FetchCapture | null } = { capture: null };
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    ref.capture = { url: String(input), init: init ?? {} };
    return responder(input, init ?? {});
  }) as typeof fetch;
  return ref;
}

function headerValue(headers: HeadersInit | undefined, name: string): string | undefined {
  if (!headers) return undefined;
  if (headers instanceof Headers) return headers.get(name) ?? undefined;
  if (Array.isArray(headers)) {
    const found = headers.find(([k]) => k.toLowerCase() === name.toLowerCase());
    return found?.[1];
  }
  const rec = headers as Record<string, string>;
  const key = Object.keys(rec).find((k) => k.toLowerCase() === name.toLowerCase());
  return key ? rec[key] : undefined;
}

const originalFetch = globalThis.fetch;

describe("lintFile transport", () => {
  beforeEach(() => {
    globalThis.fetch = originalFetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("constructs URL with API_URL fallback and encodes filename", async () => {
    const ref = installFetch(() => new Response("{\"file\":\"x\",\"diagnostics\":[]}", { status: 200 }));
    await lintFile("a b?c&d.entities", "src");
    expect(ref.capture?.url).toBe(
      "http://localhost:8080/lint?filename=a%20b%3Fc%26d.entities",
    );
  });

  it("sends POST with octet-stream Content-Type and raw body", async () => {
    const ref = installFetch(() => new Response("{\"file\":\"x\",\"diagnostics\":[]}", { status: 200 }));
    await lintFile("f.entities", "raw bytes here");
    expect(ref.capture?.init.method).toBe("POST");
    expect(headerValue(ref.capture?.init.headers, "Content-Type")).toBe("application/octet-stream");
    expect(ref.capture?.init.body).toBe("raw bytes here");
  });

  it("returns LintResponse with diagnostics array intact on 200", async () => {
    const payload: LintResponse = {
      file: "x.entities",
      diagnostics: [
        { line: 1, col: 2, severity: "error", code: "SCAN_X", message: "m" },
        { line: 9, col: 4, severity: "warning", code: "VALIDATE_Y", message: "n" },
      ],
    };
    installFetch(() => new Response(JSON.stringify(payload), { status: 200 }));
    const got = await lintFile("x.entities", "src");
    expect(got).toEqual(payload);
  });

  it("rejects with status code and body text on non-2xx", async () => {
    installFetch(
      () => new Response("body text", { status: 413, statusText: "Payload Too Large" }),
    );
    await expect(lintFile("f.entities", "src")).rejects.toThrowError(/413.*body text/);
  });

  it("falls back to statusText when res.text() itself throws", async () => {
    const fakeRes = {
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      text: () => Promise.reject(new Error("stream broken")),
    } as unknown as Response;
    installFetch(() => fakeRes);
    await expect(lintFile("f.entities", "src")).rejects.toThrowError(
      /500.*Internal Server Error/,
    );
  });
});
