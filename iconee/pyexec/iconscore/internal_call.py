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

from copy import deepcopy
from typing import TYPE_CHECKING, Optional, Any, Tuple

from ..base.address import Address, ZERO_SCORE_ADDRESS
from ..base.exception import ExceptionCode, IconServiceBaseException, IconScoreException
from ..icon_constant import Revision
from ..iconscore.icon_score_constant import STR_FALLBACK
from ..ipc import MethodName
from ..logger import Logger

if TYPE_CHECKING:
    from .icon_score_context import IconScoreContext

TAG = 'InternalCall'


class InternalCall(object):
    _proxy = None

    @classmethod
    def open(cls, proxy):
        cls._proxy = proxy

    @classmethod
    def icx_get_balance(cls, address: 'Address') -> int:
        return cls._proxy.get_balance(address)

    @classmethod
    def message_call(cls,
                     context: 'IconScoreContext',
                     addr_from: 'Address',
                     addr_to: 'Address',
                     amount: int,
                     func_name: Optional[str],
                     arg_params: Optional[tuple] = None,
                     kw_params: Optional[dict] = None) -> Any:
        if func_name is None or func_name == STR_FALLBACK:
            func_name = MethodName.FALLBACK
        new_limit = context.step_counter.step_remained
        if kw_params is not None and len(kw_params) > 0:
            params = deepcopy(kw_params)
            if arg_params is not None and len(arg_params) > 0:
                params["."] = arg_params
        elif arg_params is not None:
            params = arg_params
        else:
            params = []
        if Logger.isDebugEnabled():
            Logger.debug(f'>>> from={addr_from} to={addr_to} amount={amount} func_name={func_name}', TAG)
            Logger.debug(f'    new_limit={new_limit}, params={params}', TAG)
        status, step_used, result = \
            cls._proxy.call(addr_to, amount, new_limit, func_name, params)
        Logger.debug(f'<<< Result: {status}, {step_used}, {result}', TAG)

        if step_used > new_limit:
            context.step_counter.add_step(new_limit)
            raise AssertionError('Used step must be lower than the given limit')
        context.step_counter.add_step(step_used)

        if status == ExceptionCode.OK:
            return result
        else:
            if Revision.to_value(context.revision) < Revision.ICON2:
                if status == 1:
                    if result == "NoAccount":
                        raise AttributeError("'NoneType' object has no attribute '_IconScoreBase__is_func_readonly'")
                    elif result != "cxd8cd722d7e546809d3cd1dfa6e1ceddeefba759a":
                        raise Exception(result)
                elif status >= ExceptionCode.SCORE_ERROR:
                    if result == "Invalid bool value":
                        raise IconScoreException(result)
                    else:
                        raise IconScoreException(result, status - ExceptionCode.SCORE_ERROR)
            raise IconServiceBaseException.create(result, status)


class ChainScore(object):

    @staticmethod
    def acceptScore(context: 'IconScoreContext',
                    _from: 'Address',
                    tx_hash: bytes) -> Optional['Address']:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'acceptScore', tuple([tx_hash]))

    @staticmethod
    def rejectScore(context: 'IconScoreContext',
                    _from: 'Address',
                    tx_hash: bytes) -> Optional['Address']:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'rejectScore', tuple([tx_hash]))

    @staticmethod
    def txHashToAddress(context: 'IconScoreContext',
                        _from: 'Address',
                        tx_hash: bytes) -> Optional['Address']:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'txHashToAddress', tuple([tx_hash]))

    @staticmethod
    def addressToTxHashes(context: 'IconScoreContext',
                          _from: 'Address',
                          score_address: 'Address') -> Tuple[Optional[bytes], Optional[bytes]]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'addressToTxHashes', tuple([score_address]))

    @staticmethod
    def getScoreStatus(context: 'IconScoreContext',
                       _from: 'Address',
                       score_address: 'Address') -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'getScoreStatus', tuple([score_address]))

    @staticmethod
    def getRevision(context: 'IconScoreContext', _from: 'Address') -> Optional[int]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getRevision')

    @staticmethod
    def setRevision(context: 'IconScoreContext', _from: 'Address', code: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setRevision', tuple([code]))

    @staticmethod
    def getStepPrice(context: 'IconScoreContext', _from: 'Address') -> Optional[int]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getStepPrice')

    @staticmethod
    def setStepPrice(context: 'IconScoreContext', _from: 'Address', price: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setStepPrice', tuple([price]))

    @staticmethod
    def getStepCosts(context: 'IconScoreContext', _from: 'Address') -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getStepCosts')

    @staticmethod
    def getMaxStepLimit(context: 'IconScoreContext', _from: 'Address', _type: str) -> Optional[int]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'getMaxStepLimit', tuple([_type]))

    @staticmethod
    def blockScore(context: 'IconScoreContext', _from: 'Address', address: 'Address'):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'blockScore', tuple([address]))

    @staticmethod
    def unblockScore(context: 'IconScoreContext', _from: 'Address', address: 'Address'):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'unblockScore', tuple([address]))

    @staticmethod
    def getBlockedScores(context: 'IconScoreContext', _from: 'Address') -> Optional[list]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getBlockedScores')

    @staticmethod
    def getIRep(context: 'IconScoreContext', _from: 'Address') -> Optional[int]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getIRep')

    @staticmethod
    def setIRep(context: 'IconScoreContext', _from: 'Address', value: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setIRep', tuple([value]))

    @staticmethod
    def getPRepTerm(context: 'IconScoreContext', _from: 'Address') -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getPRepTerm')

    @staticmethod
    def getMainPReps(context: 'IconScoreContext', _from: 'Address') -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getMainPReps')

    @staticmethod
    def getSubPReps(context: 'IconScoreContext', _from: 'Address') -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0, 'getSubPReps')

    @staticmethod
    def disqualify_prep(context: 'IconScoreContext', _from: 'Address', address: 'Address') -> Tuple[bool, str]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'disqualifyPRep', tuple([address]))

    @staticmethod
    def validate_irep(context: 'IconScoreContext', _from: 'Address', irep: int) -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'validateIRep', tuple([irep]))

    @staticmethod
    def validate_reward_fund(context: 'IconScoreContext', _from: 'Address', iglobal: int) -> Optional[dict]:
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'validateRewardFund', tuple([iglobal]))

    @staticmethod
    def setRewardFund(context: 'IconScoreContext', _from: 'Address', iglobal: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setRewardFund', tuple([iglobal]))

    @staticmethod
    def setRewardFundAllocation(
            context: 'IconScoreContext', _from: 'Address', iprep: int, icps: int, irelay: int, ivoter: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setRewardFundAllocation', tuple([iprep, icps, irelay, ivoter]))

    @staticmethod
    def setStepCost(context: 'IconScoreContext', _from: 'Address', _type: str, cost: int):
        return InternalCall.message_call(context, _from, ZERO_SCORE_ADDRESS, 0,
                                         'setStepCost', tuple([_type, cost]))
