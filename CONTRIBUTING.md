# Contributing to Migr8

Thank you for your interest in contributing to Migr8! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please treat all contributors and users with respect and create a welcoming environment for everyone.

## How to Contribute

### Reporting Issues

Before creating a new issue, please:

1. Check the [existing issues](https://github.com/yourusername/migr8/issues) to avoid duplicates
2. Use the appropriate issue template
3. Provide detailed information about the problem
4. Include steps to reproduce the issue
5. Specify your environment (OS, database version, etc.)

### Suggesting Features

Feature requests are welcome! Please:

1. Check existing feature requests first
2. Use the feature request template
3. Describe the use case and expected behavior
4. Explain why this feature would be beneficial
5. Consider implementation complexity

### Code Contributions

#### Getting Started

1. **Fork the repository**
```bash
git clone https://github.com/yourusername/migr8.git
cd migr8
```

2. **Set up development environment**
```bash
# Install Go 1.21 or later
go version

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

3. **Set up test databases (optional)**
```bash
docker-compose up -d postgres mysql
```

4. **Run tests**
```bash
# Unit tests
go test ./...

# Integration tests (requires test databases)
go test -tags=integration ./...
```

#### Development Workflow

1. **Create a feature branch**
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number
```

2. **Make your changes**
   - Write clear, concise code
   - Follow Go conventions and best practices
   - Add tests for new functionality
   - Update documentation if needed

3. **Test your changes**
```bash
# Run all tests
go test ./...

# Run linter
golangci-lint run

# Test specific database (if applicable)
TEST_DB_DRIVER=postgres go test -tags=integration ./pkg/database
```

4. **Commit your changes**
```bash
git add .
git commit -m "feat: add new migration validation"
# or
git commit -m "fix: resolve connection timeout issue (#123)"
```

5. **Push and create PR**
```bash
git push origin feature/your-feature-name
```

## Coding Standards

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

### Code Organization

- Place public APIs in `pkg/` directories
- Keep internal code in `internal/` directories
- Group related functionality together
- Use interfaces for abstraction where appropriate

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}

// Bad: Ignore or return raw errors
if err != nil {
    return err
}
```

### Testing

- Write unit tests for all public functions
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error conditions
- Aim for >80% test coverage

```go
func TestMigrationLoad(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Migration
        wantErr bool
    }{
        {
            name:  "valid migration",
            input: "20231201120000_create_users.up.sql",
            want:  Migration{Filename: "20231201120000_create_users"},
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use examples in documentation when helpful
- Keep documentation up to date with code changes

```go
// LoadMigrations loads all migration files from the specified directory.
// It returns a MigrationSet containing all valid migrations sorted by timestamp.
// Invalid migration files are skipped and logged as warnings.
func LoadMigrations(directory string) (*MigrationSet, error) {
    // Implementation
}
```

### User Documentation

- Update README.md for new features
- Add examples for new functionality
- Update command help text
- Create or update relevant documentation files

## Database Support

When adding support for a new database:

1. **Implement the interface**
```go
// Add database-specific logic to pkg/database/connection.go
func (db *DB) CreateMigrationsTable(tableName string) error {
    switch db.Driver {
    case "newdb":
        // Add implementation
    }
}
```

2. **Add backup support**
```go
// Add backup logic to pkg/backup/backup.go
func (bm *BackupManager) createNewDBBackup(backupPath, timestamp string) (*BackupInfo, error) {
    // Implementation
}
```

3. **Add integration tests**
```go
func TestNewDBIntegration(t *testing.T) {
    // Test with actual database
}
```

4. **Update documentation**
   - Add to supported databases list
   - Update configuration examples
   - Add any database-specific notes

## Performance Guidelines

### Optimization Principles

- Measure before optimizing
- Profile memory and CPU usage
- Optimize for the common case
- Consider memory vs. speed tradeoffs

### Database Operations

- Use transactions for consistency
- Stream large datasets
- Implement connection pooling
- Use prepared statements

### File Operations

- Stream large files
- Use buffered I/O
- Clean up temporary files
- Handle file system errors gracefully

## Release Process

### Version Numbers

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Changelog

Update CHANGELOG.md with:
- New features
- Bug fixes
- Breaking changes
- Deprecations

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass locally
- [ ] Code is properly formatted (`gofmt`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Documentation is updated
- [ ] Changelog is updated (for significant changes)

### PR Description

Include in your PR description:
- What changes were made
- Why the changes were needed
- How to test the changes
- Any breaking changes
- Related issue numbers

### Review Process

1. Automated checks must pass
2. At least one maintainer review required
3. All conversations must be resolved
4. Maintainer will merge when approved

## Getting Help

- Join our [Discord server](https://discord.gg/migr8) for discussions
- Check the [documentation](docs/)
- Search [existing issues](https://github.com/yourusername/migr8/issues)
- Create a new issue if you can't find help

## Recognition

Contributors will be:
- Listed in the README.md contributors section
- Mentioned in release notes
- Given appropriate credit for their contributions

Thank you for contributing to Migr8! 🚀