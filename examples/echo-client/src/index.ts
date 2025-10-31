import { JSONRPCClient } from 'node-ipc-jsonrpc';

// ANSI color codes for better console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  blue: '\x1b[34m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
};

function log(message: string, color: keyof typeof colors = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logSection(title: string) {
  console.log(`\n${colors.bright}${colors.cyan}${'='.repeat(60)}${colors.reset}`);
  console.log(`${colors.bright}${colors.cyan}  ${title}${colors.reset}`);
  console.log(`${colors.bright}${colors.cyan}${'='.repeat(60)}${colors.reset}\n`);
}

async function main() {
  log('Echo Client Example', 'bright');
  log('Connecting to echo server...', 'blue');

  // Create client
  const client = new JSONRPCClient({
    socketPath: 'echo-server',
    debug: true,
    requestTimeout: 30000,
    connectionTimeout: 10000,
  });

  // Event listeners
  client.on('connected', () => {
    log('✓ Connected to server', 'green');
  });

  client.on('disconnected', () => {
    log('✗ Disconnected from server', 'yellow');
  });

  client.on('error', (error) => {
    log(`✗ Error: ${error.message}`, 'red');
  });

  client.on('notification', (method, params) => {
    log(`📩 Notification received: ${method}`, 'magenta');
    console.log('   Params:', JSON.stringify(params, null, 2));
  });

  try {
    // Connect to server
    await client.connect();

    // ========================================
    // Demo 1: Echo with different data types
    // ========================================
    logSection('Demo 1: Echo Method');

    // Echo string
    log('Sending string...', 'blue');
    const stringResult = await client.request('echo', 'Hello, World!');
    log(`Response: ${JSON.stringify(stringResult)}`, 'green');

    // Echo object
    log('\nSending object...', 'blue');
    const objectResult = await client.request('echo', {
      message: 'Hello from client',
      timestamp: new Date().toISOString(),
      nested: { value: 42 },
    });
    log(`Response: ${JSON.stringify(objectResult, null, 2)}`, 'green');

    // Echo array
    log('\nSending array...', 'blue');
    const arrayResult = await client.request('echo', [1, 2, 3, 'four', { five: 5 }]);
    log(`Response: ${JSON.stringify(arrayResult)}`, 'green');

    // Echo null
    log('\nSending null...', 'blue');
    const nullResult = await client.request('echo', null);
    log(`Response: ${JSON.stringify(nullResult)}`, 'green');

    // ========================================
    // Demo 2: Uppercase method (typed handler)
    // ========================================
    logSection('Demo 2: Uppercase Method');

    log('Sending text to uppercase...', 'blue');
    const uppercaseResult = await client.request<{ result: string }>('uppercase', {
      text: 'hello world',
    });
    log(`Response: ${JSON.stringify(uppercaseResult)}`, 'green');

    // Test validation (empty text should error)
    log('\nTesting validation (empty text)...', 'blue');
    try {
      await client.request('uppercase', { text: '' });
    } catch (error: any) {
      log(`✓ Validation error caught: ${error.message}`, 'yellow');
    }

    // ========================================
    // Demo 3: Notifications
    // ========================================
    logSection('Demo 3: Server Notifications');

    log('Starting notification sequence (5 notifications, 500ms interval)...', 'blue');

    // Set up notification counter
    let notificationCount = 0;
    const notificationHandler = (method: string, params: any) => {
      if (method === 'progress') {
        notificationCount++;
        const { current, total, percent } = params;
        const progressBar = '█'.repeat(Math.floor(percent / 5)) + '░'.repeat(20 - Math.floor(percent / 5));
        log(`Progress [${progressBar}] ${percent.toFixed(1)}% (${current}/${total})`, 'cyan');
      }
    };

    client.on('notification', notificationHandler);

    // Start notifications
    const notifResult = await client.request('startNotifications', {
      count: 5,
      interval: 500, // 500ms between notifications
    });
    log(`Server response: ${JSON.stringify(notifResult)}`, 'green');

    // Wait for all notifications to arrive
    log('\nWaiting for notifications...', 'blue');
    await new Promise((resolve) => setTimeout(resolve, 3000));

    log(`\n✓ Received ${notificationCount} notifications`, 'green');

    // Clean up listener
    client.off('notification', notificationHandler);

    // ========================================
    // Cleanup
    // ========================================
    logSection('Cleanup');
    log('Disconnecting from server...', 'blue');
    await client.disconnect();
    log('✓ Disconnected successfully', 'green');

    log('\n✓ All demos completed successfully!', 'bright');
  } catch (error: any) {
    log(`\n✗ Error: ${error.message}`, 'red');
    if (error.code) {
      log(`  Error code: ${error.code}`, 'red');
    }
    if (error.data) {
      log(`  Additional data: ${JSON.stringify(error.data)}`, 'red');
    }
    process.exit(1);
  }
}

// Handle Ctrl+C gracefully
process.on('SIGINT', () => {
  log('\n\nReceived SIGINT, exiting...', 'yellow');
  process.exit(0);
});

// Run main
main().catch((error) => {
  log(`Fatal error: ${error.message}`, 'red');
  process.exit(1);
});
