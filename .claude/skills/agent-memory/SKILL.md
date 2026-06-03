```markdown
# agent-memory Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches you the core development patterns and conventions used in the `agent-memory` Go codebase. You'll learn about file naming, import/export styles, commit message conventions, and how to write and run tests. This guide will help you contribute code that matches the project's established style and workflows.

## Coding Conventions

### File Naming
- Use **camelCase** for file names.
  - Example: `memoryStore.go`, `agentManager.go`

### Imports
- Use **relative imports** for internal packages.
  - Example:
    ```go
    import "./memory"
    ```

### Exports
- Use **named exports** for functions, types, and variables.
  - Example:
    ```go
    // Exported function
    func NewMemoryStore() *MemoryStore {
        // ...
    }
    ```

### Commit Messages
- Follow **conventional commit** format.
- Use the `fix` prefix for bug fixes.
- Keep commit messages concise (average 72 characters).
  - Example:
    ```
    fix: resolve memory leak in agent cache
    ```

## Workflows

### Fixing a Bug
**Trigger:** When you need to fix a bug in the codebase  
**Command:** `/fix-bug`

1. Identify the bug and create a new branch.
2. Make code changes following the coding conventions.
3. Write or update tests in `*_test.go` files.
4. Commit your changes using the `fix:` prefix.
   - Example: `fix: handle nil pointer in agent lookup`
5. Push your branch and open a pull request.

### Adding a New Feature
**Trigger:** When implementing a new feature  
**Command:** `/add-feature`

1. Create a new branch for your feature.
2. Add new files using camelCase naming.
3. Use relative imports for any internal dependencies.
4. Export new functions/types as needed.
5. Write corresponding tests in `*_test.go` files.
6. Commit with a clear message (e.g., `feat: add persistent memory store`).
7. Push and open a pull request.

### Running Tests
**Trigger:** To verify code correctness before merging  
**Command:** `/run-tests`

1. Ensure all test files follow the `*_test.go` pattern.
2. Run tests using the Go testing tool:
    ```sh
    go test ./...
    ```
3. Review test output and fix any failing tests.

## Testing Patterns

- Test files are named with the `*_test.go` suffix.
  - Example: `memoryStore_test.go`
- Tests are written using Go's standard testing package.
  - Example:
    ```go
    import "testing"

    func TestMemoryStore_Save(t *testing.T) {
        // test logic here
    }
    ```
- Place tests next to the code they test.

## Commands
| Command      | Purpose                                 |
|--------------|-----------------------------------------|
| /fix-bug     | Start the bug fix workflow              |
| /add-feature | Start the new feature workflow          |
| /run-tests   | Run all tests in the codebase           |
```
