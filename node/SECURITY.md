# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of ipc-jsonrpc seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please do NOT:

- Open a public GitHub issue for security vulnerabilities
- Discuss the vulnerability in public forums, social media, or mailing lists

### Please DO:

**Report security vulnerabilities privately via GitHub Security Advisories:**

1. Go to the [Security tab](https://github.com/gnana997/ipc-jsonrpc/security) of our repository
2. Click "Report a vulnerability"
3. Fill out the form with details about the vulnerability

**Or email us directly at:**

gnana097@gmail.com

### What to include in your report:

- Type of vulnerability (e.g., buffer overflow, injection, authentication bypass)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the vulnerability (what an attacker could do)
- Any potential mitigations you've identified

### What to expect:

- **Acknowledgment**: We'll acknowledge receipt within 48 hours
- **Updates**: We'll provide regular updates on our progress
- **Timeline**: We aim to fix critical vulnerabilities within 7 days
- **Credit**: With your permission, we'll credit you in the security advisory
- **Disclosure**: We'll coordinate public disclosure with you

## Security Best Practices

When using ipc-jsonrpc:

### IPC Socket Security

1. **Unix Sockets:**
   - Use appropriate file permissions (typically 0600 or 0700)
   - Place sockets in protected directories (e.g., `/tmp`, `~/.config`)
   - Clean up socket files on exit

2. **Named Pipes (Windows):**
   - Be aware that named pipes are accessible to any process
   - Consider using Windows security descriptors for access control
   - Validate all incoming data

### Input Validation

- Always validate and sanitize data received from the server
- Don't trust server responses blindly
- Implement proper error handling

### Connection Security

- Use timeouts to prevent resource exhaustion
- Implement reconnection limits
- Monitor for suspicious activity

### Dependencies

- Keep ipc-jsonrpc and its dependencies up to date
- Regularly audit your dependency tree with `npm audit`
- Use tools like Dependabot or Snyk for automated updates

## Known Security Considerations

### IPC Transport Limitations

- **No encryption**: IPC sockets provide NO encryption by default
- **Local-only**: Designed for same-machine communication
- **Trust model**: Assumes the local system is trusted

**If you need:**
- Remote communication → Use HTTPS or WebSocket with TLS
- Encryption → Wrap IPC in an encrypted tunnel (e.g., SSH)
- Authentication → Implement at the application layer

### Process Isolation

IPC communication happens between processes on the same machine. Ensure:
- Server processes run with appropriate privileges
- Client processes are trusted
- Sensitive data is protected at rest and in transit

## Security Updates

We will announce security updates through:
- GitHub Security Advisories
- Release notes in CHANGELOG.md
- npm advisory system

Subscribe to repository releases to stay informed.

## Policy Updates

This security policy may be updated from time to time. Please check back periodically for changes.

---

**Last updated:** 2025-10-31

Thank you for helping keep ipc-jsonrpc and its users safe!
