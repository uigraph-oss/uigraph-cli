# UiGraph CLI

`uigraph-cli` syncs service metadata and API specifications from your repository to UiGraph.

It is designed for CI/CD workflows and works with a repository-level `.uigraph.yaml` file.

## Documentation

Full setup, configuration, and usage guides live in the official UiGraph docs:

- [UiGraph Documentation](https://docs.uigraph.app/)
- [UiGraph CLI Docs](https://docs.uigraph.app/uigraph-cli)

This README stays intentionally brief. The hosted docs are the source of truth for configuration, supported inputs, and end-to-end workflows.

## Installation

### Install with Go

```bash
go install github.com/uigraph-oss/uigraph-cli@latest
```

### Run with Docker

No Go install required. Pull the published image from Docker Hub and sync from your repository:

```bash
docker pull uigraph-oss/uigraph-cli:latest

docker run --rm \
  -e UIGRAPH_TOKEN \
  -v "$(pwd):/workspace" \
  -w /workspace \
  uigraph/uigraph-cli:latest sync
```

Mount your repo at `/workspace` so the CLI can read `.uigraph.yaml` and API specs. Pass any sync flags after the image name (for example `sync --dry-run`).

### Build from source

```bash
git clone https://github.com/uigraph-oss/uigraph-cli.git
cd uigraph-cli
make build
```

## Quick Start

1. Add a `.uigraph.yaml` file to your repository.
2. Set `UIGRAPH_TOKEN` in your CI environment.
3. Run:

```bash
uigraph sync
```

Example:

```yaml
version: 1

project:
  name: my-product
  environment: production

service:
  name: Booking Service
  category: Backend
  description: Handles booking lifecycle and availability
  repository:
    provider: github
    url: https://github.com/company/booking-service

apis:
  - name: booking-service-openapi
    type: openapi
    path: ./openapi.yaml
```

## Common Usage

```bash
uigraph sync
uigraph sync --config ./config/.uigraph.yaml
uigraph sync --dry-run
```

## What It Supports

- Syncing service metadata to UiGraph
- Syncing API specs such as OpenAPI, GraphQL, and gRPC
- Running cleanly in CI/CD pipelines
- Capturing git metadata during sync

## Development

```bash
make build
make test
```

## License

MIT. See [`LICENSE`](LICENSE).
