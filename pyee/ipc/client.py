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
        self.unpacker = msgpack.Unpacker(self)

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
