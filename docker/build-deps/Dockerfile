ARG IMAGE_GO_DEPS
ARG IMAGE_ROCKSDB_DEPS
FROM ${IMAGE_ROCKSDB_DEPS} as rocksdb

ARG IMAGE_GO_DEPS
FROM ${IMAGE_GO_DEPS}

RUN apk add --update --no-cache zlib-dev bzip2-dev snappy-dev lz4-dev zstd-dev libtbb-dev gflags-dev
COPY --from=rocksdb /work/rocksdb/lib /usr/lib/
COPY --from=rocksdb /work/rocksdb/include /usr/include/

ARG GOLOOP_BUILDDEP_SHA
LABEL GOLOOP_BUILDDEP_SHA="$GOLOOP_BUILDDEP_SHA"