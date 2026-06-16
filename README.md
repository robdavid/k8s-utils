# k8s-utils

Go port of Deno-based Kubernetes secret management tools.

## Tools

### apply-secret
Reads secrets from the `pass` password store and writes them as Kubernetes secrets.

```
apply-secret [flags] [namespace[/name] ...]
```

- `--root` — Root folder in pass store (default: `k8s/<hostname>`)
- `--all`, `-a` — Apply all secrets under the root
- Positional args: `namespace` (all secrets in namespace) or `namespace/name` (specific secret)

### capture-secret
Reads Kubernetes secrets and stores them in the `pass` password store.

```
capture-secret [flags] [namespace[/name] ...]
```

- `--all-sealed`, `-a` — Save all secrets backed by a SealedSecret
- `--list-sealed`, `-l` — List all SealedSecrets
- Positional args: `namespace` (all secrets in namespace) or `namespace/name` (specific secret)

## Build

```
go build ./cmd/...
```

## Test

Tests use mocks and fakes — no `pass` binary or Kubernetes cluster required.

```
go test ./...
```
