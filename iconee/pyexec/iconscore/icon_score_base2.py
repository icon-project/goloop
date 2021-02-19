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

import hashlib
import json
from abc import ABC, ABCMeta
from enum import IntEnum
from typing import Optional, Any, Callable

from coincurve import PublicKey

from .icon_score_constant import T, FORMAT_IS_NOT_DERIVED_OF_OBJECT
from .icon_score_context import ContextContainer, IconScoreContext
from .icon_score_step import StepType
from .internal_call import InternalCall
from ..base.address import Address, AddressPrefix
from ..base.exception import InvalidParamsException, IconScoreException, InvalidInstanceException
from ..icon_constant import REVISION_COMPACT_JSON, CHARSET_ENCODING


class InterfaceScoreMeta(ABCMeta):
    def __new__(mcs, name, bases, namespace, **kwargs):
        if ABC in bases:
            return super().__new__(mcs, name, bases, namespace, **kwargs)

        cls = super().__new__(mcs, name, bases, namespace, **kwargs)
        return cls


class InterfaceScore(ABC, metaclass=InterfaceScoreMeta):
    """
    An interface class that is used to invoke other SCORE’s external method.
    """
    def __init__(self, addr_to: 'Address'):
        """
        A Python init function. Invoked when the contract call create_interface_score()
        """
        self.__addr_to = addr_to
        self.__icx = 0

    @property
    def addr_to(self) -> 'Address':
        """
        The address of SCORE to invoke

        :return: :class:`.Address` SCORE address
        """
        return self.__addr_to

    def icx(self, value: int):
        """Set the number of ICX coins to send on inter-call.

        This function can be used when you want to call payable functions of other SCOREs along with ICX coins.

        It is strongly recommended to use icx() in method chaining like the following:
        ``interface_score.icx(2 * 10 ** 18).func()``

        .. note::
            The unit of value is not icx but loop.
            1 icx is 10 ** 18 loop.

        :param value: the number of ICX coins to send (unit: loop)
        :type value: int
        :return: :class:`.InterfaceScore` object
        """
        if not (isinstance(value, int) and value >= 0):
            raise InvalidParamsException(f"Invalid ICX value")

        self.__icx = value
        return self

    def __get_icx(self) -> int:
        return self.__icx

    def __reset_icx(self):
        self.__icx = 0


class Block(object):
    def __init__(self, block_height: int, timestamp: int) -> None:
        """Constructor

        :param block_height: block height
        :param timestamp: block timestamp
        """
        self._height = block_height
        # unit: microsecond
        self._timestamp = timestamp

    @property
    def height(self) -> int:
        return self._height

    @property
    def timestamp(self) -> int:
        return self._timestamp


class Icx(object):
    """
    Class for handling ICX coin transfer
    """

    def __init__(self, context: 'IconScoreContext', address: 'Address') -> None:
        """Constructor
        """
        self._context = context
        self._address = address

    def transfer(self, addr_to: 'Address', amount: int) -> None:
        """
        transfer the amount of icx to the given 'addr_to'
        If failed, an exception will be raised

        :param addr_to: receiver address
        :param amount: the amount of icx to transfer
        """
        InternalCall.message_call(self._context, self._address, addr_to, amount, None)

    def send(self, addr_to: 'Address', amount: int) -> bool:
        """
        transfer the amount of icx to the given 'addr_to'

        :param addr_to: receiver address
        :param amount: the amount of icx to transfer
        :return: True(success) False(failed)
        """
        try:
            self.transfer(addr_to, amount)
            return True
        except:
            return False

    def get_balance(self, address: 'Address') -> int:
        """
        Returns the ICX balance of given address

        :param address: address
        :return: ICX balance of given address
        """
        return InternalCall.icx_get_balance(address)


class ScoreApiStepRatio(IntEnum):
    SHA3_256 = 1000
    SHA_256 = 1000
    CREATE_ADDRESS_WITH_COMPRESSED_KEY = 15000
    CREATE_ADDRESS_WITH_UNCOMPRESSED_KEY = 1500
    JSON_DUMPS = 5000
    JSON_LOADS = 4000
    RECOVER_KEY = 70000


def _get_api_call_step_cost(context: 'IconScoreContext', ratio: ScoreApiStepRatio) -> int:
    """Returns the step cost for a given SCORE API

    API CALL step cost in context.step_counter means the step cost of sha3_256(b'')
    Each step cost for other APIs is calculated from the relative ratio based on sha3_256(b'')

    other_api_call_step_cost =
        API_CALL_STEP_COST * other_api_call_ratio // ScoreApiStepRatio.SHA3_256

    :param context: IconScoreContext instance
    :param ratio: The ratio of a given SCORE API based on ScoreApiStepRatio.SHA3_256
    :return: step_cost for a given SCORE API
    """
    api_call_step: int = context.step_counter.get_step_cost(StepType.API_CALL)
    return api_call_step * ratio // ScoreApiStepRatio.SHA3_256


def revert(message: Optional[str] = None, code: int = 0) -> None:
    """
    Reverts the transaction and breaks.
    All the changes of state DB in current transaction will be rolled back.

    :param message: revert message
    :param code: code
    """
    try:
        if not isinstance(code, int):
            code = int(code)

        if not isinstance(message, str):
            message = str(message)
    except:
        raise InvalidParamsException("Revert error: code or message is invalid")
    else:
        raise IconScoreException(message, code)


def sha3_256(data: bytes) -> bytes:
    """
    Computes sha3_256 hash using the input data

    :param data: input data
    :return: hashed data in bytes
    """
    return _hash("sha3_256", data)


def sha_256(data: bytes) -> bytes:
    """
    Computes sha256 hash using the input data

    :param data: input data
    :return: hashed data in bytes
    """
    return _hash("sha256", data)


