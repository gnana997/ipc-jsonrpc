import * as vscode from 'vscode';
import { IPCClient } from './client';

let client: IPCClient | null = null;
let outputChannel: vscode.OutputChannel;
let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
  console.log('IPC JSON-RPC Example extension is now active');

  // Create output channel for logging
  outputChannel = vscode.window.createOutputChannel('IPC JSON-RPC Example');
  context.subscriptions.push(outputChannel);

  // Create status bar item
  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
  statusBarItem.text = '$(circle-slash) IPC: Disconnected';
  statusBarItem.tooltip = 'Click to connect to IPC server';
  statusBarItem.command = 'ipcExample.connect';
  statusBarItem.show();
  context.subscriptions.push(statusBarItem);

  // Initialize client
  client = new IPCClient('echo-server', outputChannel, statusBarItem);
  context.subscriptions.push({
    dispose: () => client?.dispose(),
  });

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand('ipcExample.connect', async () => {
      await connectCommand();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('ipcExample.disconnect', async () => {
      await disconnectCommand();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('ipcExample.echo', async () => {
      await echoCommand();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('ipcExample.uppercase', async () => {
      await uppercaseCommand();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('ipcExample.startNotifications', async () => {
      await startNotificationsCommand();
    })
  );

  outputChannel.appendLine('IPC JSON-RPC Example extension activated');
  outputChannel.appendLine('Available commands:');
  outputChannel.appendLine('  - IPC Example: Connect to Server');
  outputChannel.appendLine('  - IPC Example: Disconnect from Server');
  outputChannel.appendLine('  - IPC Example: Echo Message');
  outputChannel.appendLine('  - IPC Example: Uppercase Text');
  outputChannel.appendLine('  - IPC Example: Start Notifications');
  outputChannel.appendLine('');
  outputChannel.appendLine('Note: Make sure the Go echo server is running first!');
  outputChannel.appendLine('  cd examples/echo && go run main.go');
}

async function connectCommand() {
  if (!client) {
    vscode.window.showErrorMessage('Client not initialized');
    return;
  }

  if (client.isConnected()) {
    vscode.window.showInformationMessage('Already connected to server');
    return;
  }

  try {
    await client.connect();
  } catch (error: any) {
    vscode.window.showErrorMessage(`Failed to connect: ${error.message}`);
  }
}

async function disconnectCommand() {
  if (!client) {
    return;
  }

  if (!client.isConnected()) {
    vscode.window.showInformationMessage('Not connected to server');
    return;
  }

  try {
    await client.disconnect();
    vscode.window.showInformationMessage('Disconnected from server');
  } catch (error: any) {
    vscode.window.showErrorMessage(`Failed to disconnect: ${error.message}`);
  }
}

async function echoCommand() {
  if (!client || !client.isConnected()) {
    vscode.window.showWarningMessage('Not connected to server. Connect first.');
    return;
  }

  const input = await vscode.window.showInputBox({
    prompt: 'Enter a message to echo',
    placeHolder: 'Hello, World!',
    value: 'Hello from VSCode!',
  });

  if (input === undefined) {
    return; // User cancelled
  }

  try {
    const result = await client.request('echo', input);
    vscode.window.showInformationMessage(`Echo response: ${JSON.stringify(result)}`);
  } catch (error: any) {
    vscode.window.showErrorMessage(`Echo failed: ${error.message}`);
  }
}

async function uppercaseCommand() {
  if (!client || !client.isConnected()) {
    vscode.window.showWarningMessage('Not connected to server. Connect first.');
    return;
  }

  const input = await vscode.window.showInputBox({
    prompt: 'Enter text to uppercase',
    placeHolder: 'hello world',
    value: 'hello world',
  });

  if (input === undefined || input === '') {
    return;
  }

  try {
    const result = await client.request<{ result: string }>('uppercase', { text: input });
    vscode.window.showInformationMessage(`Uppercase result: ${result.result}`);
  } catch (error: any) {
    vscode.window.showErrorMessage(`Uppercase failed: ${error.message}`);
  }
}

async function startNotificationsCommand() {
  if (!client || !client.isConnected()) {
    vscode.window.showWarningMessage('Not connected to server. Connect first.');
    return;
  }

  // Show quick pick for count
  const countStr = await vscode.window.showQuickPick(
    ['3', '5', '10', '20'],
    {
      placeHolder: 'How many notifications?',
      title: 'Notification Count',
    }
  );

  if (!countStr) {
    return;
  }

  const count = parseInt(countStr, 10);

  // Show quick pick for interval
  const intervalStr = await vscode.window.showQuickPick(
    ['100', '250', '500', '1000'],
    {
      placeHolder: 'Interval between notifications (ms)?',
      title: 'Notification Interval',
    }
  );

  if (!intervalStr) {
    return;
  }

  const interval = parseInt(intervalStr, 10);

  try {
    await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: 'Receiving server notifications',
        cancellable: false,
      },
      async (progress) => {
        // Start notifications
        const result = await client!.request('startNotifications', { count, interval });

        outputChannel.appendLine(`Server response: ${JSON.stringify(result)}`);

        // Wait for all notifications to arrive
        // (they're handled in the notification event listener)
        await new Promise((resolve) =>
          setTimeout(resolve, count * interval + 1000)
        );

        progress.report({ increment: 100, message: 'Complete!' });
      }
    );

    vscode.window.showInformationMessage(
      `Received ${count} notifications from server`
    );
  } catch (error: any) {
    vscode.window.showErrorMessage(`Start notifications failed: ${error.message}`);
  }
}

export function deactivate() {
  if (client) {
    client.dispose();
  }
  outputChannel.appendLine('IPC JSON-RPC Example extension deactivated');
}
