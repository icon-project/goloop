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

import traceback
from abc import ABCMeta, abstractmethod
from typing import Any, Tuple, List, Union, Callable, Optional

from .client import Client

TAG = 'Proxy'

# Set this value to non-zero value for actual
MAX_SET_VALUE_HANDLERS = 5


# Convert python int to bytes of golang big.Int.
def int_to_bytes(v: int) -> bytes:
    n_bytes = ((v + (v < 0)).bit_length() + 8) // 8
    return v.to_bytes(n_bytes, byteorder="big", signed=True)


# Convert bytes of golang big.Int to python int.
def bytes_to_int(v: bytes) -> int:
    return int.from_bytes(v, "big", signed=True)


class Message(object):
    VERSION = 0
    INVOKE = 1
    RESULT = 2
    GETVALUE = 3
    SETVALUE = 4
    CALL = 5
    EVENT = 6
    GETINFO = 7
    GETBALANCE = 8
    GETAPI = 9
    LOG = 10
    CLOSE = 11
    SETFEEPCT = 15
    CONTAINS = 16


class InvokeFlag(object):
    READ_ONLY = 1
    TRACE = 2


class Log(object):
    PANIC = 0
    FATAL = 1
    ERROR = 2
    WARN = 3
    INFO = 4
    DEBUG = 5
    TRACE = 6

    __levels = ["panic", "fatal", "error", "warn", "info", "debug", "trace"]

    __py_levels = ["", "", "error", "warning", "", "info", "debug"]
    __to_py_levels = {
        "panic": "error",
        "fatal": "error",
        "error": "error",
        "warn": "warning",
        "info": "info",
        "debug": "info",
        "trace": "debug"
    }

    @classmethod
    def from_string(cls, name: str) -> int:
        idx = cls.__levels.index(name)
        if idx < 0:
            raise Exception(f"unknown log level name={name}")
        return idx

    @classmethod
    def to_string(cls, level: int) -> str:
        return cls.__levels[level]

    @classmethod
    def from_py_level(cls, name: str):
        idx = cls.__py_levels.index(name.lower())
        if idx < 0:
            raise Exception(f"unknown log level name={name}")
        return idx

    @classmethod
    def to_py_level(cls, level: str) -> str:
        return cls.__to_py_levels.get(level, "info")


class LogFlag(object):
    TRACE = 1


class Status(object):
    SUCCESS = 0
    SYSTEM_FAILURE = 1


class SetValueFlag(object):
    DELETE = 1
    OLDVALUE = 2


SetHandler = Callable[[bool, int, int], None]


class Info(object):
    BLOCK_TIMESTAMP = "B.timestamp"
    BLOCK_HEIGHT = "B.height"
    TX_HASH = "T.hash"
    TX_INDEX = "T.index"
    TX_FROM = "T.from"
    TX_TIMESTAMP = "T.timestamp"
    TX_NONCE = "T.nonce"
    REVISION = "Revision"
    STEP_COSTS = "StepCosts"
    CONTRACT_OWNER = "C.owner"


class Codec(metaclass=ABCMeta):
    @abstractmethod
    def encode(self, o: Any) -> Tuple[int, bytes]:
        pass

    @abstractmethod
    def decode(self, t: int, bs: bytes) -> Any:
        pass


class TypeTag(object):
    NIL = 0
    DICT = 1
    LIST = 2
    BYTES = 3
    STRING = 4
    BOOL = 5

    CUSTOM = 10
    ADDRESS = CUSTOM
    INT = CUSTOM + 1


class APIType(object):
    FUNCTION = 0
    FALLBACK = 1
    EVENT = 2


class APIFlag(object):
    READONLY = 1
    EXTERNAL = 2
    PAYABLE = 4
    ISOLATED = 8


class DataType(object):
    INTEGER = 1
    STRING = 2
    BYTES = 3
    BOOL = 4
    ADDRESS = 5
    LIST = 6
    DICT = 7
    STRUCT = 8


