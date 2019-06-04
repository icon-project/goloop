ARG REPO_PY_DEPS=goloop/py-deps
ARG TAG_PY_DEPS=latest
FROM ${REPO_PY_DEPS}:${TAG_PY_DEPS}
#[deprecated 'MAINTAINER']
MAINTAINER t_arch@icon.foundation

ARG GOLOOP_VERSION
LABEL GOLOOP_VERSION="$GOLOOP_VERSION"

# installing python executor
ADD dist/pyee /goloop/pyee
RUN /entrypoint python3 -m pip install /goloop/pyee/pyexec-*.whl && \
    rm -rf /goloop/pyee

# installing gochain and other stuff
ADD dist/bin/* /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp
VOLUME ["/goloop/data"]

# goloop entrypoint
ENV GOLOOP_DATA_ROOT=/goloop/data
ENV GOLOOP_NODE_DIR=/goloop/data
ENV GOLOOP_CONFIG=/goloop/config.json
ENV GOLOOP_KEY_STORE=/goloop/keystore.json
ENV GOLOOP_P2P_LISTEN=":8080"
ENV GOLOOP_RPC_ADDR=":9080"

# entrypoint
RUN { \
  		echo '#!/bin/sh'; \
  		echo 'set -e'; \
  		echo 'if [ "$GOLOOP_CONFIG" != "" ] && [ ! -f "$GOLOOP_CONFIG" ]; then'; \
  		echo '  ENV_UNSET="GOLOOP_CONFIG=''"'; \
  		echo '  if [ "$GOLOOP_KEY_STORE" != "" ] && [ ! -f "$GOLOOP_KEY_STORE" ]; then'; \
  		echo '    GOLOOP_SAVE_KEY_STORE="--save_key_store=$GOLOOP_KEY_STORE"'; \
  		echo '    ENV_UNSET="$ENV_UNSET GOLOOP_KEY_STORE=''"'; \
  		echo '    sh -c "$ENV_UNSET goloop server save $GOLOOP_CONFIG --save_key_store=$GOLOOP_KEY_STORE"'; \
  		echo '  else'; \
  		echo '    sh -c "$ENV_UNSET goloop server save $GOLOOP_CONFIG"'; \
  		echo '  fi'; \
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