# tiny-loadbalancer

TinyLoadbalancer is an HTTP load balancer that supports several strategies for distributing incoming requests across a pool of backend servers. It uses a configuration file to define its behavior and the servers to which it should forward requests.

## Features

- Supports multiple load balancing strategies:
  - Round Robin
  - Weighted Round Robin
  - Random
  - IP hashing
- Health checks for backend servers.
- Retry requests on failure.
- Customizable configuration via `config.json`.

## Configuration

TinyLoadbalancer expects a path to a `config.json` file that defines its configuration. (See `config.json` for an example configuration.) [strategies](https://github.com/D-Andreev/tiny-loadbalancer/blob/main/internal/constants/constants.go#L5)

```json
{
  "port": 3333,
  "strategy": "weighted-round-robin",
  "healthCheckInterval": "30s",
  "retryRequests": true,
  "servers": [
    {
      "url": "http://localhost:8081",
      "weight": 5
    },
    {
      "url": "http://localhost:8082",
      "weight": 3
    },
    {
      "url": "http://localhost:8083",
      "weight": 2
    }
  ]
}
```

## Local Run
  * Start a server with `go run e2e_tests/server/server.go 8081`. You can start multiple servers on different ports.
  * Run the load balancer with `go run main.go config.json`.
  * `curl http://localhost:3333` to see the load balancer in action.

## Testing
  * `make test`
