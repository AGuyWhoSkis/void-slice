import { ExtensionContext } from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(_context: ExtensionContext): void {
  const serverOptions: ServerOptions = {
    run: { command: "voidslice", args: ["lsp"], transport: TransportKind.stdio },
    debug: { command: "voidslice", args: ["lsp"], transport: TransportKind.stdio },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "voidslice-decl" },
      { scheme: "file", language: "voidslice-entitydef" },
      { scheme: "file", language: "voidslice-entities" },
    ],
  };

  client = new LanguageClient(
    "voidslice",
    "voidslice",
    serverOptions,
    clientOptions
  );

  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  return client?.stop();
}
