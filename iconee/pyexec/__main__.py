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

from .ipc.proxy import ServiceManagerProxy, Log
from .pyexec import PyExecEngine

default_address = '/tmp/ee.socket'
default_log_config = {
    "log": {
        "logger": "pyexec",
        "level": "info",
        "outputType": "custom",
        "format": "%(filename)s:%(lineno)s %(message)s"
    }
}


def main():
    parser = argparse.ArgumentParser(prog='pyexec', description='Python Executor for ICON SCORE')
    parser.add_argument('-v', '--verbose', dest='verbose', action='store_true',
                        help='verbose mode (deprecated)')
    parser.add_argument('-d', '--debug', dest='log_level',
                        help='debugging log level')
    parser.add_argument('-s', '--socket', dest='socket',
                        help='a UNIX domain socket address for connection')
    parser.add_argument('-p', '--verify-package', dest='verify_package', action='store_true',
                        help='enable SCORE package validator')
    parser.add_argument('-u', '--uuid', dest='uuid', required=True,
                        help='a UUID for this instance')
    args = parser.parse_args()

    if args.socket:
        server_address = args.socket
    else:
        server_address = default_address

    if args.log_level:
        log_level = args.log_level
    elif args.verbose:
        log_level = "debug"
    else:
        log_level = "info"
    default_log_config["log"]["level"] = Log.to_py_level(log_level)

    engine = PyExecEngine(ServiceManagerProxy(), args.verify_package)
    engine.init_logger(default_log_config)
    engine.connect(server_address, args.uuid)
    engine.process()
    engine.close()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("exit")
