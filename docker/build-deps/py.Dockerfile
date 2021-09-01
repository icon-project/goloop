ARG PYTHON_VERSION
ARG ALPINE_VERSION
FROM python:${PYTHON_VERSION}-alpine${ALPINE_VERSION}

# required by 'pip install coincurve cryptography'
RUN apk add --no-cache build-base libffi-dev openssl-dev

# setup python env
ADD requirements.txt /goloop/
WORKDIR /goloop
RUN python3 -m venv /goloop/venv && \
    source /goloop/venv/bin/activate && \
    pip install --upgrade pip && \
    pip install wheel && \
    pip install -r /goloop/requirements.txt

ARG GOLOOP_PYDEP_SHA
LABEL GOLOOP_PYDEP_SHA="$GOLOOP_PYDEP_SHA"

# entrypoint
RUN { \
        echo '#!/bin/sh'; \
        echo 'set -e'; \
        echo 'source /goloop/venv/bin/activate'; \
        echo 'exec "$@"'; \
    } > /entrypoint \
    && chmod +x /entrypoint
ENTRYPOINT ["/entrypoint"]

CMD ["/bin/sh"]
