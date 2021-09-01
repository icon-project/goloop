ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache openjdk11-jdk build-base
ENV JAVA_HOME /usr/lib/jvm/default-jvm

ARG GOLOOP_JADEP_SHA
LABEL GOLOOP_JADEP_SHA="$GOLOOP_JADEP_SHA"
COPY gradlew /goloop/
COPY gradle/ /goloop/gradle/
WORKDIR /goloop

RUN ./gradlew tasks \
    && find ~/.gradle/ -name "*.lock" -type f -delete

CMD ["/bin/sh"]
