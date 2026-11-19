# Rule: Go Standards & CLI Development

## Purpose
Enforce idiomatic Go 1.24+ coding practices, race condition immunity, directory traversal defense, and hermetic binary builds for CLI tooling.

## Scope
Applies to all code in `cmd/network/`, `cmd/plugin-manager/`, `internal/networkcli/`, and `internal/pluginmanager/`.

## MUST
- **MUST** format all Go code with `gofmt` and pass `go vet ./...` without warnings.
- **MUST** run and pass the Go race detector on all unit tests:
  ```bash
  go test -v -race ./...
  ```
- **MUST** sanitize all file paths using `filepath.Clean` and reject path traversal / Zip Slip attempts (`strings.HasPrefix(clean, "..")`).
- **MUST** handle all errors explicitly (`if err != nil { return ... }`).
- **MUST** build production binaries with `-trimpath` and stripped symbols:
  ```bash
  go build -trimpath -ldflags="-s -w" -o bin/network ./cmd/network
  go build -trimpath -ldflags="-s -w" -o bin/plugin-manager ./cmd/plugin-manager
  ```
- **MUST** acquire operational locks (`.network-operation.lock`) during destructive or stateful CLI operations (`restore`, `update`, `install`).

## MUST NOT
- **MUST NOT** ignore returned errors using blank identifiers (`_ = file.Close()`) without deliberate justification.
- **MUST NOT** commit code that introduces data races or unsafe pointer manipulations.
- **MUST NOT** use unescaped shell concatenation or raw `os.Exec` calls with untrusted user input.
- **MUST NOT** produce local debug binaries without `-trimpath`.

## Verification
Execute the test suite with race detection and compile production binaries:
```bash
docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...
go build -trimpath -ldflags="-s -w" -o bin/network ./cmd/network
```
Tests must pass with 0 data races and binaries must build cleanly.
