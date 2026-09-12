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

To watch some traffic move through the gateway:

```sh
sh scripts/traffic-demo.sh
```

This starts the Compose backends and a demo gateway on a free local port using
`config/cfg.yaml`. It uses the same gateway code as the normal command, so you
can leave your gateway on 8080 running. The terminal shows traffic per service,
in-flight counts, errors, and recent requests with client, gateway, and backend
timings. Gateway time includes waiting for the backend; it isn't proxy overhead.
The backend name comes from the response. In-flight counts show which route
each request was sent to, before that response arrives.

Requests go to users and orders, with delays between 80 and 699 ms. Every tenth
request asks for a 503 so errors are visible too. The script restarts the two
test backends to pick up changes to their code. Ctrl+C stops the demo gateway
and traffic after pending requests finish, leaving the backends running.

For a busier, one-minute run:

```sh
sh scripts/traffic-demo.sh -rate 30 -concurrency 20 -duration 1m
```

Use `-plain` for periodic text snapshots instead of a live terminal display.
The rate is a target; requests are skipped if the concurrency limit is reached.
You can also try delays manually with `/users?delay_ms=500` (maximum 2000 ms).

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
