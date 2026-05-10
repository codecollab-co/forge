# forge-cli

`forge` — CLI for Forge.

Status: scaffold only. Built TDD-first, one command at a time. See ADR-0010
(forge CLI scope) once it lands.

## Usage (so far)

```bash
forge --version
```

## Local build

```bash
go build -o /tmp/forge ./cmd/forge
/tmp/forge --version
```

## Tests

```bash
go test ./...
```

Tests are integration-style: black-box calls to `cli.Run(args, stdout, stderr)`
that capture output and assert on what users actually see.
