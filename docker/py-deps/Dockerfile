ARG PYTHON_VERSION=3.7.5
FROM python:${PYTHON_VERSION}-alpine
RUN apk add git

#required by 'pip install secp256k1, cryptography'
RUN apk add --no-cache pkgconfig libffi-dev build-base autoconf automake libtool libressl-dev

ARG GOLOOP_PYREQ_SHA
LABEL GOLOOP_PYREQ_SHA="$GOLOOP_PYREQ_SHA"
ADD requirements.txt /goloop/
WORKDIR /goloop

RUN python3 -m venv /goloop/venv && \
    source /goloop/venv/bin/activate && \
    pip install -r /goloop/requirements.txt && \
    pip install wheel

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
