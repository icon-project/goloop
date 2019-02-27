FROM python:3.7.2-alpine3.9
MAINTAINER t_arch@icon.foundation

# extra packages
RUN apk add bash make libc-dev go

# python tools to build whl file.
RUN python3 -m pip install setuptools wheel
CMD ["/bin/bash"]
