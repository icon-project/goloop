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
from iconcommons import IconConfig, Logger

from .pyexec import PyExecEngine
from .ipc.proxy import ServiceManagerProxy

default_address = '/tmp/pyee_uds_socket'
default_log_config = {
    "log": {
        "logger": "pyexec",
        "level": "info",
        "colorLog": True,
        "outputType": "console",
    }
}


def init_logger():
    conf = IconConfig("", default_log_config)
    Logger.load_config(conf)


def main():
    parser = argparse.ArgumentParser(prog='pyexec', description='Python Executor for ICON SCORE')
    parser.add_argument('-v', '--verbose', dest='verbose', action='store_true',
                        help='verbose mode')
    parser.add_argument('-s', '--socket', dest='socket',
                        help='an UNIX domain socket address for the server')
    args = parser.parse_args()

    if args.socket:
        server_address = args.socket
    else:
        server_address = default_address

    if args.verbose:
        init_logger()

    engine = PyExecEngine(ServiceManagerProxy())
    engine.connect(server_address)
    engine.process()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("exit")
