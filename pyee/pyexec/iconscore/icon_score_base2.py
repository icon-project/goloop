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
from typing import TYPE_CHECKING, Optional, Any

from ..base.address import Address
from ..base.exception import InvalidParamsException, IconScoreException
from .icon_score_constant import STR_FALLBACK
from .icon_score_context import ContextContainer, IconScoreContext
from .internal_call import InternalCall


if TYPE_CHECKING:
    from .icon_score_base import IconScoreBase


class InterfaceScoreMeta(ABCMeta):
    def __new__(mcs, name, bases, namespace, **kwargs):
        if ABC in bases:
            return super().__new__(mcs, name, bases, namespace, **kwargs)

        cls = super().__new__(mcs, name, bases, namespace, **kwargs)
        return cls


class InterfaceScore(ABC, metaclass=InterfaceScoreMeta):
    def __init__(self, addr_to: 'Address', from_score: 'IconScoreBase'):
        self.__addr_to = addr_to
        self.__from_score = from_score

    @property
    def addr_to(self) -> 'Address':
        return self.__addr_to

    @property
    def from_score(self) -> 'IconScoreBase':
        return self.__from_score


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
        InternalCall.message_call(self._context, self._address, addr_to, amount, STR_FALLBACK)

    def send(self, addr_to: 'Address', amount: int) -> bool:
        """
        transfer the amount of icx to the given 'addr_to'

        :param addr_to: receiver address
        :param amount: the amount of icx to transfer
        :return: True(success) False(failed)
        """
        try:
            self.transfer(addr_to, amount)
            if not addr_to.is_contract and self._is_icx_send_defective():
                return False
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

    # noinspection PyBroadException
    def _is_icx_send_defective(self) -> bool:
        # try:
        #     governance_score = IconScoreContextUtil.get_icon_score(self._context, GOVERNANCE_SCORE_ADDRESS)
        #     if governance_score is not None:
        #         if hasattr(governance_score, 'getVersion'):
        #             version = governance_score.getVersion()
        #             return version == '0.0.2'
        # except BaseException:
        #     pass

        return False


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
    Computes hash using the input data

    :param data: input data
    :return: hashed data in bytes
    """
    # context = ContextContainer._get_context()
    # if context.step_counter:
    #     step_count = 1
    #     if data:
    #         step_count += len(data)
    #     context.step_counter.apply_step(StepType.API_CALL, step_count)

    return hashlib.sha3_256(data).digest()


def json_dumps(obj: Any, **kwargs) -> str:
    """
    Converts a python object `obj` to a JSON string

    :param obj: a python object to be converted
    :param kwargs: json options (see https://docs.python.org/3/library/json.html#json.dumps)
    :return: json string
    """
    return json.dumps(obj, **kwargs)


def json_loads(src: str, **kwargs) -> Any:
    """
    Parses a JSON string `src` and converts it to a python object

    :param src: a JSON string to be converted
    :param kwargs: kwargs: json options (see https://docs.python.org/3/library/json.html#json.loads)
    :return: a python object
    """
    return json.loads(src, **kwargs)
