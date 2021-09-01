ARG IMAGE_BASE
FROM ${IMAGE_BASE}
LABEL MAINTAINER="t_arch@iconloop.com"

ARG GOLOOP_VERSION
LABEL GOLOOP_VERSION="$GOLOOP_VERSION"

# install python executor
ADD dist/pyee /goloop/pyee
RUN /entrypoint python3 -m pip -q install /goloop/pyee/pyexec-*.whl && \
    rm -rf /goloop/pyee

# install java executor
ARG JAVAEE_VERSION
ADD dist/execman-${JAVAEE_VERSION}.zip /goloop/
RUN unzip -q /goloop/execman-${JAVAEE_VERSION}.zip -d /goloop/ && \
    mv /goloop/execman-${JAVAEE_VERSION} /goloop/execman && \
    rm -f /goloop/execman-*.zip
ENV JAVAEE_BIN /goloop/execman/bin/execman

# install goloop and other stuff
ADD dist/bin/* /goloop/bin/
ENV PATH $PATH:/goloop/bin

# container configuration
WORKDIR /goloop
EXPOSE 9080/tcp
EXPOSE 8080/tcp
VOLUME ["/goloop/data"]

# goloop entrypoint
ENV GOLOOP_DATA_ROOT=/goloop/data
ENV GOLOOP_NODE_DIR=/goloop/data
ENV GOLOOP_CONFIG=/goloop/config/server.json
ENV GOLOOP_KEY_STORE=/goloop/config/keystore.json
ENV GOLOOP_KEY_SECRET=/goloop/config/keysecret
ENV GOLOOP_P2P_LISTEN=":8080"
ENV GOLOOP_RPC_ADDR=":9080"
ENV GOLOOP_ENGINES="python,java"

# entrypoint
RUN { \
        echo '#!/bin/sh'; \
        echo 'set -e'; \
        echo 'if [ "$GOLOOP_CONFIG" != "" ] && [ ! -f "$GOLOOP_CONFIG" ]; then'; \
        echo '  UNSET="GOLOOP_CONFIG"' ; \
        echo '  CMD="goloop server save $GOLOOP_CONFIG"'; \
        echo '  if [ "$GOLOOP_KEY_SECRET" != "" ] && [ ! -f "$GOLOOP_KEY_SECRET" ]; then'; \
        echo '    mkdir -p $(dirname $GOLOOP_KEY_SECRET)'; \
        echo '    echo -n $(date|md5sum|head -c16) > $GOLOOP_KEY_SECRET' ; \
        echo '  fi'; \
        echo '  if [ "$GOLOOP_KEY_STORE" != "" ] && [ ! -f "$GOLOOP_KEY_STORE" ]; then'; \
        echo '    UNSET="$UNSET GOLOOP_KEY_STORE"' ; \
        echo '    CMD="$CMD --save_key_store=$GOLOOP_KEY_STORE"' ; \
        echo '  fi'; \
        echo '  sh -c "unset $UNSET ; $CMD"' ; \
        echo 'fi'; \
        echo ; \
        echo 'source /goloop/venv/bin/activate'; \
        echo 'if [ "${GOLOOP_LOGFILE}" != "" ]; then'; \
        echo '  GOLOOP_LOGDIR=$(dirname ${GOLOOP_LOGFILE})'; \
        echo '  if [ ! -d "${GOLOOP_LOGDIR}" ]; then'; \
        echo '     mkdir -p ${GOLOOP_LOGDIR}'; \
        echo '  fi'; \
        echo '  exec "$@ 2>&1 | tee -a ${GOLOOP_LOGFILE}"'; \
        echo 'else'; \
        echo '  exec "$@"'; \
        echo 'fi'; \
    } > /entrypoint \
    && chmod +x /entrypoint
ENTRYPOINT ["/entrypoint"]

CMD goloop server start
