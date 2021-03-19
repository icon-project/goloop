ARG IMAGE_PY_DEPS
FROM ${IMAGE_PY_DEPS}
LABEL MAINTAINER="t_icondev@iconloop.com"

# install JRE
RUN apk add --no-cache openjdk11-jre-headless

ARG GOCHAIN_ICON_VERSION
LABEL GOCHAIN_ICON_VERSION="$GOCHAIN_ICON_VERSION"

# install python executor
ADD dist/iconee-*.whl /goloop/
RUN /entrypoint python3 -m pip -q install /goloop/iconee-*.whl && \
    rm -f /goloop/iconee-*.whl

# install java executor
ARG JAVAEE_VERSION
ADD dist/execman-${JAVAEE_VERSION}.zip /goloop/
RUN unzip -q /goloop/execman-${JAVAEE_VERSION}.zip -d /goloop/ && \
    mv /goloop/execman-${JAVAEE_VERSION} /goloop/execman && \
    rm -f /goloop/execman-*.zip

# install gochain and other stuff
ADD dist/gochain /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp

ADD run.sh /goloop/
CMD /goloop/run.sh