def _hash(name: str, data: bytes) -> bytes:
    """Protected hash function

    :param name: hash function name: "sha256" or "sha3_256"
    :param data: data to hash
    :return: hashed data in bytes
    """
    if name not in ("sha3_256", "sha256"):
        raise InvalidParamsException(f"Not supported: {name}")
    if not isinstance(data, bytes):
        raise InvalidParamsException("Invalid dataType")

    context = ContextContainer._get_context()
    assert context

    if context:
        size = len(data)
        chunks = size // 32
        if size % 32 > 0:
            chunks += 1

        step_cost: int = context.step_counter.get_step_cost(StepType.API_CALL)
        step: int = step_cost + step_cost * chunks // 10

        context.step_counter.consume_step(StepType.API_CALL, step)

    func = getattr(hashlib, name)
    return func(data).digest()


def json_dumps(obj: Any) -> str:
    """
    Converts a python object `obj` to a JSON string

    :param obj: a python object to be converted
    :return: json string
    """
    context = ContextContainer._get_context()
    assert context

    if context and (context.revision & REVISION_COMPACT_JSON) != 0:
        ret: str = json.dumps(obj, separators=(',', ':'))

        step_cost: int = _get_api_call_step_cost(context, ScoreApiStepRatio.JSON_DUMPS)
        step: int = step_cost + step_cost * len(ret.encode(CHARSET_ENCODING)) // 100

        context.step_counter.consume_step(StepType.API_CALL, step)
    else:
        ret: str = json.dumps(obj)

    return ret


def json_loads(src: str) -> Any:
    """
    Parses a JSON string `src` and converts it to a python object

    :param src: a JSON string to be converted
    :return: a python object
    """
    if not isinstance(src, str):
        return None

    context = ContextContainer._get_context()
    assert context

    if context and (context.revision & REVISION_COMPACT_JSON) != 0:
        step_cost: int = _get_api_call_step_cost(context, ScoreApiStepRatio.JSON_LOADS)
        step: int = step_cost + step_cost * len(src.encode(CHARSET_ENCODING)) // 100

        context.step_counter.consume_step(StepType.API_CALL, step)

    return json.loads(src)


def create_address_with_key(public_key: bytes) -> Optional['Address']:
    """Create an address with a given public key

    :param public_key: Public key based on secp256k1
    :return: Address created from a given public key or None if failed
    """
    if not isinstance(public_key, bytes):
        return None

    # 33: prefix(1 byte) + keyBody(32 bytes)
    # 65: prefix(1 byte) + keyBody(64 bytes)
    key_size: int = len(public_key)
    if key_size not in (33, 65):
        return None

    context = ContextContainer._get_context()
    assert context

    if context:
        if key_size == 33:
            ratio = ScoreApiStepRatio.CREATE_ADDRESS_WITH_COMPRESSED_KEY
        else:
            ratio = ScoreApiStepRatio.CREATE_ADDRESS_WITH_UNCOMPRESSED_KEY

        step: int = _get_api_call_step_cost(context, ratio)
        context.step_counter.consume_step(StepType.API_CALL, step)

    try:
        return _create_address_with_key(public_key)
    except:
        return None


def _create_address_with_key(public_key: bytes) -> Optional['Address']:
    assert isinstance(public_key, bytes)
    assert len(public_key) in (33, 65)

    size = len(public_key)
    prefix: int = public_key[0]

    if size == 33 and prefix in (0x02, 0x03):
        uncompressed_public_key: bytes = _convert_key(public_key, compressed=True)
    elif size == 65 and prefix == 0x04:
        uncompressed_public_key: bytes = public_key
    else:
        return None

    body: bytes = hashlib.sha3_256(uncompressed_public_key[1:]).digest()[-20:]
    return Address(AddressPrefix.EOA, body)


def _convert_key(public_key: bytes, compressed: bool) -> Optional[bytes]:
    """Convert key between compressed and uncompressed keys

    :param public_key: compressed or uncompressed key
    :return: the counterpart key of a given public_key
    """
    public_key_object = PublicKey(public_key)
    return public_key_object.format(compressed=not compressed)


def recover_key(msg_hash: bytes, signature: bytes, compressed: bool = True) -> Optional[bytes]:
    """Returns the public key from message hash and recoverable signature

    :param msg_hash: 32 bytes data
    :param signature: signature_data(64) + recovery_id(1)
    :param compressed: the type of public key to return
    :return: public key recovered from msg_hash and signature
        (compressed: 33 bytes key, uncompressed: 65 bytes key)
    """
    context = ContextContainer._get_context()
    assert context

    if context:
        step_cost: int = _get_api_call_step_cost(context, ScoreApiStepRatio.RECOVER_KEY)
        context.step_counter.consume_step(StepType.API_CALL, step_cost)

    try:
        return _recover_key(msg_hash, signature, compressed)
    except:
        return None


def _recover_key(msg_hash: bytes, signature: bytes, compressed: bool) -> Optional[bytes]:
    if isinstance(msg_hash, bytes) \
            and len(msg_hash) == 32 \
            and isinstance(signature, bytes) \
            and len(signature) == 65:
        return PublicKey.from_signature_and_message(signature, msg_hash, hasher=None).format(compressed)

    return None


def create_interface_score(addr_to: 'Address',
                           interface_cls: Callable[['Address'], T]) -> T:
    """
    Creates an object, through which you have an access to the designated SCORE’s external functions.

    :param addr_to: SCORE address
    :param interface_cls: interface class
    :return: An instance of given class
    """
    if interface_cls is InterfaceScore:
        raise InvalidInstanceException(FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(InterfaceScore.__name__))
    return interface_cls(addr_to)
