ARG REPO_PY_DEPS=goloop/py-deps
ARG TAG_PY_DEPS=latest
FROM ${REPO_PY_DEPS}:${TAG_PY_DEPS}
#[deprecated 'MAINTAINER']
MAINTAINER t_arch@icon.foundation

ARG GOLOOP_VERSION
LABEL GOLOOP_VERSION="$GOLOOP_VERSION"

# installing python executor
ADD dist/pyexec-*.whl /goloop/
RUN /entrypoint python3 -m pip install /goloop/pyexec-*.whl && \
    rm -f /goloop/pyexec-*.whl

# installing gochain and other stuff
ADD dist/goloop /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp

ADD entrypoint /
RUN chmod +x /entrypoint
ENTRYPOINT ["/entrypoint"]

ENV GOLOOP_NODE_DIR=/goloop/data
ENV GOLOOP_CONFIG=/goloop/config.json
ENV GOLOOP_KEY_STORE=/goloop/keystore.json

CMD goloop server start
