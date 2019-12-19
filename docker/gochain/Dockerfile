ARG REPO_PY_DEPS=goloop/py-deps
ARG TAG_PY_DEPS=latest
FROM ${REPO_PY_DEPS}:${TAG_PY_DEPS}
#[deprecated 'MAINTAINER']
MAINTAINER t_arch@iconloop.com

ARG GOCHAIN_VERSION
LABEL GOCHAIN_VERSION="$GOCHAIN_VERSION"

# installing python executor
ADD dist/pyexec-*.whl /goloop/
RUN /entrypoint python3 -m pip install /goloop/pyexec-*.whl && \
    rm -f /goloop/pyexec-*.whl

# installing JRE & Java executor
RUN apk add --no-cache openjdk11-jre-headless
ADD dist/exectest.zip /goloop/
RUN unzip /goloop/exectest.zip -d /goloop/ && \
    rm -f /goloop/exectest.zip
RUN unzip /goloop/exectest/lib/exec.jar libclient.so -d /goloop/exectest/lib && \
    sed -i 's/java.library.path=[a-zA-Z0-9\/\-]*/java.library.path=\/goloop\/exectest\/lib/' \
    /goloop/exectest/bin/exectest

# installing gochain and other stuff
ADD dist/gochain /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp

ADD run.sh /goloop/
CMD /goloop/run.sh
