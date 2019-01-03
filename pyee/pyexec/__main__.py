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

import os
import sys
from typing import Tuple, Any, Union, List

from .icon_constant import IconScoreContextType
from .base.address import Address
from .base.block import Block
from .base.message import Message
from .base.transaction import Transaction
from .service_engine import ServiceEngine, IconScoreContext
from .iconscore.icon_score_step import IconScoreStepCounter
from .ipc.proxy import ServiceManagerProxy, Codec, TypeTag, APIInfo, APIType, DataType

server_address = '/tmp/pyee_uds_socket'
version_number = 1


class EECodec(Codec):
    def encode(self, obj) -> Tuple[int, bytes]:
        if isinstance(obj, Address):
            return TypeTag.ADDRESS, obj.to_bytes()
        raise Exception

    def decode(self, t: int, b: bytes) -> Any:
        if t == TypeTag.ADDRESS:
            return Address.from_bytes(b)
        else:
            raise Exception(f"UnknownType: {type(t)}")


def convert_data_type(typ: str) -> DataType:
    if typ == 'int':
        return DataType.INTEGER
    elif typ == 'str':
        return DataType.STRING
    elif typ == 'bytes':
        return DataType.BYTES
    elif typ == 'bool':
        return DataType.BOOL
    elif typ == 'Address':
        return DataType.ADDRESS
    else:
        raise Exception(f"UnknownType: {typ}")


def convert_inputs(params: list) -> List[Tuple[str, int]]:
    result = list()
    for param in params:
        name: str = param.get('name')
        typ: int = convert_data_type(param.get('type'))
        # TODO optional parameter must be set.
        default: Any = None
        result.append((name, typ, default))
    return result


def convert_output(params: list) -> List[int]:
    result = list()
    for param in params:
        result.append(convert_data_type(param.get('type')))
    return result


class PyExecEngine(object):
    def __init__(self, proxy: 'ServiceManagerProxy'):
        self.__proxy = proxy
        proxy.set_codec(EECodec())
        proxy.set_invoke_handler(self.invoke_handler)
        proxy.set_api_handler(self.api_handler)
        ServiceEngine.open(self)

    def invoke_handler(self, code: str, is_query: bool, _from: 'Address', to: 'Address',
                       value: int, limit: int, method: str, params: Any) -> Tuple[int, int, Any]:
        print(f'\ninvoke_handler(code={repr(code)},is_query={is_query},from={_from},to={to},' +
              f'value={value},limit={limit},method={repr(method)},params={params})')
        context = IconScoreContext(IconScoreContextType.QUERY if is_query
                                   else IconScoreContextType.INVOKE)
        context.set_invoke_params(code, to, method, params)
        # Get transaction info and set the context
        info = self.get_info()
        context.tx = Transaction(tx_hash=info.get('T.hash'),
                                 index=info.get('T.index'),
                                 origin=_from,
                                 timestamp=info.get('T.timestamp'),
                                 nonce=info.get('T.nonce'))
        context.block = Block(info.get('B.height'),
                              info.get('B.hash'),
                              info.get('B.timestamp'),
                              info.get('B.prevhash'))
        context.msg = Message(sender=_from, value=value)
        context.owner: Address = info.get('Owner')
        context.step_counter = IconScoreStepCounter(info.get('StepCosts'),
                                                    limit)
        print(f'[Transaction] {context.tx}')
        print(f'[Block] {context.block}')
        print(f'[Message] {context.msg}')
        print(f'[Owner] {context.owner}')
        print(f'[StepCounter] {context.step_counter}')
        return ServiceEngine.invoke(context)

    def api_handler(self, code: str) -> APIInfo:
        print(f'[api_handler] code={code}')
        apis = ServiceEngine.get_score_api(code)
        print(f"get_api({code}) -> {apis}")
        info = APIInfo(self.__proxy)
        for api in apis:
            typ = api[0]
            if typ == APIType.FUNCTION:
                # TODO optional parameter count must be applied.
                optional = 0
                info.add_function(api[1], api[2], optional, convert_inputs(api[3]), convert_output(api[4]))
            elif typ == APIType.FALLBACK:
                info.add_fallback(api[1], api[2], convert_inputs(api[3]))
            elif typ == APIType.EVENT:
                info.add_event(api[1], api[2], convert_inputs(api[3]))
        return info

    def connect(self, addr: str):
        print(f"connect({addr})")
        self.__proxy.connect(addr)
        self.__proxy.send_version(version_number, os.getpid(), "python")

    def get_info(self) -> Any:
        info = self.__proxy.get_info()
        print(f"get_info() -> {info}")
        return info

    def call(self, to: Address, value: int, limit: int,
             method: str, params: Any) -> Tuple[int, int, Any]:
        return self.__proxy.call(to, value, limit, method, params)

    def get_value(self, k: bytes) -> Union[bytes, None]:
        ret = self.__proxy.get_value(k)
        print(f"get_value({repr(k)}) -> {repr(ret)}")
        return ret

    def set_value(self, k: bytes, v: Union[bytes, None]):
        print(f"set_value({repr(k)},{repr(v)})")
        self.__proxy.set_value(k, v)

    def get_balance(self, addr: Address) -> int:
        ret = self.__proxy.get_balance(addr)
        print(f"get_balance({repr(addr)}) -> {ret}")
        return ret

    def send_event(self, indexed: List[Any], data: List[Any]):
        print(f"send_event({indexed},{data})")
        self.__proxy.send_event(indexed, data)

    def process(self):
        self.__proxy.loop()


def main():
    engine = PyExecEngine(ServiceManagerProxy())
    engine.connect(server_address)
    engine.process()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("exit")
