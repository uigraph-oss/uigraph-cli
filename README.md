# UIGraph CLI

`uigraph-cli` syncs service metadata and API specifications from your repository to UiGraph.

It is designed for CI/CD workflows and works with a repository-level `.uigraph.yaml` file.

## Documentation

Full setup, configuration, and usage guides live in the official UiGraph docs:

- [UIGraph Documentation](https://docs.uigraph.app/)
- [UIGraph CLI Docs](https://docs.uigraph.app/uigraph-cli)

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
  -e UIGRAPH_GATEWAY_URL \
  -v "$(pwd):/workspace" \
  -w /workspace \
  uigraph-oss/uigraph-cli:latest sync
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
2. Set `UIGRAPH_TOKEN` and `UIGRAPH_GATEWAY_URL` in your CI environment for sync.
3. Validate the local artifacts, then sync:

```bash
uigraph-cli validate
uigraph-cli sync
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
uigraph-cli validate
uigraph-cli validate --config ./config/.uigraph.yaml
uigraph-cli sync
uigraph-cli sync --config ./config/.uigraph.yaml
uigraph-cli sync --dry-run
```

`validate` checks the config schema, referenced files, structured YAML/JSON artifacts, and timeline sources locally. It does not read `UIGRAPH_TOKEN` or `UIGRAPH_GATEWAY_URL`, contact the gateway, or connect to MLflow.

Record a release from a tag-triggered pipeline job. The version, notes and
commit range are resolved from git at the moment the tag is cut:

```bash
uigraph-cli release
uigraph-cli release --version v1.2.3 --notes-file ./RELEASE_NOTES.md
uigraph-cli release --dry-run
```

## What It Supports

- Syncing service metadata to UIGraph
- Syncing API specs such as OpenAPI, GraphQL, and gRPC
- Syncing cost category tags that decide which cloud resources roll up into a service's costs
- Syncing test packs and test cases, including reference screenshots
- Validating generated artifacts without credentials or network access
- Scanning the repo for timeline events — ADRs, postmortems, and CHANGELOG releases
- Recording a release on the timeline from a CI/CD tag job via `uigraph-cli release`
- Running cleanly in CI/CD pipelines
- Capturing git metadata during sync

## Development

```bash
make build
make test
```

## License

MIT. See [`LICENSE`](LICENSE).
