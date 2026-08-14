# Hook Relay

Hook Relay is a small webhook delivery service. It keeps endpoint registrations, accepts events, signs request bodies, schedules retry attempts, moves exhausted deliveries to a dead-letter queue, and exposes read-only audit information.

Run the service with `go run ./cmd/hookrelay`. The HTTP API is available on `127.0.0.1:8080` by default; set `HOOK_RELAY_ADDR` to change the listen address.

The project has no external dependencies. Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...` to verify it.
