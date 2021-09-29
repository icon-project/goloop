# Copyright 2019 ICON Foundation
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

from logging import Handler
from typing import Tuple, Any, List, Optional

from .base.address import Address
from .base.block import Block
from .base.message import Message
from .base.transaction import Transaction
from .icon_constant import IconScoreContextType, Status, Revision
from .iconscore.icon_score_context import IconScoreContext
from .iconscore.icon_score_step import IconScoreStepCounter
from .ipc.proxy import ServiceManagerProxy, Codec, TypeTag, APIInfo, APIType, DataType, Info, Log, SetHandler
from .logger import Logger
from .service_engine import ServiceEngine

TAG = 'PyExec'
version_number = 1


class EECodec(Codec):
    def encode(self, obj) -> Tuple[int, bytes]:
        if isinstance(obj, Address):
            return TypeTag.ADDRESS, obj.to_canonical_bytes()
        elif isinstance(obj, float):
            return TypeTag.FLOAT, str(obj).encode('utf-8')
        raise Exception(f"UnknownType: {type(obj)}")

    def decode(self, t: int, b: bytes) -> Any:
        if t == TypeTag.ADDRESS:
            return Address.from_bytes(b)
        elif t == TypeTag.FLOAT:
            return float(b.decode('utf-8'))
        raise Exception(f"UnknownType: {t}")


class EECodecV2(Codec):
    def encode(self, obj) -> Tuple[int, bytes]:
        if isinstance(obj, Address):
            return TypeTag.ADDRESS, obj.to_canonical_bytes()
        raise Exception(f"UnknownType: {type(obj)}")

    def decode(self, t: int, b: bytes) -> Any:
        if t == TypeTag.ADDRESS:
            return Address.from_bytes(b)
        raise Exception(f"UnknownType: {t}")


class ProxyStreamHandler(Handler):
    def __init__(self, proxy):
        Handler.__init__(self)
        self._proxy = proxy

    def emit(self, record):
        try:
            msg = self.format(record)
            self.write(record.levelname, msg)
        except Exception:
            self.handleError(record)

    def write(self, levelname, msg):
        self._proxy.log(Log.from_py_level(levelname), msg)


def convert_data_type(typ: str) -> int:
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
    elif typ == 'struct':
        return DataType.STRUCT
    elif typ.startswith('[]'):
        return 0x10 + convert_data_type(typ[2:])
    else:
        raise Exception(f"UnknownType: {typ}")


def convert_output_data_type(typ: str) -> int:
    if typ == 'list':
        return DataType.LIST
    elif typ == 'dict':
        return DataType.DICT
    else:
        return convert_data_type(typ)


def convert_inputs(params: list) -> Tuple[int, List[Tuple[str, int, Any]]]:
    result = list()
    optional = 0
    for param in params:
        name: str = param.get('name')
        typ: int = convert_data_type(param.get('type'))
        if 'default' in param:
            default = param.get('default')
            optional += 1
        else:
            default: Any = None
        if 'fields' in param:
            _, fields = convert_inputs(param.get('fields'))
            result.append((name, typ, default, fields))
        else:
            result.append((name, typ, default))
    return optional, result


def convert_output(params: list) -> List[int]:
    result = list()
    for param in params:
        result.append(convert_output_data_type(param.get('type')))
    return result


