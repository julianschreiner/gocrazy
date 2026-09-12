# API gateway

A small Go gateway with method and path routing. Each upstream pool currently
supports one target; load balancing is still to come.

You'll need Docker Compose and the Go version in `go.mod`. To run it locally:

```sh
docker compose up -d --wait
go run ./cmd/gateway -config config/cfg.yaml
```

Compose starts two dummy services: users on `127.0.0.1:8081` and orders on
`127.0.0.1:8082`. They return JSON showing what they received, so you can
check where a request went and whether the gateway passed it through correctly.

With the gateway running, try these in another terminal:

```sh
curl -i http://localhost:8080/users
curl -i http://localhost:8080/orders
curl -i -H 'X-Request-ID: demo' -d '{"name":"Ada"}' 'http://localhost:8080/users?status=201'
```

Add `?status=503` to make a backend return an error, or use another status code
to check how the gateway handles it.

The routes and upstream addresses are in `config/cfg.yaml`. The integration
tests use their own config in `config/integration.yaml`.

Run unit tests:

```sh
go test ./...
```

For integration tests:

```sh
sh scripts/test-integration.sh
```

The script starts the backends, waits for them to be ready, and runs the Go
tests. These check routing, forwarded requests, backend responses, and missing
routes. The tests run their own gateway on a temporary port, so you don't need
to stop one already running on 8080.

Integration tests don't run with `go test ./...`. If the backends are already
up, you can run them directly:

```sh
go test -tags=integration ./test/integration -count=1 -v
```

The script leaves the backends running. When you're done:

```sh
docker compose down
```
