ARG IMAGE_BASE
FROM ${IMAGE_BASE}
LABEL MAINTAINER="t_arch@iconloop.com"

ARG GOCHAIN_VERSION
LABEL GOCHAIN_VERSION="$GOCHAIN_VERSION"

# install python executor
ADD dist/pyexec-*.whl /goloop/
RUN /entrypoint python3 -m pip -q install /goloop/pyexec-*.whl && \
    rm -f /goloop/pyexec-*.whl

# install java executor
ARG JAVAEE_VERSION
ADD dist/execman-${JAVAEE_VERSION}.zip /goloop/
RUN unzip -q /goloop/execman-${JAVAEE_VERSION}.zip -d /goloop/ && \
    mv /goloop/execman-${JAVAEE_VERSION} /goloop/execman && \
    rm -f /goloop/execman-*.zip
ENV JAVAEE_BIN /goloop/execman/bin/execman

# install gochain and other stuff
ADD dist/gochain /goloop/bin/
ENV PATH $PATH:/goloop/bin

# container configuration
WORKDIR /goloop
EXPOSE 9080/tcp
EXPOSE 8080/tcp

ADD run.sh /goloop/
CMD /goloop/run.sh
