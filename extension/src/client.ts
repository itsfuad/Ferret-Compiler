import * as path from "path";
import { workspace, ExtensionContext } from "vscode";

import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(context: ExtensionContext) {
  const serverExec = context.asAbsolutePath(
    path.join("bin", "ferret-lsp.exe")
  );

  // Define the server's socket connection options
  const serverOptions: ServerOptions = {
    run: {
      command: serverExec,
      transport: {
        kind: TransportKind.socket,
        port: 8487,
      },
    },
    debug: {
      command: serverExec,
      transport: {
        kind: TransportKind.socket,
        port: 8487,
      }
    },
  }

  // Options to control the language client
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "ferret" }],
    synchronize: {
      // Notify the server about file changes to .fer files contained in the workspace
      fileEvents: workspace.createFileSystemWatcher("**/*.{wal,ferret}"),
    },
  };

  // Create the language client and start the client.
  client = new LanguageClient(
    "ferretLanguageServer",
    "ferret Language Server",
    serverOptions,
    clientOptions
  );

  // Start the client. This will also launch the server
  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
