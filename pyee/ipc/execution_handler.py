from typing import Any, Tuple, List

from .client import Client
import os


# Convert python int to bytes of golang big.Int.
def int_to_bytes(v: int) -> bytes:
    n_bytes = ((v + (v < 0)).bit_length() + 8) // 8
    return v.to_bytes(n_bytes, byteorder="big", signed=True)


# Convert bytes of golang big.Int to python int.
def bytes_to_int(v: bytes) -> int:
    return int.from_bytes(v, "big", signed=True)


class SMProxy:
    M_VERSION = 0
    M_INVOKE = 1
    M_RESULT = 2
    M_GETVALUE = 3
    M_SETVALUE = 4
    M_CALL = 5
    M_EVENT = 6

    R_FAILURE = 0x758b
    R_SUCCESS = 0

    def __init__(self):
        self.__client = Client()
        self.__invoke = None
        self.__codec = None

    def connect(self, addr):
        self.__client.conn(addr)
        self.__client.send(SMProxy.M_VERSION, {
            "version": 1,
            "pid": os.getpid(),
        })

    def set_invoke_handler(self, invoke) -> None:
        self.__invoke = invoke

    def set_codec(self, codec) -> None:
        self.__codec = codec

    def decode(self, t: str, b: bytes) -> 'Any':
        if t == 'int':
            return bytes_to_int(b)
        elif t == 'str':
            return b.decode('utf-8')
        else:
            return self.__codec.decode(t, b)

    def encode(self, o: 'Any') -> bytes:
        if o is None:
            return bytes([])
        if isinstance(o, int):
            return int_to_bytes(o)
        elif isinstance(o, str):
            return o.encode('utf-8')
        else:
            return self.__codec.encode(o)

    def __handle_invoke(self, data):
        try:
            code = self.decode('str', data[0])
            _from = self.decode('Address', data[1])
            _to = self.decode('Address', data[2])
            value = self.decode('int', data[3])
            limit = self.decode('int', data[4])
            method = self.decode('str', data[5])
            params = data[6]

            status, step_used, result = self.__invoke(code, _from, _to, value, limit, method, params)

            self.__client.send(SMProxy.M_RESULT, [status, self.encode(step_used)])
        except BaseException:
            self.__client.send(SMProxy.M_RESULT, [SMProxy.R_UNKNOWN, self.encode(limit)])

    def loop(self):
        while True:
            msg, data = self.__client.receive()
            if msg == SMProxy.M_INVOKE:
                self.__handle_invoke(data)

    def call(self, to: 'Address', value: int, step_limit: int, method: str, params: 'Any') -> Tuple[int,int]:
        self.__client.send(SMProxy.M_CALL, [
            self.encode(to), self.encode(value), self.encode(step_limit),
            self.encode(method), self.encode(params),
        ])

        while True:
            msg, data = self.__client.receive()
            if msg == SMProxy.M_INVOKE:
                self.__handle_invoke(data)
            elif msg == SMProxy.M_RESULT:
                return data[0], self.decode('int', data[1])

    def get_value(self, key: bytes) -> bytes:
        msg, value = self.__client.send_and_receive(SMProxy.M_GETVALUE, key)
        if msg != SMProxy.M_GETVALUE:
            raise Exception("InvalidMsg(%d) exp=%d" % (msg, SMProxy.M_GETVALUE))
        return value

    def set_value(self, key: bytes, value: bytes):
        self.__client.send(SMProxy.M_SETVALUE, [key, value])

    def send_event(self, idxcnt: int, event: List[Any]):
        self.__client.send(SMProxy.M_EVENT, [
            idxcnt,
            map(lambda x: self.encode(x), event),
        ])
