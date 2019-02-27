FROM python:3.7.2-alpine3.9
MAINTAINER t_arch@icon.foundation

# installing required package
#RUN apt-get update && apt-get -qq -y install python3 python3-pip
RUN apk add bash

# preparing working directory
RUN mkdir /work
WORKDIR /work

# installing python executor
ADD pyexec-*.whl ./
RUN python3 -m pip install pyexec-*.whl
RUN rm pyexec-*.whl

# installing gochain and other stuff
ADD gochain ./
ADD run.sh ./
ADD systemInfo.json ./

# container configuration
EXPOSE 9080/tcp
EXPOSE 8080/tcp
ENTRYPOINT ["/bin/bash", "-c"]
CMD ["./run.sh"]

