# Contributing

Issues and pull requests are welcome.

## Setup

Go 1.26 or newer. No dependencies beyond the standard library.

## Checks

Run everything CI runs before opening a PR:

```bash
go vet ./...
gofmt -l .
go test -race ./...
golangci-lint run ./...
```

The live integration test is opt-in and never runs in CI:
`LATHER_LIVE=1 go test ./soap/ -run TestLive`.

## Rules

- Every README claim must map to a test. If you fix a generator bug, add the
  minimal WSDL shape that reproduces it as a regression test first.
- Collections are the contract: never flatten occurrence bounds. If a shape
  cannot be represented faithfully, degrade to XSDAny (raw XML preserved),
  never to a silent loss.
- Keep the public API small. A new entry point needs a real WSDL that needs it.
