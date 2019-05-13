ARG PYTHON_VERSION=3.7.3
FROM python:${PYTHON_VERSION}-alpine
RUN apk add git

ARG GOLOOP_PYREQ_SHA
LABEL GOLOOP_PYREQ_SHA="$GOLOOP_PYREQ_SHA"
ADD requirements.txt /goloop/
WORKDIR /goloop

RUN python3 -m venv /goloop/venv && \
    source /goloop/venv/bin/activate && \
    pip install -r /goloop/requirements.txt && \
    pip install wheel
COPY entrypoint /
ENTRYPOINT ["/entrypoint"]
RUN chmod +x /entrypoint

CMD ["/bin/sh"]
