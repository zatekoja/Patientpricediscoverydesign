# Docker Buildx Build Action

A reusable composite GitHub Action that encapsulates Docker BuildKit image builds with local layer caching and flexible output options.

## Features

- **BuildKit/Buildx Support**: Uses `docker/setup-buildx-action` for advanced build features
- **Local Layer Caching**: Persists build cache in `.github/.docker-cache/` across workflow runs per service
- **Multi-platform Builds**: Supports building for multiple architectures (e.g., `linux/amd64`, `linux/arm64`)
- **Flexible Output**: Load images locally (`--load`) or push to registry (`--push`)
- **Build Arguments & Labels**: Pass custom build args and OCI labels
- **Validation**: Checks Dockerfile and context paths before building

## Usage

### Basic Local Build

```yaml
- name: Build Service Image
  uses: ./.github/actions/docker-buildx
  with:
    context: Backend/Services
    dockerfile: Backend/Services/deployments/docker/api/my-service/Dockerfile
    image-tag: my-service:latest
    load: true
```

### Build with Arguments and Custom Cache

```yaml
- name: Build Service Image
  uses: ./.github/actions/docker-buildx
  with:
    context: Backend/Services
    dockerfile: Backend/Services/deployments/docker/api/my-service/Dockerfile
    image-tag: my-registry.io/my-service:v1.2.3
    build-args: |
      SERVICE_NAME=my-service
      ENVIRONMENT=production
      GO_VERSION=1.24
    cache-key: my-service-prod
    platform: linux/amd64,linux/arm64
    push: true
    load: false
```

### Build with Labels

```yaml
- name: Build Service Image
  uses: ./.github/actions/docker-buildx
  with:
    context: Backend
    dockerfile: Backend/Services/deployments/docker/ecs/worker/Dockerfile
    image-tag: worker:${{ github.sha }}
    labels: |
      org.opencontainers.image.created=${{ github.event.head_commit.timestamp }}
      org.opencontainers.image.revision=${{ github.sha }}
      org.opencontainers.image.source=${{ github.repositoryUrl }}
    load: true
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `context` | Build context path (e.g., `Backend` or `Backend/Services`) | ✅ Yes | - |
| `dockerfile` | Path to Dockerfile relative to repository root | ✅ Yes | - |
| `image-tag` | Image tag to apply (e.g., `service-name:latest` or full registry path) | ✅ Yes | - |
| `build-args` | Build arguments as multi-line string (e.g., `SERVICE_NAME=foo\nENVIRONMENT=dev`) | No | `""` |
| `platform` | Target platform(s) for multi-arch builds (e.g., `linux/amd64,linux/arm64`) | No | `linux/amd64` |
| `cache-key` | Cache key for buildx cache directory (defaults to sanitized `image-tag` if not provided) | No | Derived from `image-tag` |
| `push` | Push image to registry after build (requires registry login beforehand) | No | `false` |
| `load` | Load image into local Docker daemon (cannot be used with `push=true` for multi-platform) | No | `true` |
| `labels` | Image labels as multi-line string (e.g., `org.opencontainers.image.created=$DATE`) | No | `""` |
| `cache-mode` | Cache export mode (`min` or `max`) | No | `max` |

## Outputs

| Output | Description |
|--------|-------------|
| `image-tag` | The image tag that was built |
| `image-digest` | Image digest (sha256) from build metadata |
| `cache-hit` | Whether cache was used effectively |

## Cache Strategy

The action uses a **local directory cache** strategy for BuildKit layer caching:

- Cache is stored in `.github/.docker-cache/{cache-key}/`
- Default `cache-key` is derived from `image-tag` (sanitized for filesystem)
- Each service can have its own cache by providing a unique `cache-key`
- Cache is reused across workflow runs and services
- `cache-mode=max` exports all layers to maximize reuse

### Example Cache Keys

```yaml
# Per-service cache (recommended for matrix builds)
cache-key: ${{ matrix.service.service }}

# Per-environment cache
cache-key: ${{ inputs.environment }}-my-service

# Shared cache across services
cache-key: shared-backend-services
```

## Local-Only Mode

This action is designed for **local-only builds** by default:

- No automatic registry authentication
- Default `load: true` imports image into local Docker daemon
- If pushing to a registry is needed:
  1. Set `push: true` and `load: false`
  2. Authenticate to your registry **before** calling this action (e.g., via `aws-actions/amazon-ecr-login` or `docker/login-action`)

## Integration with LocalStack Workflow

The `_test-service-localstack.yml` workflow uses this action to pre-build service images before `docker compose up`:

1. **Build Step**: Calls this action to build the service image with `load: true`
2. **Compose Override**: Creates a temporary `docker-compose.override.yml` to reference the pre-built image tag
3. **Start Service**: Runs `docker compose up` without rebuilding

This separation ensures:
- Faster test cycles (no rebuild on each test run)
- Better cache utilization across services
- Clearer separation of build vs. runtime concerns

## Notes

- **Multi-platform Builds**: When building for multiple platforms with `push: false`, you must set `load: false` as Docker cannot load multi-platform images locally
- **Registry Login**: If `push: true`, ensure you authenticate to the registry in a prior step
- **Cache Growth**: The cache directory can grow over time; consider periodic cleanup in CI/CD pipelines
- **Validation**: The action validates that both `dockerfile` and `context` exist before attempting to build

## Examples from the Codebase

### Test Service Build (LocalStack)

```yaml
- name: Build Service Image
  uses: ./.github/actions/docker-buildx
  with:
    context: Backend/Services
    dockerfile: ${{ matrix.service.dockerfile }}
    image-tag: p2pbets-${{ matrix.service.service }}:local-test
    build-args: |
      SERVICE_NAME=${{ matrix.service.service }}
      ENVIRONMENT=test
    cache-key: ${{ matrix.service.service }}
    load: true
    push: false
```

### Production Build (ECR Push)

```yaml
- name: Login to Amazon ECR
  uses: aws-actions/amazon-ecr-login@v2

- name: Build and Push Service Image
  uses: ./.github/actions/docker-buildx
  with:
    context: Backend
    dockerfile: Backend/Services/deployments/docker/api/my-service/Dockerfile
    image-tag: ${{ vars.AWS_ACCOUNT_ID }}.dkr.ecr.us-west-2.amazonaws.com/p2pbets-image-prod:my-service-${{ github.sha }}
    context: Backend/Services
      SERVICE_NAME=my-service
      ENVIRONMENT=production
    cache-key: my-service-prod
    platform: linux/amd64
    push: true
    load: false
```

