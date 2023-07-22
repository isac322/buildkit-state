#!/usr/bin/env bash

set -ex

# prepare base images
docker pull python:3.11
docker pull python:3.11-slim

# remove intermediate layers
docker image prune -f

# remove buildkit caches
docker buildx prune --all --force

# first build without cache
echo 'first build without buildkit cache'
time docker build -f Dockerfile-no-cache . --quiet

# remove intermediate layers (mimic Github Actions's environment)
docker image prune -f

## First build

# first build with cache
echo 'first build with buildkit cache'
time docker build -f Dockerfile-buildkit-cache . --quiet

# remove intermediate layers & uncached buildkit state (mimic Github Actions's environment & buildkit-state Action)
docker image prune -f
# delete all BuildKit caches excepts `exec.cachemount` and `frontend`
docker buildx prune --force --filter type=regular
docker buildx prune --force --filter type=source.local
docker buildx prune --force --filter type=internal

## Second build

# second build without cache
echo 'second build without buildkit cache'
time docker build -f Dockerfile-no-cache . --quiet

# remove intermediate layers (mimic Github Actions's environment)
docker image prune -f

# second build with cache
echo 'second build with buildkit cache'
time docker build -f Dockerfile-buildkit-cache . --quiet

# remove intermediate layers & uncached buildkit state (mimic Github Actions's environment & buildkit-state Action)
docker image prune -f
# delete all BuildKit caches excepts `exec.cachemount` and `frontend`
docker buildx prune --force --filter type=regular
docker buildx prune --force --filter type=source.local
docker buildx prune --force --filter type=internal

## Build after meaningless project changes

# add a dependency
sed -Ei 's/(\w*"pydantic==.+?".*)/\1\n"arrow==1.2.3",/' pyproject.toml

# build without cache
echo 'build without buildkit cache after adding a new dependency'
time docker build -f Dockerfile-no-cache . --quiet

# remove intermediate layers (mimic Github Actions's environment)
docker image prune -f

# build with cache
echo 'build with buildkit cache after adding a new dependency'
time docker build -f Dockerfile-buildkit-cache . --quiet
