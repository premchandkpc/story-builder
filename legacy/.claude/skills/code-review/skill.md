---
name: Code Review
description: Systematic review of API endpoints, service logic, domain types, error handling, and test coverage. Use whenever the user asks to review the project code, find bugs, audit API surface, check error handling patterns, verify service logic, or assess test coverage across handlers/services/domain.
---

# Code Review

Review the entire project systematically: API surface, service logic, domain model, error handling, and test coverage. Uses a layered approach — endpoints first, then services, then domain.

## Review Layers (run in order)

### Layer 1: API Surface Audit

For every registered route, verify:
- HTTP method + path is correct
- Handler function exists and is wired
- Request body is decoded with error handling (bad JSON returns 400)
- Path params are extracted with chi
- All success/error response codes are documented or consistent
- No handler returns unimplemented stubs for critical endpoints

Commands:
```bash
# Find all route registrations
rg "r\.(Get|Post|Put|Delete|Route)\(" internal/api/server.go

# Find all handler implementations
rg "func \(h \*Handlers\)" internal/api/*.go | sort

# Find stubs
rg "NotImplemented|EmptyArray" internal/api/server.go
```

### Layer 2: Handler Inspection

For each handler file, verify:
- Request validation (empty fields, bad JSON)
- Service call error handling
- Response consistency (same shape for success, same error format)
- No leaked internal errors in production responses

Check patterns:
```bash
# Handlers with missing validation
rg -l "json.NewDecoder\(r.Body\)" internal/api/*.go | xargs rg -L "if.*err != nil"

# Handlers that might leak internal errors
rg "writeError\(w.*err\.Error\(\)" internal/api/*.go | head -20
```

### Layer 3: Service Logic

For each service file, verify:
- Input validation before repository calls
- Repository error handling (wrap with context)
- Transactional correctness (no partial writes on failure)
- Nil guards for optional dependencies (event bus, progress hub)
- Context propagation through all calls

```bash
# List all service files
ls internal/service/*.go

# Check for missing context checks
rg "func \(s \*" internal/service/*.go | grep -v "_test.go"
```

### Layer 4: Domain Model

Verify:
- BSON/JSON tags are consistent on domain types
- No schema drift between domain types and repository usage
- Constants for status fields are used consistently
- Time fields use `time.Time` not pointers where appropriate
- Empty slices vs nil are handled in JSON serialization

```bash
# List all domain types
rg "^type " internal/domain/*.go | sort

# Check for missing json tags
rg "^type \w+ struct" -A 20 internal/domain/*.go | rg -B1 "^\t\w+ .*bson" | rg -v "bson.*json" || echo "all tagged"
```

### Layer 5: Test Coverage

Map test files to the code they cover:
```bash
# Test files
rg -l "_test\.go" --type go | sort

# Integration tests
rg "build integration" -l --type go
```

Flag as HIGH risk:
- Service files with zero test coverage
- Handlers with no validation tests
- Domain types missing from test assertions
- New endpoints not in integration tests

## Output Format

```markdown
## Code Review Report

### Summary
- **Routes reviewed**: X/Y (Z stubs)
- **Handlers validated**: X/Y
- **Services inspected**: X/Y
- **Test coverage**: X/Y files tested
- **Issues found**: X high, Y medium, Z low

### High Priority Issues
1. [issue] — file:line — description + fix

### Medium Priority Issues
...

### Low Priority / Observations
...

### Recommendations
- Top 3 things to fix
- Suggested test additions
- Architecture notes
```
