ARG ALPINE_VERSION
ARG IMAGE_PY_DEPS
ARG IMAGE_ROCKSDB_DEPS
ARG IMAGE_PY_DEPS
ARG BASE
FROM ${IMAGE_PY_DEPS} as base-py

FROM alpine:${ALPINE_VERSION} as base-java
RUN apk add --no-cache openjdk11-jre-headless

FROM base-py as base-all
RUN apk add --no-cache openjdk11-jre-headless

FROM ${IMAGE_ROCKSDB_DEPS} as rocksdb

ARG BASE
FROM ${BASE}
ARG ALPINE_UPDATES
LABEL MAINTAINER="t_arch@iconloop.com"
RUN apk add --update --no-cache zlib bzip2-dev snappy lz4-dev zstd-dev libtbb gflags libc6-compat ${ALPINE_UPDATES}
COPY --from=rocksdb /work/rocksdb/lib /usr/lib/

ARG GOLOOP_BASE_SHA
LABEL GOLOOP_BASE_SHA="$GOLOOP_BASE_SHA"
