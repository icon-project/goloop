FROM golang:1.12.9-alpine3.9
MAINTAINER t_arch@icon.foundation

# extra packages
RUN apk add bash make libc-dev git
WORKDIR /work
ADD go.mod /work
ADD go.sum /work
RUN go mod download

# python tools to build whl file.
CMD ["/bin/bash"]
