# buildkit-state

Restore and save buildkit state to Github cache during image build.  
Based on https://github.com/dashevo/gh-action-cache-buildkit-state.

## Goal

The [Buildkit cache (e.g. `--mount=type=cache,target=/some/path`)](https://docs.docker.com/engine/reference/builder/#run---mounttypecache) is different from the layer cache and [it can make image build very fast](https://vsupalov.com/buildkit-cache-mount-dockerfile/). But unlike layer cache, [docker/build-push-action](https://github.com/docker/build-push-action)'s `cache-to` & `cache-from` does not includes buildkit cache.  
This action can restore/save buildkit state on workflow to speed up CI/CD.

## Usage

### Example Dockerfile

```dockerfile
FROM python:3.11

RUN --mount=type=cache,target=/root/.cache/pip \
    pip install pandas uvicorn[standard] fastapi
```

### Caching every commit

```yaml
name: ci

on:
  pull_request:
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Set up Docker Buildx
        id: buildx
        uses: docker/setup-buildx-action@v2

      - name: Restore buildkit state
        uses: isac322/buildkit-state@v1
        with:
          buildx-name: ${{ steps.buildx.outputs.name }}

      - name: Build
        uses: docker/build-push-action@v3
        with:
          push: false
```

### Caching by version

If your state does not change within same version, you can specify version to cache key to reduce step execution time.
The buildkit state still be restored but it does not save back to Gtihub cache (because same state already exists on Github cache).

```yaml
name: ci

on:
  pull_request:
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Get app version
        id: version
        run: |
          echo "version=some-app-version" >> $GITHUB_OUTPUT

      - name: Set up Docker Buildx
        id: buildx
        uses: docker/setup-buildx-action@v2

      - name: Restore buildkit state
        uses: isac322/buildkit-state@v1
        with:
          buildx-name: ${{ steps.buildx.outputs.name }}
          cache-key: ${{ runner.os }}-buildkit_state-${{ steps.version.outputs.version }}
          cache-restore-keys: |
            ${{ runner.os }}-buildkit_state-
          target-types: |
            exec.cachemount
            frontend

      - name: Build
        uses: docker/build-push-action@v3
        with:
          push: false
```

## Customize

### Inputs

- `buildx-name`: Name of buildx. Required if `buildx_container_name` is not given.
  Fill as `name` output of `docker/setup-buildx-action` actions.
- `buildx-container-name`: Name of buildx container. Required if `buildx_name` is not given.

- `cache-key`: (optional) Github cache key. (default: `${{ runner.os }}-buildkit_state-${{ github.sha }}`)
- `cache-restore-keys`: (optional) Github cache restore key. (default: `${{ runner.os }}-buildkit_state-`)
- `target-types`: (optional) List of `regular` | `source.local` | `exec.cachemount` | `frontend` | `internal` (default: `exec.cachemount` and `frontend`). Refer to https://github.com/docker/cli/issues/2325#issuecomment-733975408
