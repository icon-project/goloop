ARG REPO_PY_DEPS=goloop/py-deps
ARG TAG_PY_DEPS=latest
FROM ${REPO_PY_DEPS}:${TAG_PY_DEPS}
LABEL MAINTAINER="t_arch@iconloop.com"

# install JRE
RUN apk add --no-cache openjdk11-jre-headless

ARG GOCHAIN_VERSION
LABEL GOCHAIN_VERSION="$GOCHAIN_VERSION"

# install python executor
ADD dist/pyexec-*.whl /goloop/
RUN /entrypoint python3 -m pip -q install /goloop/pyexec-*.whl && \
    rm -f /goloop/pyexec-*.whl

# install java executor
ADD dist/execman.zip /goloop/
RUN unzip -q /goloop/execman.zip -d /goloop/ && \
    rm -f /goloop/execman.zip
RUN unzip -q /goloop/execman/lib/exec.jar libclient.so -d /goloop/execman/lib && \
    sed -i 's/java.library.path=[a-zA-Z0-9\/\-]*/java.library.path=\/goloop\/execman\/lib/' \
    /goloop/execman/bin/execman

# install gochain and other stuff
ADD dist/gochain /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp

ADD run.sh /goloop/
CMD /goloop/run.sh
