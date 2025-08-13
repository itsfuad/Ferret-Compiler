import { workspace, ExtensionContext, commands, window } from "vscode";
import { spawn, ChildProcess } from "child_process";
import * as net from "net";

import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  StreamInfo,
  State,
} from "vscode-languageclient/node";

let client: LanguageClient | null = null;
let serverProcess: ChildProcess | null = null;
let isLSPEnabled: boolean = true;
let connectionSocket: net.Socket | null = null;

function createSocketConnection(port: number): Promise<StreamInfo> {
  return new Promise<StreamInfo>((resolve, reject) => {
    const socket = net.createConnection(port, '127.0.0.1');
    connectionSocket = socket;
    
    socket.on('connect', () => {
      console.log(`Connected to Ferret LSP server on port ${port}`);
      resolve({
        reader: socket,
        writer: socket
      });
    });
    
    socket.on('error', (error) => {
      console.error(`Failed to connect to LSP server: ${error}`);
      connectionSocket = null;
      reject(error);
    });

    socket.on('close', () => {
      console.log('LSP server connection closed');
      connectionSocket = null;
    });

    socket.on('end', () => {
      console.log('LSP server connection ended');
      connectionSocket = null;
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
  // If needed, get desired port from settings or default to dynamic
  const config = workspace.getConfiguration('ferretLanguageServer');
  const desiredPort = config.get<number>('port', 0); // 0 = dynamic
  console.log(`Launching Ferret LSP with port=${desiredPort === 0 ? 'dynamic' : desiredPort}`);

  serverProcess = spawn(serverExec, ['--port', desiredPort.toString()]); // pass port to server
  
  serverProcess.stdout?.on('data', (data: Buffer) => {
    handleServerData(data, resolve, reject);
  });
  
  serverProcess.stderr?.on('data', (data: Buffer) => {
    console.log(`LSP Server: ${data.toString()}`);
  });
  
  serverProcess.on('error', (error) => {
    console.error(`Failed to start LSP server: ${error}`);
    cleanupServerProcess();
    reject(error);
  });
  
  serverProcess.on('exit', (code) => {
    console.log(`LSP server exited with code ${code}`);
    cleanupServerProcess();
  });
}
//Client ferret Language Server: connection to server is erroring. Cannot call write after a stream was destroyed

function cleanupServerProcess(): void {
  if (connectionSocket) {
    connectionSocket.removeAllListeners();
    connectionSocket.unref(); // Unreference to allow process to exit
    //connectionSocket.destroy();
  }
  if (serverProcess) {
    try {
      serverProcess.kill();
    } catch (error) {
      console.error('Error killing server process:', error);
    }
  }
  connectionSocket = null;
  serverProcess = null;
}

// Function to start the LSP client
function startLSPClient(context: ExtensionContext) {
  if (client) {
    console.log("LSP client already exists, skipping start");
    return;
  }

  console.log("Starting Ferret LSP client...");
  
  //use the "ferret-lsp" from PATH
  const serverExec = "ferret-lsp";

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
    errorHandler: {
      error: (error, message, count) => {
        console.error(`LSP Client Error: ${error}, Message: ${typeof message === 'object' ? JSON.stringify(message) : String(message)}, Count: ${count}`);
        if (count !== undefined && count <= 3) {
          return { action: 1 }; // Continue
        }
        return { action: 2 }; // Shutdown
      },
      closed: () => {
        console.log('LSP server connection closed');
        cleanupServerProcess();
        return { action: 1 }; // Restart
      }
    }
  };

  // Create the language client
  client = new LanguageClient(
    "ferretLanguageServer",
    "ferret Language Server",
    serverOptions,
    clientOptions
  );

  // Add state change handler
  client.onDidChangeState((event) => {
    console.log(`LSP Client state changed: ${State[event.oldState]} -> ${State[event.newState]}`);
    if (event.newState === State.Stopped) {
      cleanupServerProcess();
    }
  });

  // Start the client (this will launch the server automatically)
  client.start();
  console.log("Ferret LSP client started");
}

// Function to stop the LSP client
async function stopLSPClient(): Promise<void> {
  console.log("Stopping Ferret LSP client...");
  if (client) {
    try {
      await client.stop();
      client = null;
      cleanupServerProcess();
      console.log("LSP client stopped successfully");
    } catch (error) {
      console.error("Error stopping LSP client:", error);
    }
  }
}

// Function to toggle LSP server
async function toggleLSP(context: ExtensionContext) {
  const config = workspace.getConfiguration('ferretLanguageServer');
  const currentState = config.get<boolean>('enabled', true);
  const newState = !currentState;
  
  try {
    if (newState) {
      // Enable LSP - clean restart
      if (client) {
        await stopLSPClient();
      }
      
      startLSPClient(context);
      await config.update('enabled', newState, true);
      isLSPEnabled = newState;
      window.showInformationMessage('Ferret LSP Server enabled');
    } else {
      // Disable LSP - just stop
      await stopLSPClient();
      await config.update('enabled', newState, true);
      isLSPEnabled = newState;
      window.showInformationMessage('Ferret LSP Server disabled (syntax highlighting only)');
    }
  } catch (error) {
    console.error("Error toggling LSP:", error);
    window.showErrorMessage(`Failed to toggle LSP: ${error}`);
  }
}

export function activate(context: ExtensionContext) {
  // Get initial LSP state from configuration
  const config = workspace.getConfiguration('ferretLanguageServer');
  isLSPEnabled = config.get<boolean>('enabled', true);

  // Register toggle command
  const toggleCommand = commands.registerCommand('ferret.toggleLSP', () => {
    toggleLSP(context);
  });
  context.subscriptions.push(toggleCommand);

  // Start LSP if enabled
  if (isLSPEnabled) {
    startLSPClient(context);
  } else {
    console.log("Ferret LSP is disabled. Only syntax highlighting is active.");
  }
}

export function deactivate(): Thenable<void> | undefined {
  console.log("Deactivating Ferret LSP extension...");
  if (client) {
    return stopLSPClient().then(() => undefined);
  }
  
  // Ensure cleanup even if no client
  cleanupServerProcess();
  return undefined;
}