class MethodName(object):
    FALLBACK = ""


class APIInfo(object):
    def __init__(self, proxy: 'ServiceManagerProxy'):
        self.__values = []
        self.__proxy = proxy

    def __encode_inputs(self, inputs: List[Tuple[str, int, Any]], optional: int) -> list:
        mandatory = len(inputs) - optional
        new_inputs = []
        for i in range(len(inputs)):
            name, _type, default = inputs[i][:3]
            item = [name, _type]
            if i < mandatory:
                item.append(None)
            else:
                item.append(self.__proxy.encode(default))
            if len(inputs[i]) == 4:
                fields = inputs[i][3]
                item.append(fields)
            new_inputs.append(item)
        return new_inputs

    def add_function(self, name: str, flags: int, optional: int, inputs: List[Tuple[str, int, Any]],
                     outputs: List[int]):
        self.__values.append([
            APIType.FUNCTION,
            name,
            flags,
            len(inputs) - optional,
            self.__encode_inputs(inputs, optional),
            outputs,
        ])

    def add_fallback(self, name: str, flags: int, inputs: List[Tuple[str, int, Any]]):
        if len(inputs) > 0:
            return
        if (flags & APIFlag.PAYABLE) == 0:
            return
        self.__values.append([
            APIType.FALLBACK,
            name,
            flags,
            0,
            [],
            [],
        ])

    def add_event(self, name: str, indexed: int, inputs: List[Tuple[str, int, Any]]):
        if indexed > len(inputs):
            raise Exception("IllegalIndexedCount")
        self.__values.append([
            APIType.EVENT,
            name,
            0,
            indexed,
            self.__encode_inputs(inputs, 0),
            [],
        ])

    def get_data(self):
        return self.__values


