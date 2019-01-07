# Copyright 2018 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import sys
from .pyexec import PyExecEngine
from .ipc.proxy import ServiceManagerProxy

default_address = '/tmp/pyee_uds_socket'


def main():
    parser = argparse.ArgumentParser(prog='pyexec', description='Python Executor for ICON SCORE')
    parser.add_argument('-s', '--socket', dest='socket',
                        help='an UNIX domain socket address for the server')
    args = parser.parse_args()
    if args.socket:
        server_address = args.socket
    else:
        server_address = default_address

    engine = PyExecEngine(ServiceManagerProxy())
    engine.connect(server_address)
    engine.process()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("exit")
