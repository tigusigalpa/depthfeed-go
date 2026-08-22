# Contributing to depthfeed-go

Thank you for your interest in improving `depthfeed-go`. This document covers development setup, testing, and contribution expectations.

## Development setup

```bash
git clone https://github.com/tigusigalpa/depthfeed-go.git
cd depthfeed-go
go mod download
```

Requires Go 1.22+.

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run a specific package
go test ./stream/...

# Check formatting
gofmt -l .

# Run go vet
go vet ./...
```

Tests use a mock HTTP server (`internal/testutil`) and the fake API key `df_test_example_not_a_real_key`. **No real API calls are made.** Do not add tests that call the real API.

### Integration tests

Integration tests (if added in the future) must be skipped by default and only run when `DEPTHFEED_INTEGRATION=1` is set. They must never use a real API key from source control — read from the environment.

## Code style

- Run `gofmt -w .` before committing.
- Run `go vet ./...` and fix all warnings.
- Follow idiomatic Go conventions.
- Do not add or remove comments unless asked.
- Keep functions focused; prefer composition over inheritance.

## Adding a new endpoint

1. Add the route to the endpoint coverage table in `services_test.go`.
2. Update `TestEndpointCount` with the new count.
3. Add typed request/response structs in `types.go` (or the relevant service file).
4. Implement the service method.
5. Add a test case using the mock server.
6. Update the README API coverage table.
7. Update the CHANGELOG under `Unreleased`.

## Security

- **Never commit real API keys.** Use `df_test_example_not_a_real_key` in tests.
- Never log API keys or signing secrets.
- Webhook verification must use raw request bytes and constant-time comparison.
- Production defaults must use HTTPS.
- Do not add keys to URLs or query parameters (except the documented browser-only WebSocket query-string auth mode).

## Pull requests

1. Fork the repository and create a feature branch.
2. Run `go test ./...`, `go vet ./...`, and `gofmt -l .` — all must pass.
3. Write a clear commit message explaining the "why" not just the "what".
4. Reference any related issues.

## Release process

1. Update `CHANGELOG.md` with the new version and date.
2. Tag the release: `git tag v1.0.0`.
3. The CI workflow validates the tag.

Do not tag or publish without maintainer approval.
