---
name: code-review
description: Review a diff or pull request for bugs, security issues and maintainability problems. Use when asked to review code, check for bugs, or audit changes.
---

# Code Review

Review the change in this order and report findings grouped by severity. Only report what you can point to in the diff; do not speculate.

## 1. Correctness

- Logic errors: off-by-one, nil / null handling, missed edge cases
- Error handling: swallowed errors, missing early returns, wrong error messages
- Concurrency: shared state touched without synchronization
- Resource leaks: unclosed files, connections, goroutines that never exit

## 2. Security

- Injection: shell, SQL, path traversal, template
- Secrets: credentials or tokens hardcoded or logged
- Unchecked input reaching the file system or the shell

## 3. Maintainability

- Naming that hides intent, functions doing several unrelated things
- Duplicated logic that already exists elsewhere in the codebase
- Missing or misleading tests for the changed behaviour

## Output

```text
### Critical
- <file:line> — <what is wrong> — <how to fix>

### Warning
- ...

### Nit
- ...
```

Finish with one sentence saying whether the change is safe to merge.
