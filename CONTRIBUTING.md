# Contributing to uigraph-cli

Thank you for considering contributing to uigraph-cli! This document provides guidelines for contributing to the project.

## Philosophy

The uigraph-cli is designed with a very specific philosophy:

> **The CLI is a sync courier, not a brain.**

This means:
- ✅ It reads config files and git metadata
- ✅ It sends data to the gateway backend
- ❌ It does NOT infer, diff, or make intelligent decisions
- ❌ It does NOT maintain state or cache
- ❌ It does NOT provide interactive features


## Development Setup

### Prerequisites

- Go 1.23 or higher
- Git
- Make (optional, but recommended)

### Clone and Build

```bash
git clone https://github.com/uigraph-app/uigraph-cli.git
cd uigraph-cli
make build
```

### Run Tests

```bash
make test
```

### Run Example

```bash
cd examples
UIGRAPH_TOKEN=test-token ../bin/uigraph sync --dry-run
```

## Project Structure

```
uigraph-cli/
├── cmd/              # CLI commands
│   ├── root.go      # Root command setup
│   └── sync.go      # Sync command implementation
├── pkg/              # Core packages
│   ├── config/      # Config loading and validation
│   ├── git/         # Git metadata capture
│   └── gateway/     # Gateway API client
├── examples/        # Example configs and CI/CD workflows
└── main.go          # Entry point
```

## Making Changes

### Before You Start

1. Check existing issues or create one to discuss your changes
2. Ensure your change aligns with the CLI's philosophy (see above)
3. If adding features, ask yourself: "Should this be in the gateway instead?"

### Code Guidelines

1. **Keep it simple** - This CLI should be easy to understand and maintain
2. **Fail fast** - Clear, immediate error messages
3. **No magic** - Explicit behavior, no hidden state
4. **Stateless** - No local caching or state files
5. **CI/CD first** - Design for non-interactive execution

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Keep functions small and focused
- Add comments for non-obvious logic
- Write tests for new functionality

### Testing

All new code should include tests:

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests for specific package
go test ./pkg/config -v
```

### Test Requirements

- Unit tests for all packages
- Integration tests for gateway client
- Config validation tests
- Error handling tests

## Pull Request Process

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

- Write clean, documented code
- Add tests for new functionality
- Update documentation if needed

### 3. Test Your Changes

```bash
make test
make build
make run-example
```

### 4. Commit Your Changes

Use clear, descriptive commit messages:

```bash
git commit -m "feat: add support for custom config validation"
git commit -m "fix: handle git metadata when not in a repository"
git commit -m "docs: update README with new examples"
```

### 5. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

### 6. PR Requirements

- [ ] Tests pass
- [ ] Code follows Go conventions
- [ ] Documentation updated (if applicable)
- [ ] Changes align with CLI philosophy
- [ ] Clear description of changes
- [ ] No breaking changes (or clearly documented)

## What to Contribute

### Good Contributions ✅

- Bug fixes
- Error message improvements
- Documentation updates
- CI/CD workflow examples
- Test coverage improvements
- Performance optimizations
- Support for new git providers (in config schema)
- Better error handling

### Contributions Needing Discussion ⚠️

- New CLI commands
- New flags or options
- Changes to config schema
- New dependencies
- Breaking changes

### Not Accepted ❌

- Interactive features (prompts, wizards)
- Local state/caching
- Service discovery/inference
- Schema validation logic
- Diffing capabilities
- Token management UI
- Complex business logic

**Why?** These belong in the gateway backend, not the CLI.

## Bug Reports

### Before Reporting

1. Search existing issues
2. Try with latest version
3. Check if it's a configuration issue

### Good Bug Reports Include

- uigraph-cli version (`uigraph --version`)
- Go version
- Operating system
- Complete error message
- Steps to reproduce
- Expected vs actual behavior
- Minimal config file (if relevant)

### Example Bug Report

```markdown
## Bug: CLI fails with valid YAML config

**Version:** uigraph v1.0.0
**OS:** Ubuntu 22.04
**Go:** 1.23

**Steps:**
1. Create .uigraph.yaml with valid content
2. Run `uigraph sync`

**Expected:** Service syncs successfully
**Actual:** Error: "failed to parse YAML"

**Config:**
```yaml
version: 1
project:
  name: test
...
```

**Error:**
```
Error: failed to parse YAML: line 5: ...
```
```

## Feature Requests

Before requesting features, consider:

1. **Does this belong in the CLI?** Most intelligence should be in the gateway
2. **Is it CI/CD friendly?** No interactive features
3. **Is it stateless?** No local caching or state

### Good Feature Requests

- Support for new API types
- Additional git metadata capture
- New CI/CD platform examples
- Better error messages
- Performance improvements

### Feature Requests to Reconsider

- Interactive configuration
- Local API schema validation
- Service discovery
- Automatic API detection
- Dashboard/UI features

## Code Review

All submissions require review. We look for:

- [ ] Code quality and clarity
- [ ] Test coverage
- [ ] Documentation
- [ ] Alignment with project philosophy
- [ ] No breaking changes (or properly documented)
- [ ] CI/CD compatibility

## Release Process

(For maintainers)

1. Update version in relevant files
2. Update CHANGELOG.md
3. Tag release: `git tag v1.x.x`
4. Push tag: `git push origin v1.x.x`
5. GitHub Actions builds and publishes releases

## Questions?

- 📧 Email: dev@uigraph.app
- 💬 Slack: [uigraph.slack.com](https://uigraph.slack.com)
- 🐛 Issues: [GitHub Issues](https://github.com/uigraph-app/uigraph-cli/issues)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to uigraph-cli! 🎉**
