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

# installing gochain and other stuff
ADD dist/gochain /goloop/bin/
ENV PATH $PATH:/goloop/bin
WORKDIR /goloop

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp

# add testsuite configuration
#ADD dist/testsuite.tar.gz /goloop/
#ENV GOCHAIN_CONFIG=/goloop/testsuite/config.json
#ENV GOCHAIN_DATA=/goloop/testsuite/data

##TODO run.sh to entrypoint
#ADD entrypoint /
#ENTRYPOINT /entrypoint
#CMD server start
ADD run.sh /goloop/
CMD /goloop/run.sh
