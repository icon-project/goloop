ARG IMAGE_PY_DEPS
FROM ${IMAGE_PY_DEPS}
LABEL MAINTAINER="t_arch@iconloop.com"

# install JRE
RUN apk add --no-cache openjdk11-jre-headless

ARG LCIMPORT_VERSION
LABEL LCIMPORT_VERSION="$LCIMPORT_VERSION"

# install python executor
ADD dist/iconee-*.whl /goloop/
RUN /entrypoint python3 -m pip -q install /goloop/iconee-*.whl && \
    rm -f /goloop/iconee-*.whl

# install java executor
#ARG JAVAEE_VERSION
#ADD dist/execman-${JAVAEE_VERSION}.zip /goloop/
#RUN unzip -q /goloop/execman-${JAVAEE_VERSION}.zip -d /goloop/ && \
#    mv /goloop/execman-${JAVAEE_VERSION} /goloop/execman && \
#    rm -f /goloop/execman-*.zip

# install gochain and other stuff
ADD dist/lcimport /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# install genesis governance
ADD icon_governance.zip /goloop/

# container configuration
VOLUME [ "/goloop/data" ]
ENV LCIMPORT_DATA="/goloop/data"
ENV PYEE_VERIFY_PACKAGE="true"
CMD [ "/goloop/bin/lcimport", "executor", "run" ]