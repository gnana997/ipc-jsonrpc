# Contributing to ipc-jsonrpc

Thank you for your interest in contributing to ipc-jsonrpc! We welcome contributions from the community.

## Development Setup

### Prerequisites

- Node.js >= 18.0.0
- npm or pnpm
- Git

### Getting Started

1. **Fork and clone the repository:**

```bash
git clone https://github.com/gnana997/ipc-jsonrpc.git
cd ipc-jsonrpc
```

2. **Install dependencies:**

```bash
npm install
```

3. **Build the project:**

```bash
npm run build
```

4. **Run tests:**

```bash
npm test
```

5. **Run tests in watch mode:**

```bash
npm run test:watch
```

## Development Workflow

### Running Tests

We use Vitest for testing. Tests must pass before any PR is merged.

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

### Code Quality

We use Biome for linting and formatting.

```bash
# Check code quality
npm run lint

# Auto-fix issues
npm run lint:fix

# Format code
npm run format

# Type check
npm run typecheck
```

### Building

```bash
# Build once
npm run build

# Build in watch mode (for development)
npm run dev
```

## Making Changes

### 1. Create a Branch

Create a feature branch from `main`:

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/add-websocket-support`
- `fix/connection-timeout-issue`
- `docs/improve-readme`

### 2. Make Your Changes

- Write clear, concise commit messages
- Add tests for new features
- Update documentation as needed
- Ensure all tests pass
- Run `npm run lint:fix` before committing

### 3. Add a Changeset

For changes that should be included in release notes:

```bash
npm run changeset
```

Follow the prompts to describe your changes. This helps us maintain the CHANGELOG automatically.

### 4. Commit Your Changes

We follow conventional commit format:

```bash
git commit -m "feat: add support for WebSocket transport"
git commit -m "fix: resolve connection timeout issue"
git commit -m "docs: update installation instructions"
```

**Commit types:**
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions or changes
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

### 5. Push and Create a Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:
- Clear description of changes
- Link to related issues (if any)
- Screenshots/examples (if applicable)

## Coding Standards

### TypeScript

- Use TypeScript strict mode
- Provide type annotations for public APIs
- Avoid `any` types when possible
- Use descriptive variable and function names

### Documentation

- Add JSDoc comments for all public APIs
- Include `@param`, `@returns`, and `@example` tags
- Keep comments clear and concise
- Update README for significant changes

### Testing

- Write tests for all new features
- Maintain or improve code coverage
- Test edge cases and error conditions
- Use descriptive test names

### Code Style

- Follow the project's Biome configuration
- Keep functions small and focused
- Use meaningful variable names
- Add comments only when necessary (code should be self-documenting)

## Project Structure

```
packages/jsonrpc-ipc-client/
├── src/
│   ├── index.ts          # Main entry point
│   ├── client.ts         # JSONRPCClient class
│   └── types.ts          # Type definitions
├── tests/
│   ├── client.test.ts    # Client tests
│   ├── error.test.ts     # Error handling tests
│   └── types.test.ts     # Type tests
├── dist/                 # Build output (generated)
├── coverage/             # Test coverage (generated)
└── package.json          # Package configuration
```

## Pull Request Process

1. **Ensure CI passes** - All tests and linting must pass
2. **Get reviews** - At least one maintainer approval required
3. **Update documentation** - Keep README and CHANGELOG up to date
4. **Squash commits** - We'll squash commits when merging

## Releasing (Maintainers Only)

We use Changesets for version management:

```bash
# Create a new version
npm run version

# Publish to npm
npm run release
```

## Getting Help

- 📖 Read the [README](README.md)
- 🐛 Check [existing issues](https://github.com/gnana997/ipc-jsonrpc/issues)
- 💬 Ask questions in discussions
- 📧 Contact maintainers

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to ipc-jsonrpc!** 🎉