class PyExecEngine(object):
    ACCEPT_TX_MAP = {
        '6b93c461e6aa171f1d3ecdfb45f1fad2031899235392a90c0504ae60c044a24e': True
    }

    def __init__(self, proxy: 'ServiceManagerProxy', verify_package: bool):
        self.__proxy = proxy
        self.__verify_package = verify_package
        self.__codec = EECodec()
        self.__codec2 = EECodecV2()
        proxy.set_codec(self.__codec2)
        proxy.set_invoke_handler(self.invoke_handler)
        proxy.set_api_handler(self.api_handler)
        ServiceEngine.open(self)

    def init_logger(self, config: dict):
        Logger.load_config(config, ProxyStreamHandler(self.__proxy))

    def invoke_handler(self, code: str, is_query: bool, _from: 'Address', to: 'Address',
                       value: int, limit: int, method: str, params: Any, info: Any) -> Tuple[int, int, Any]:
        if Logger.isDebugEnabled():
            Logger.debug(f'[invoke_handle] code={repr(code)},is_query={is_query},from={_from},to={to},' +
                         f'value={value},limit={limit},method={repr(method)},params={params},info={info}', TAG)
        context = IconScoreContext(IconScoreContextType.QUERY if is_query
                                   else IconScoreContextType.INVOKE)
        context.set_invoke_params(code, to, method, params)
        tx_hash = info.get(Info.TX_HASH)
        if (tx_hash is not None and tx_hash.hex() not in self.ACCEPT_TX_MAP) or method == 'acceptScore':
            context.tx = Transaction(tx_hash=tx_hash,
                                     index=info.get(Info.TX_INDEX),
                                     origin=info.get(Info.TX_FROM),
                                     timestamp=info.get(Info.TX_TIMESTAMP),
                                     nonce=info.get(Info.TX_NONCE))
        context.block = Block(info.get(Info.BLOCK_HEIGHT),
                              info.get(Info.BLOCK_TIMESTAMP))
        context.msg = Message(sender=_from, value=value)
        context.owner = info.get(Info.CONTRACT_OWNER)
        context.step_counter = IconScoreStepCounter(info.get(Info.STEP_COSTS), limit,
                                                    self.handle_set_values)
        context.revision = info.get(Info.REVISION)
        if Revision.to_value(context.revision) < Revision.ICON2:
            self.__proxy.set_codec(self.__codec)
        else:
            self.__proxy.set_codec(self.__codec2)
        if Logger.isDebugEnabled():
            Logger.debug(f'[Transaction] {context.tx}', TAG)
            Logger.debug(f'[Block] {context.block}', TAG)
            Logger.debug(f'[Message] {context.msg}', TAG)
            Logger.debug(f'[Owner] {context.owner}', TAG)
            Logger.debug(f'[Revision] {context.revision}', TAG)
        return ServiceEngine.invoke(context)

    def api_handler(self, code: str) -> Tuple[int, APIInfo]:
        Logger.debug(f'[api_handler] code={code}', TAG)
        status, apis = ServiceEngine.get_score_api(code, self.__verify_package)
        Logger.debug(f"get_api({code}) -> {status} {apis}", TAG)
        info = APIInfo(self.__proxy)
        if status == Status.SUCCESS:
            for api in apis:
                typ = api[0]
                optional, inputs = convert_inputs(api[3])
                if typ == APIType.FUNCTION:
                    info.add_function(api[1], api[2], optional, inputs, convert_output(api[4]))
                elif typ == APIType.FALLBACK:
                    info.add_fallback(api[1], api[2], inputs)
                elif typ == APIType.EVENT:
                    info.add_event(api[1], api[2], inputs)
        return status, info

    def connect(self, addr: str, uuid: str):
        self.__proxy.connect(addr)
        self.__proxy.send_version(version_number, uuid, "python")
        # Logger should be invoked after connect
        Logger.info(f"connect({addr}, {uuid})", TAG)

    def get_info(self) -> Any:
        info = self.__proxy.get_info()
        Logger.debug(f"get_info() -> {info}", TAG)
        return info

    def call(self, to: Address, value: int, limit: int,
             method: str, params: Any) -> Tuple[int, int, Any]:
        return self.__proxy.call(to, value, limit, method, params)

    def get_value(self, k: bytes) -> Optional[bytes]:
        ret = self.__proxy.get_value(k)
        Logger.debug(f"get_value({repr(k)}) -> {repr(ret)}", TAG)
        return ret

    def set_value(self, k: bytes, v: Optional[bytes], cb: Optional[SetHandler] = None):
        Logger.debug(f"set_value({repr(k)},{repr(v)})", TAG)
        self.__proxy.set_value(k, v, cb)

    def contains(self, prefix: bytes, value: bytes, limit: int) -> Tuple[bool,int,int]:
        Logger.debug(f"contains({repr(prefix)},{repr(value)},{repr(limit)})", TAG)
        return self.__proxy.contains(prefix, value, limit)

    def set_fee_proportion(self, pct: int):
        Logger.debug(f"set_fee_proportion({repr(pct)})", TAG)
        self.__proxy.set_fee_proportion(pct)

    def handle_set_values(self) -> bool:
        return self.__proxy.handle_set_values()

    def get_balance(self, addr: Address) -> int:
        ret = self.__proxy.get_balance(addr)
        Logger.debug(f"get_balance({repr(addr)}) -> {ret}", TAG)
        return ret

    def send_event(self, indexed: List[Any], data: List[Any]):
        Logger.debug(f"send_event({indexed}, {data})", TAG)
        self.__proxy.send_event(indexed, data)

    def process(self):
        self.__proxy.loop()

    def close(self):
        self.__proxy.close()
