import * as vscode from 'vscode';
import { JSONRPCClient, JSONRPCError } from 'node-ipc-jsonrpc';

export class IPCClient {
  private client: JSONRPCClient | null = null;
  private outputChannel: vscode.OutputChannel;
  private statusBarItem: vscode.StatusBarItem;

  constructor(
    private readonly socketPath: string,
    outputChannel: vscode.OutputChannel,
    statusBarItem: vscode.StatusBarItem
  ) {
    this.outputChannel = outputChannel;
    this.statusBarItem = statusBarItem;
  }

  async connect(): Promise<void> {
    if (this.client && this.client.isConnected()) {
      throw new Error('Already connected');
    }

    this.updateStatus('$(sync~spin) Connecting...', 'Connecting to IPC server');
    this.log('Connecting to IPC server...');

    this.client = new JSONRPCClient({
      socketPath: this.socketPath,
      debug: false,
      requestTimeout: 30000,
      connectionTimeout: 10000,
    });

    // Set up event listeners
    this.client.on('connected', () => {
      this.log('✓ Connected to server');
      this.updateStatus('$(check) Connected', 'Connected to IPC server');
      vscode.window.showInformationMessage('Successfully connected to IPC server');
    });

    this.client.on('disconnected', () => {
      this.log('✗ Disconnected from server');
      this.updateStatus('$(x) Disconnected', 'Disconnected from IPC server');
    });

    this.client.on('error', (error) => {
      this.log(`✗ Error: ${error.message}`);
      this.updateStatus('$(error) Error', `Error: ${error.message}`);
    });

    this.client.on('notification', (method, params) => {
      this.log(`📩 Notification: ${method} - ${JSON.stringify(params)}`);
      this.handleNotification(method, params);
    });

    try {
      await this.client.connect();
    } catch (error: any) {
      this.updateStatus('$(x) Connection Failed', 'Failed to connect');
      this.log(`✗ Connection failed: ${error.message}`);
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    if (!this.client) {
      return;
    }

    this.log('Disconnecting from server...');
    await this.client.disconnect();
    this.client = null;
    this.updateStatus('$(circle-slash) Disconnected', 'Not connected');
    this.log('✓ Disconnected successfully');
  }

  async request<T = any>(method: string, params?: any): Promise<T> {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('Not connected to server');
    }

    this.log(`→ Request: ${method} ${params ? JSON.stringify(params) : ''}`);

    try {
      const result = await this.client.request<T>(method, params);
      this.log(`← Response: ${JSON.stringify(result)}`);
      return result;
    } catch (error: any) {
      if (error instanceof JSONRPCError) {
        this.log(`✗ RPC Error ${error.code}: ${error.message}`);
        throw new Error(`RPC Error: ${error.message}`);
      }
      this.log(`✗ Request failed: ${error.message}`);
      throw error;
    }
  }

  isConnected(): boolean {
    return this.client?.isConnected() ?? false;
  }

  private handleNotification(method: string, params: any): void {
    switch (method) {
      case 'progress':
        this.handleProgressNotification(params);
        break;
      default:
        vscode.window.showInformationMessage(
          `Notification: ${method} - ${JSON.stringify(params)}`
        );
    }
  }

  private handleProgressNotification(params: any): void {
    const { current, total, percent } = params;
    const message = `Progress: ${current}/${total} (${percent.toFixed(1)}%)`;

    // Show progress in status bar briefly
    this.updateStatus(`$(sync~spin) ${message}`, message);

    // Reset status after a short delay
    setTimeout(() => {
      if (this.isConnected()) {
        this.updateStatus('$(check) Connected', 'Connected to IPC server');
      }
    }, 1000);
  }

  private updateStatus(text: string, tooltip: string): void {
    this.statusBarItem.text = text;
    this.statusBarItem.tooltip = tooltip;
    this.statusBarItem.show();
  }

  private log(message: string): void {
    const timestamp = new Date().toISOString();
    this.outputChannel.appendLine(`[${timestamp}] ${message}`);
  }

  dispose(): void {
    if (this.client) {
      this.client.disconnect().catch((err) => {
        this.log(`Error during disposal: ${err.message}`);
      });
    }
  }
}
