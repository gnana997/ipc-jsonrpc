# Contributing to ipc-jsonrpc

Thank you for your interest in contributing to ipc-jsonrpc! This monorepo contains both Go server and Node.js client implementations.

## Table of Contents

- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)

## Development Setup

### Prerequisites

- **Go**: 1.21 or higher
- **Node.js**: 18 or higher
- **npm**: 9 or higher
- **Git**: Latest version

### Initial Setup

```bash
# Clone repository
git clone https://github.com/gnana997/ipc-jsonrpc.git
cd ipc-jsonrpc

# Install Node.js dependencies (also installs workspace dependencies)
npm install

# Build Node.js package
npm run build

# Run all tests to verify setup
npm test
```

## Project Structure

```
ipc-jsonrpc/
├── go.mod                          # Root Go module
├── *.go                            # Go server source files
├── node/                           # Node.js client package
│   ├── package.json
│   ├── src/                        # TypeScript source
│   ├── tests/                      # Test files
│   └── dist/                       # Build output (generated)
├── examples/                       # Cross-language examples
│   └── echo/                       # Basic echo example
│       ├── go.mod                  # Separate module for example
│       └── main.go
├── docs/                           # Documentation
│   └── protocol.md                 # Protocol specification
├── .github/workflows/              # CI/CD workflows
│   ├── go.yml                      # Go CI
│   ├── node.yml                    # Node.js CI
│   └── release.yml                 # npm publishing
├── package.json                    # Root workspace config
├── CHANGELOG.md                    # Version history
└── README.md                       # Main documentation
```

## Making Changes

### Branching Strategy

Create a feature branch from `main`:

```bash
git checkout -b feature/your-feature-name
```

**Branch naming conventions:**
- `feature/add-new-handler` - New features
- `fix/connection-timeout` - Bug fixes
- `docs/improve-readme` - Documentation
- `refactor/simplify-codec` - Code refactoring

### Go Changes

#### 1. Edit Go Files

Make changes to `*.go` files in the root directory.

#### 2. Run Tests

```bash
# Run all Go tests
npm run test:go
# or directly:
go test -v -race ./...

# Run specific test
go test -v -run TestServerBasic
```

#### 3. Check Code Quality

```bash
# Run linter
npm run lint:go
# or directly:
golangci-lint run ./...

# Format code
gofmt -w .

# Verify formatting
gofmt -l .
```

#### 4. Update Documentation

- Update godoc comments for public APIs
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Run `go doc` to verify documentation looks good

### Node.js Changes

#### 1. Edit TypeScript Files

Make changes in `node/src/`.

#### 2. Build

```bash
# Build package
npm run build --workspace=node

# Watch mode for development
npm run dev --workspace=node
```

#### 3. Run Tests

```bash
# Run all tests
npm run test:node

# Watch mode
npm run test:watch --workspace=node

# With coverage
npm run test:coverage --workspace=node
```

#### 4. Check Code Quality

```bash
# Run linter
npm run lint:node

# Fix linting issues
npm run lint:fix --workspace=node

# Format code
npm run format --workspace=node

# Type check
npm run typecheck --workspace=node
```

#### 5. Add Changeset

For changes that should be included in release notes:

```bash
npm run changeset
```

Follow prompts to describe your changes. This creates a changeset file that will be used for versioning and changelog generation.

### Documentation Changes

