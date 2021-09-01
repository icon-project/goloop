ARG IMAGE_BASE
FROM ${IMAGE_BASE}
LABEL MAINTAINER="t_arch@iconloop.com"

ARG LCIMPORT_VERSION
LABEL LCIMPORT_VERSION="$LCIMPORT_VERSION"

# install python executor
ADD dist/pyee /goloop/pyee
RUN /entrypoint python3 -m pip -q install /goloop/pyee/iconee-*.whl && \
    rm -rf /goloop/pyee

# install java executor
#ARG JAVAEE_VERSION
#ADD dist/execman-${JAVAEE_VERSION}.zip /goloop/
#RUN unzip -q /goloop/execman-${JAVAEE_VERSION}.zip -d /goloop/ && \
#    mv /goloop/execman-${JAVAEE_VERSION} /goloop/execman && \
#    rm -f /goloop/execman-*.zip
#ENV JAVAEE_BIN /goloop/execman/bin/execman

# install lcimport and other stuff
ADD dist/bin/* /goloop/bin/
ENV PATH $PATH:/goloop/bin

# install genesis governance
ADD icon_governance.zip /goloop/

# container configuration
WORKDIR /goloop
VOLUME [ "/goloop/data" ]

ENV LCIMPORT_DATA="/goloop/data"

ENV PYEE_VERIFY_PACKAGE="true"
CMD [ "/goloop/bin/lcimport", "executor", "run" ]
