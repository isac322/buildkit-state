# syntax=docker/dockerfile:1.5

FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx
FROM --platform=$BUILDPLATFORM golang:1.20.5-alpine as builder
# copy xx scripts to your build stage
COPY --from=xx / /
# export TARGETPLATFORM (or other TARGET*)
ARG TARGETPLATFORM

WORKDIR /app

RUN --mount=type=bind,source=go.mod,target=/app/go.mod \
    --mount=type=bind,source=go.sum,target=/app/go.sum \
    --mount=type=bind,source=cmd,target=/app/cmd \
    --mount=type=bind,source=internal,target=/app/internal \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 xx-go build  \
      -trimpath \
      -ldflags '-s -w' \
      -o "/release/$(xx-info os)-$(xx-info arch)" \
      github.com/isac322/buildkit-state/probe/cmd/probe \
    && xx-verify --static "/release/$(xx-info os)-$(xx-info arch)"

FROM --platform=$BUILDPLATFORM alpine as compressor
ARG TARGETPLATFORM
WORKDIR /release
COPY --from=builder /release ./
RUN if [ "$TARGETPLATFORM" != 'windows/arm64' ]; then  \
      apk add upx \
      && upx --lzma --best *; \
    fi

FROM scratch
COPY --link --from=compressor /release ./