- **Root README**: For monorepo-level documentation
- **GO_README.md**: For Go-specific API documentation
- **node/README.md**: For Node.js-specific API documentation
- **docs/**: For protocol specification and architecture

## Testing

### Running All Tests

```bash
# Run both Go and Node tests
npm test

# Or separately:
npm run test:go
npm run test:node
```

### Writing Tests

#### Go Tests

```go
// server_test.go
func TestMyFeature(t *testing.T) {
    server, err := NewServer(ServerConfig{
        SocketPath: "test-socket",
    })
    if err != nil {
        t.Fatal(err)
    }

    // Test implementation...
}
```

#### Node.js Tests

```typescript
// node/tests/myfeature.test.ts
import { describe, it, expect } from 'vitest';
import { JSONRPCClient } from '../src';

describe('MyFeature', () => {
  it('should work correctly', async () => {
    const client = new JSONRPCClient({ socketPath: 'test' });
    // Test implementation...
    expect(result).toBe(expected);
  });
});
```

### Test Coverage

Maintain or improve test coverage:

```bash
# Go coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Node.js coverage
npm run test:coverage --workspace=node
# Opens coverage report in browser
```

## Code Style

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (automatic in most IDEs)
- Run `golangci-lint` before committing
- Keep functions small and focused
- Use descriptive variable names
- Add godoc comments for exported functions/types

**Example:**

```go
// NewServer creates a new JSON-RPC server with the given configuration.
// It returns an error if the socket path is invalid or cannot be created.
func NewServer(config ServerConfig) (*Server, error) {
    // Implementation...
}
```

### TypeScript Style

- Follow project's Biome configuration
- Use provided npm scripts for linting/formatting
- Prefer `const` over `let`
- Use async/await over callbacks
- Add JSDoc comments for public APIs

**Example:**

```typescript
/**
 * Creates a new JSON-RPC client
 *
 * @param config - Client configuration
 * @returns A new JSONRPCClient instance
 *
 * @example
 * ```typescript
 * const client = new JSONRPCClient({ socketPath: 'myapp' });
 * await client.connect();
 * ```
 */
constructor(config: JSONRPCClientConfig) {
  // Implementation...
}
```

### Commit Messages

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(server): add timeout middleware

Add configurable timeout middleware to prevent long-running requests
from blocking the server.

Closes #123
```

```
fix(client): resolve reconnection race condition

The client could attempt multiple reconnections simultaneously,
leading to connection leaks. This adds proper synchronization.
```

## Pull Request Process

### 1. Ensure Quality

Before opening a PR:

- [ ] All tests pass: `npm test`
- [ ] Code is formatted: `npm run format`
- [ ] Linting passes: `npm run lint`
- [ ] Type checking passes: `npm run typecheck`
- [ ] Add changeset if needed: `npm run changeset`
- [ ] Update documentation if needed

### 2. Create Pull Request

1. Push your branch: `git push origin feature/your-feature`
2. Go to GitHub and create a Pull Request
3. Fill out the PR template completely
4. Link related issues

### 3. PR Review

- At least one maintainer approval required
- All CI checks must pass
- Address review comments
- Keep PR scope focused (one feature/fix per PR)

### 4. Merging

- Maintainers will merge approved PRs
- Commits are squashed on merge
- Delete branch after merge

## Versioning

We use [Changesets](https://github.com/changesets/changesets) for version management:

### Creating a Changeset

```bash
npm run changeset
```

Select:
1. **Packages to version**: Choose `node-ipc-jsonrpc`
2. **Version bump**: Major, minor, or patch
3. **Description**: Summary of changes

### Version Bump Types

- **Major** (1.0.0 → 2.0.0): Breaking changes
- **Minor** (1.0.0 → 1.1.0): New features, backwards compatible
- **Patch** (1.0.0 → 1.0.1): Bug fixes, backwards compatible

### Release Process (Maintainers Only)

```bash
# Update versions and CHANGELOG
npm run version

# Publish to npm
npm run release
```

## Getting Help

- 📖 Read the [README](./README.md)
- 📚 Check [Documentation](./docs/)
- 🐛 Search [Issues](https://github.com/gnana997/ipc-jsonrpc/issues)
- 💬 Start a [Discussion](https://github.com/gnana997/ipc-jsonrpc/discussions)
- 📧 Contact maintainers

## Code of Conduct

Please read and follow our [Code of Conduct](./CODE_OF_CONDUCT.md) - currently in node/ directory.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to ipc-jsonrpc!** 🎉