class ServiceManagerProxy:
    def __init__(self):
        self.__client = Client()
        self.__invoke = None
        self.__get_api = None
        self.__codec = None
        self.__readonly_stack = []
        self.__readonly = False
        self.__set_handlers: List[Tuple[SetHandler, int]] = []

    def connect(self, addr):
        self.__client.connect(addr)

    def send_version(self, v: int, uuid: str, name: str):
        self.__client.send(Message.VERSION, [
            v,
            uuid,
            name,
        ])

    def set_invoke_handler(self, invoke):
        self.__invoke = invoke

    def set_api_handler(self, api: Callable[[str], Tuple[int, APIInfo]]):
        self.__get_api = api

    def set_codec(self, codec: Codec) -> None:
        self.__codec = codec

    def decode(self, tag: int, val: bytes) -> 'Any':
        if tag == TypeTag.BYTES:
            return val
        elif tag == TypeTag.STRING:
            return val.decode('utf-8')
        elif tag == TypeTag.INT:
            return bytes_to_int(val)
        elif tag == TypeTag.BOOL:
            if val == b'\x00':
                return False
            elif val == b'\x01':
                return True
            else:
                raise Exception(f'IllegalBoolBytes:{val.hex()}')
        else:
            return self.__codec.decode(tag, val)

    def encode(self, o: Any) -> Optional[bytes]:
        if o is None:
            return None
        if isinstance(o, str):
            return o.encode('utf-8')
        elif isinstance(o, bytes):
            return o
        elif isinstance(o, bool):
            if o:
                return b'\x01'
            else:
                return b'\x00'
        elif isinstance(o, int):
            return int_to_bytes(o)
        else:
            t, v = self.__codec.encode(o)
            return v

    def decode_any(self, to: list) -> Any:
        tag: int = to[0]
        val: Union[bytes, dict, list] = to[1]
        if tag == TypeTag.NIL:
            return None
        elif tag == TypeTag.DICT:
            obj = {}
            for k, v in val.items():
                if isinstance(k, bytes):
                    k = k.decode('utf-8')
                obj[k] = self.decode_any(v)
            return obj
        elif tag == TypeTag.LIST:
            obj = []
            for v in val:
                obj.append(self.decode_any(v))
            return obj
        else:
            return self.decode(tag, val)

    def encode_any(self, o: Any) -> Tuple[int, Any]:
        if o is None:
            return TypeTag.NIL, None
        elif isinstance(o, dict):
            m = {}
            for k, v in o.items():
                if not isinstance(k, str):
                    raise Exception(f'InvalidKeyType:{type(k)}')
                m[k] = self.encode_any(v)
            return TypeTag.DICT, m
        elif isinstance(o, list) or isinstance(o, tuple):
            lst = []
            for v in o:
                lst.append(self.encode_any(v))
            return TypeTag.LIST, lst
        elif isinstance(o, bytes):
            return TypeTag.BYTES, o
        elif isinstance(o, str):
            return TypeTag.STRING, o.encode('utf-8')
        elif isinstance(o, bool):
            if o:
                return TypeTag.BOOL, b'\x01'
            else:
                return TypeTag.BOOL, b'\x00'
        elif isinstance(o, int):
            return TypeTag.INT, int_to_bytes(o)
        else:
            return self.__codec.encode(o)

    def __handle_invoke(self, data):
        code = self.decode(TypeTag.STRING, data[0])
        option = data[1]
        is_query = (option & InvokeFlag.READ_ONLY) != 0
        if data[2] is None:
            _from = None
        else:
            _from = self.decode(TypeTag.ADDRESS, data[2])
        _to = self.decode(TypeTag.ADDRESS, data[3])
        value = self.decode(TypeTag.INT, data[4])
        limit = self.decode(TypeTag.INT, data[5])
        method = self.decode(TypeTag.STRING, data[6])
        params = data[7]
        if isinstance(params, list):
            params = self.decode_any(params)
        info = data[8]
        if isinstance(info, list):
            info = self.decode_any(info)

        try:
            self.__readonly_stack.append(self.__readonly)
            self.__readonly = is_query
            status, step_used, result = self.__invoke(
                code, is_query, _from, _to, value, limit, method, params, info)

            self.__client.send(Message.RESULT, [
                status,
                self.encode(step_used),
                self.encode_any(result)
            ])
        except Exception as e:
            e_str = traceback.format_exc()
            self.debug(f"Exception in INVOKE:\n{e_str}", TAG)
            self.__client.send(Message.RESULT, [
                Status.SYSTEM_FAILURE,
                self.encode(limit),
                self.encode_any(f'ExceptionInInvoke({str(e)})')
            ])
        finally:
            self.__readonly = self.__readonly_stack.pop(-1)

    def __handle_get_api(self, data):
        try:
            code = self.decode(TypeTag.STRING, data)
            status, obj = self.__get_api(code)
            if status == Status.SUCCESS:
                if isinstance(obj, APIInfo):
                    self.__client.send(Message.GETAPI, [Status.SUCCESS, obj.get_data()])
                else:
                    self.__client.send(Message.GETAPI, [Status.SYSTEM_FAILURE, None])
            else:
                self.__client.send(Message.GETAPI, [status, None])
        except Exception:
            e_str = traceback.format_exc()
            self.debug(f"Exception in GETAPI:\n{e_str}", TAG)
            self.__client.send(Message.GETAPI, [Status.SYSTEM_FAILURE, None])

    def loop(self):
        while True:
            msg, data = self.__client.receive()
            if msg == Message.INVOKE:
                self.__handle_invoke(data)
            elif msg == Message.GETAPI:
                self.__handle_get_api(data)
            elif msg == Message.CLOSE:
                return

    def call(self, to: 'Address', value: int,
             step_limit: int, method: str,
             params: Any) -> Tuple[int, int, Any]:
        data = {
            'method': method,
            'params': params
        }
        self.__client.send(Message.CALL, [
            self.encode(to), self.encode(value), self.encode(step_limit),
            self.encode('call'), self.encode_any(data)
        ])

        while True:
            msg, data = self.__client.receive()
            if msg == Message.INVOKE:
                self.__handle_invoke(data)
            elif msg == Message.GETAPI:
                self.__handle_get_api(data)
            elif msg == Message.RESULT:
                return data[0], self.decode(TypeTag.INT, data[1]), self.decode_any(data[2])

    def handle_set_values(self) -> bool:
        count: int = len(self.__set_handlers)
        if count > 0:
            self.__handle_set_values(count)
            return True
        return False

    def __handle_set_values(self, cnt: int):
        for i in range(0, cnt):
            handler, size = self.__set_handlers.pop(0)
            msg, data = self.__client.receive()
            if msg != Message.SETVALUE:
                raise Exception(f'InvalidMsg({msg}) exp={Message.SETVALUE}')
            if handler is not None:
                if data[0]:
                    handler(True, data[1], size)
                else:
                    handler(False, 0, 0)

    def send_and_receive(self, msg: int, data: bytes) -> Tuple[int, Any]:
        self.__client.send(msg, data)
        self.handle_set_values()
        return self.__client.receive()

    def get_value(self, key: bytes) -> Optional[bytes]:
        msg, value = self.send_and_receive(Message.GETVALUE, key)
        if msg != Message.GETVALUE:
            raise Exception(f'InvalidMsg({msg}) exp={Message.GETVALUE}')
        if value[0]:
            return value[1]
        else:
            return None

    def set_value(self, key: bytes, value: Optional[bytes], cb: Optional[SetHandler]):
        if self.__readonly:
            raise Exception('NoPermissionToWrite')
        flag = 0
        if cb is not None:
            flag |= SetValueFlag.OLDVALUE
        if value is None:
            flag |= SetValueFlag.DELETE
        self.__client.send(Message.SETVALUE, [key, flag, value])

        if cb is not None:
            size = 0 if value is None else len(value)
            self.__set_handlers.append((cb, size))
            if len(self.__set_handlers) > MAX_SET_VALUE_HANDLERS:
                self.__handle_set_values(len(self.__set_handlers) - MAX_SET_VALUE_HANDLERS)

    def set_fee_proportion(self, pct: int):
        if self.__readonly:
            raise Exception('NoPermissionToWrite')
        pct = pct | 0
        if pct < 0 or pct > 100:
            raise Exception('InvalidParameter')
        self.__client.send(Message.SETFEEPCT, pct)

    def contains(self, prefix: bytes, value: bytes, limit: int) -> Tuple[bool, int, int]:
        msg, ret = self.send_and_receive(Message.CONTAINS, [prefix, value, limit])
        if msg != Message.CONTAINS:
            raise Exception(f'InvalidMsg({msg}) exp={Message.CONTAINS}')
        return ret[0], ret[1], ret[2]

    def get_info(self) -> Any:
        msg, value = self.send_and_receive(Message.GETINFO, b'')
        if msg != Message.GETINFO:
            raise Exception(f'InvalidMsg({msg}) exp={Message.GETINFO}')
        return self.decode_any(value)

    def get_balance(self, addr: 'Address') -> int:
        msg, value = self.send_and_receive(Message.GETBALANCE, self.encode(addr))
        if msg != Message.GETBALANCE:
            raise Exception(f'InvalidMsg({msg}) exp={Message.GETBALANCE}')
        return self.decode(TypeTag.INT, value)

    def send_event(self, indexed: List[Any], data: List[Any]):
        if self.__readonly:
            return
        self.__client.send(Message.EVENT, [
            [self.encode(v) for v in indexed],
            [self.encode(v) for v in data]
        ])

    def log(self, level: int, msg: str) -> None:
        flag = 0
        if level == Log.DEBUG or level == Log.TRACE:
            flag = LogFlag.TRACE
        self.__client.send(Message.LOG, [int(level), int(flag), str(msg)])

    def debug(self, msg: str, tag: str = 'LOG') -> None:
        self.log(Log.DEBUG, f"[{tag}] {msg}")

    def close(self):
        self.__client.close()
