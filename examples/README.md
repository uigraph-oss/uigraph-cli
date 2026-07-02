# Examples

This directory contains comprehensive examples for using `uigraph-cli` in various scenarios.

## Directory Structure

```
examples/
├── configs/              # Sample .uigraph.yaml configurations
├── ci-cd/               # CI/CD workflow examples
├── specs/               # Sample API specifications
│   ├── openapi/        # OpenAPI 3.0 examples
│   ├── graphql/        # GraphQL schema examples
│   └── grpc/           # Protocol Buffer examples
└── README.md           # This file
```

## Quick Start

### 1. Basic Usage

```bash
cd examples/configs
export UIGRAPH_TOKEN=your-token
uigraph sync --config backend-service.yaml
```

### 2. Dry Run

Preview what will be synced without actually sending data:

```bash
uigraph sync --config backend-service.yaml --dry-run
```

## Configuration Examples

### [`configs/backend-service.yaml`](configs/backend-service.yaml)
Complete backend service example with OpenAPI spec, integrations, and full metadata.

**Use case:** Typical microservice with REST API

### [`configs/frontend-service.yaml`](configs/frontend-service.yaml)
Frontend web application configuration.

**Use case:** React/Vue/Angular application with backend API

### [`configs/multi-api-service.yaml`](configs/multi-api-service.yaml)
Service with multiple API types (GraphQL + gRPC).

**Use case:** Modern service exposing multiple API protocols

### [`configs/staging-service.yaml`](configs/staging-service.yaml)
Staging environment configuration.

**Use case:** Non-production environment tracking

### [`configs/minimal-service.yaml`](configs/minimal-service.yaml)
Minimal configuration with only required fields.

**Use case:** Quick setup or simple services

## CI/CD Examples

### GitHub Actions

#### Basic Setup
See [`ci-cd/github-actions.yml`](ci-cd/github-actions.yml)

```yaml
- name: Sync to UiGraph
  env:
    UIGRAPH_TOKEN: ${{ secrets.UIGRAPH_TOKEN }}
  run: uigraph sync
```

#### Advanced (with PR dry-run)
See [`ci-cd/github-actions-advanced.yml`](ci-cd/github-actions-advanced.yml)

Automatically runs dry-run on pull requests and syncs on merge to main.

### GitLab CI

See [`ci-cd/gitlab-ci.yml`](ci-cd/gitlab-ci.yml)

```yaml
uigraph-sync:
  script:
    - curl -sSL https://cli.uigraph.app/install.sh | sh
    - uigraph sync
```

### Bitbucket Pipelines

See [`ci-cd/bitbucket-pipelines.yml`](ci-cd/bitbucket-pipelines.yml)

Includes configuration for branches and pull requests.

### CircleCI

See [`ci-cd/circleci.yml`](ci-cd/circleci.yml)

Uses CircleCI contexts for secure token management.

## API Specification Examples

### OpenAPI 3.0

- **[`specs/openapi/booking-service.yaml`](specs/openapi/booking-service.yaml)** - Full REST API example with CRUD operations
- **[`specs/openapi/web-app-api.yaml`](specs/openapi/web-app-api.yaml)** - Simplified web app API
- **[`specs/openapi/payment-api.yaml`](specs/openapi/payment-api.yaml)** - Payment processing API
- **[`specs/openapi/mobile-api.yaml`](specs/openapi/mobile-api.yaml)** - Mobile backend API

### GraphQL

- **[`specs/graphql/schema.graphql`](specs/graphql/schema.graphql)** - Analytics service GraphQL schema

### gRPC

- **[`specs/grpc/analytics.proto`](specs/grpc/analytics.proto)** - Analytics service Protocol Buffer definition

## Common Patterns

### Pattern 1: Single API Service

```yaml
version: 1
project:
  name: my-product
service:
  name: My Service
  category: Backend
  description: Service description
  repository:
    provider: github
    url: https://github.com/org/repo
apis:
  - name: my-api
    type: openapi
    path: ./openapi.yaml
```

### Pattern 2: Multi-API Service

```yaml
apis:
  - name: rest-api
    type: openapi
    path: ./specs/openapi.yaml
  - name: graphql-api
    type: graphql
    path: ./specs/schema.graphql
  - name: grpc-api
    type: grpc
    path: ./specs/service.proto
```

### Pattern 3: Service with Integrations

```yaml
service:
  # ... other fields
  integrations:
    jira:
      url: https://company.atlassian.net/browse/PROJECT
    slack:
      url: https://company.slack.com/archives/CHANNEL
```

### Pattern 4: Environment-Specific Config

```yaml
project:
  name: my-product
  environment: staging  # or production, development
```

## Testing Locally

### 1. Install CLI

```bash
# From repository root
make build
```

### 2. Set Token

```bash
export UIGRAPH_TOKEN=your-test-token
```

### 3. Run Examples

```bash
# Dry run (safe, no data sent)
./bin/uigraph sync --config examples/configs/backend-service.yaml --dry-run

# Actual sync (sends data to gateway)
./bin/uigraph sync --config examples/configs/backend-service.yaml
```

## Tips & Best Practices

### 1. Use Dry-Run in CI/CD

Always preview changes before syncing:

```yaml
# In your CI/CD
- name: Preview changes
  run: uigraph sync --dry-run

- name: Sync if approved
  run: uigraph sync
```

### 2. Validate Config Locally

Use dry-run to catch configuration errors early:

```bash
uigraph sync --config .uigraph.yaml --dry-run
```

### 3. Organize API Specs

Keep API specs in a dedicated directory:

```
project/
├── .uigraph.yaml
├── docs/
│   ├── openapi.yaml
│   ├── schema.graphql
│   └── service.proto
```

### 4. Use Relative Paths

Reference API specs with relative paths:

```yaml
apis:
  - name: my-api
    type: openapi
    path: ./docs/openapi.yaml  # Relative to .uigraph.yaml
```

### 5. Version Your Specs

Track API changes alongside code:

```bash
git add .uigraph.yaml docs/openapi.yaml
git commit -m "Update API spec"
```

## Troubleshooting

### Config Validation Errors

```bash
# Check config syntax
uigraph sync --config .uigraph.yaml --dry-run
```

### Missing API Spec File

```
Error: apis[0].path file does not exist: ./openapi.yaml
```

**Solution:** Verify the path is correct and relative to the config file.

### Authentication Errors

```
Error: UIGRAPH_TOKEN environment variable is required
```

**Solution:** Set the token in your environment or CI/CD secrets.

### Git Metadata Missing

```
Warning: Git metadata unavailable, continuing without it
```

**Solution:** Ensure you're running in a git repository, or ignore if not needed.

## Further Reading

- [Main README](../README.md) - CLI documentation
- [ARCHITECTURE](../ARCHITECTURE.md) - Design decisions
- [CONTRIBUTING](../CONTRIBUTING.md) - How to contribute

## Need Help?

- 📧 Email: support@uigraph.app
- 💬 Slack: [uigraph.slack.com](https://uigraph.slack.com)
- 🐛 Issues: [GitHub Issues](https://github.com/uigraph-app/uigraph-cli/issues)
