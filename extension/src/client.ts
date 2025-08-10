import * as path from "path";
import { workspace, ExtensionContext } from "vscode";
import { spawn, ChildProcess } from "child_process";
import * as net from "net";

import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  StreamInfo,
} from "vscode-languageclient/node";

let client: LanguageClient;
let serverProcess: ChildProcess | null = null;

export function activate(context: ExtensionContext) {
  const serverExec = context.asAbsolutePath(
    path.join("bin", "ferret-lsp.exe")
  );

  // Create server options that spawn the server and connect via TCP
  const serverOptions: ServerOptions = () => {
    return new Promise<StreamInfo>((resolve, reject) => {
      // Start the LSP server process
      serverProcess = spawn(serverExec);
      
      serverProcess.stdout?.on('data', (data: Buffer) => {
        const output = data.toString();
        const portMatch = /PORT:(\d+)/.exec(output);
        if (portMatch) {
          const port = parseInt(portMatch[1]);
          console.log(`Connecting to Ferret LSP server on port ${port}`);
          
          // Create TCP connection to the server
          const socket = net.createConnection(port, '127.0.0.1');
          
          socket.on('connect', () => {
            console.log(`Connected to Ferret LSP server on port ${port}`);
            resolve({
              reader: socket,
              writer: socket
            });
          });
          
          socket.on('error', (error) => {
            console.error(`Failed to connect to LSP server: ${error}`);
            reject(error);
          });
        }
      });
      
      serverProcess.stderr?.on('data', (data: Buffer) => {
        // Log server stderr for debugging (this is where server logs go)
        console.log(`LSP Server: ${data.toString()}`);
      });
      
      serverProcess.on('error', (error) => {
        console.error(`Failed to start LSP server: ${error}`);
        reject(error);
      });
      
      serverProcess.on('exit', (code) => {
        console.log(`LSP server exited with code ${code}`);
      });
    });
  };

  // Options to control the language client
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "ferret-fer" }],
    synchronize: {
      // Notify the server about file changes to .fer files contained in the workspace
      fileEvents: workspace.createFileSystemWatcher("**/*.fer"),
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
  // Clean up the server process
  if (serverProcess) {
    serverProcess.kill();
    serverProcess = null;
  }
  
  if (!client) {
    return undefined;
  }
  return client.stop();
}
