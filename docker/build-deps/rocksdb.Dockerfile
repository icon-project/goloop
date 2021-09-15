ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

RUN apk add --update --no-cache build-base linux-headers git cmake bash perl
RUN apk add --update --no-cache zlib-dev bzip2-dev snappy-dev lz4-dev zstd-dev libtbb-dev gflags-dev

WORKDIR /work
RUN mkdir -p /work

ARG ROCKSDB_VERSION
LABEL ROCKSDB_VERSION="$ROCKSDB_VERSION"
RUN cd /work && \
    git clone https://github.com/facebook/rocksdb.git && \
    cd rocksdb && \
    git checkout ${ROCKSDB_VERSION} && \
    PORTABLE=1 make shared_lib

RUN cd /work/rocksdb && \
    mkdir lib && \
    cp -P librocksdb.so* lib/

ARG GOLOOP_ROCKSDBDEP_SHA
LABEL GOLOOP_ROCKSDBDEP_SHA="$GOLOOP_ROCKSDBDEP_SHA"

