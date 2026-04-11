# k8s-graceful-shutdown-prestop-sleep sample

Runnable sample code for the Zenn article
`articles/k8s-graceful-shutdown-prestop-sleep-2026-04-11.md`.

## Contents

- `server.go` — HTTP server with a readiness flag that flips on shutdown,
  and a `Shutdown(ctx)` method that drains in-flight requests.
- `server_test.go` — Tests that verify each claim made in the article:
  - `/readyz` flips from 200 to 503 when `Shutdown` is called.
  - In-flight requests complete with 200 even if `Shutdown` starts while
    the handler is still running.
  - New connections are refused after `Shutdown` returns.
- `helm-snippet.yaml` — Minimal Deployment fragment showing the
  `preStop` sleep hook and `terminationGracePeriodSeconds` settings.

## Running the tests

```bash
go test ./... -v
```

Expected output:

```
=== RUN   TestReadyzFlipsTo503OnShutdown
--- PASS: TestReadyzFlipsTo503OnShutdown
=== RUN   TestShutdownWaitsForInFlightRequest
--- PASS: TestShutdownWaitsForInFlightRequest
=== RUN   TestShutdownRejectsNewConnections
--- PASS: TestShutdownRejectsNewConnections
=== RUN   TestIsReadyReflectsShutdownState
--- PASS: TestIsReadyReflectsShutdownState
PASS
```

No third-party dependencies are required — everything uses the Go
standard library.
