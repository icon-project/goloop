FROM alpine:3.10

RUN apk add --no-cache openjdk11-jdk build-base
ENV JAVA_HOME /usr/lib/jvm/default-jvm
RUN ln -s ${JAVA_HOME}/bin/javac /usr/bin/javac

ARG GOLOOP_JADEP_SHA
LABEL GOLOOP_JADEP_SHA="$GOLOOP_JADEP_SHA"
WORKDIR /goloop

CMD ["/bin/sh"]
