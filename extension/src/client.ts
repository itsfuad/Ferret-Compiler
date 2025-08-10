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

function createSocketConnection(port: number): Promise<StreamInfo> {
  return new Promise<StreamInfo>((resolve, reject) => {
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
  });
}

function handleServerData(data: Buffer, resolve: (value: StreamInfo) => void, reject: (reason?: any) => void): void {
  const output = data.toString();
  const portMatch = /PORT:(\d+)/.exec(output);
  if (portMatch) {
    const port = parseInt(portMatch[1], 10);
    if (isNaN(port) || port < 1 || port > 65535) {
      reject(new Error(`Invalid port number: ${portMatch[1]}`));
      return;
    }
    console.log(`Connecting to Ferret LSP server on port ${port}`);
    
    createSocketConnection(port)
      .then(resolve)
      .catch(reject);
  }
}

function setupServerProcess(serverExec: string, resolve: (value: StreamInfo) => void, reject: (reason?: any) => void): void {
  serverProcess = spawn(serverExec);
  
  serverProcess.stdout?.on('data', (data: Buffer) => {
    handleServerData(data, resolve, reject);
  });
  
  serverProcess.stderr?.on('data', (data: Buffer) => {
    console.log(`LSP Server: ${data.toString()}`);
  });
  
  serverProcess.on('error', (error) => {
    console.error(`Failed to start LSP server: ${error}`);
    reject(error);
  });
  
  serverProcess.on('exit', (code) => {
    console.log(`LSP server exited with code ${code}`);
  });
}

export function activate(context: ExtensionContext) {
  const serverExec = context.asAbsolutePath(
    path.join("bin", "ferret-lsp.exe")
  );

  // Create server options that spawn the server and connect via TCP
  const serverOptions: ServerOptions = () => {
    return new Promise<StreamInfo>((resolve, reject) => {
      setupServerProcess(serverExec, resolve, reject);
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
