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

import socket
from typing import Any
from typing import Tuple
import msgpack


class Client:
    def __init__(self) -> None:
        self.conn = None
        self.req_sn: int = 0
        self.unpacker = None

    def _send(self, msg: int, data: Any):
        msg_to_send = [msg, data]
        msgpack.dump(msg_to_send, self)

    def _recv(self) -> Tuple[int, Any]:
        msg = self.unpacker.unpack()
        return msg[0], msg[1]

    def connect(self, address: str) -> None:
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.connect(address)
        self.conn = sock
        self.unpacker = msgpack.Unpacker(self,
                                         max_bin_len=10*1024*1024,
                                         max_buffer_size=2**31)

    def write(self, b: bytes) -> None:
        self.conn.sendall(b)

    def read(self, n=None) -> bytes:
        if n is None:
            n = 1024
        return self.conn.recv(n)

    def send(self, msg: int, data: Any):
        self._send(msg, data)

    def send_and_receive(self, msg: int, data: Any) -> Tuple[int, Any]:
        self._send(msg, data)
        return self._recv()

    def receive(self) -> Tuple[int, Any]:
        return self._recv()

    def close(self):
        return self.conn.close()
