# How uigraph-cli works

uigraph-cli is how repositories get into UIGraph. A repository describes itself in
`.uigraph.yaml` and files under `.uigraph/`. CI runs `uigraph-cli sync`, which pushes
everything through uigraph-gateway into uigraph-api.

## Commands

- `uigraph-cli sync [--config .uigraph.yaml] [--dry-run] [--verbose]` upserts the whole
  config. `--dry-run` prints the payloads without sending them.
- `uigraph-cli release [--version] [--notes | --notes-file]` records one release on the
  service timeline. It reads the tag at `HEAD`, so it needs full history. Notes come from
  `--notes`, then `--notes-file`, then the annotated tag body, then the matching changelog
  section, then up to 50 commit subjects since the previous tag.

## Target and credentials

`UIGRAPH_TOKEN` is a service account token (`uig_…`), sent as `X-API-Token`. The gateway is
chosen by exactly one of:

| Flag | Gateway |
|---|---|
| `--enterprise` | `https://console.uigraph.app/gateway` (UIGraph Cloud) |
| `--enterprise=DEV` | `https://console.dev.uigraph.app/gateway` |
| `--api-url` or `UIGRAPH_GATEWAY_URL` | A self-hosted gateway |

## Order of a sync

Service, then databases and saved queries, API groups, dependencies and cost tags,
diagrams, test packs, docs and videos, timeline, maps, and MLflow projects. Every call is an
idempotent upsert keyed by name, so a sync can be rerun safely. Without a `service` block
only maps and frames sync.

Files (docs, frames, test screenshots) use prepare and complete. The gateway answers `skip`
when the hash matches, or hands back a presigned URL the CLI uploads to directly. Videos
are listed under `videos:` and are turned into docs before they sync.

## Where it runs

- In CI from the `uigraph/uigraph-cli` Docker image, which `release.yml` publishes to
  Docker Hub.
- In repositories onboarded through the UIGraph GitHub App: the workflows that uigraph-api
  commits run `go install github.com/uigraph-oss/uigraph-cli@main` and then sync.
- Locally with `go install`.
