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

from typing import TYPE_CHECKING, Optional, Any
from iconcommons import Logger

from ..base.address import Address
from ..base.exception import ExceptionCode, IconServiceBaseException, IconScoreException
from ..icon_constant import ICX_TRANSFER_EVENT_LOG, Status
from ..iconscore.icon_score_constant import STR_FALLBACK
from ..iconscore.icon_score_step import StepType
from .icon_score_eventlog import EventLogEmitter

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
                     func_name: str,
                     arg_params: Optional[tuple] = None,
                     kw_params: Optional[dict] = None) -> Any:
        if func_name is None:
            func_name = STR_FALLBACK
        print(f'[InterCall] from={addr_from} to={addr_to} amount={amount} func_name={func_name}')
        Logger.info(f'arg_params={arg_params}, kw_params={kw_params}', TAG)
        new_limit = context.step_counter.check_step_remained(StepType.CONTRACT_CALL)
        Logger.info(f'new_limit={new_limit}', TAG)
        if arg_params is not None:
            params = arg_params
        elif kw_params is not None:
            params = kw_params
        else:
            params = []
        status, step_used, result = \
            cls._proxy.call(addr_to, amount, new_limit, func_name, params)
        print(f'[InterCall] Result: {status}, {step_used}, {result}')

        if step_used > new_limit:
            context.step_counter.add_step(new_limit)
            raise AssertionError('Used step must be lower than the given limit')
        context.step_counter.add_step(step_used)

        if status == Status.SUCCESS:
            if amount > 0:
                InternalCall.emit_event_log_for_icx_transfer(context, addr_from, addr_to, amount)
            return result
        else:
            if status < ExceptionCode.SCORE_ERROR:
                raise IconServiceBaseException(result, status)
            else:
                code = status - ExceptionCode.SCORE_ERROR
                raise IconScoreException(result, code)

    @staticmethod
    def emit_event_log_for_icx_transfer(context: 'IconScoreContext',
                                        from_: 'Address',
                                        to: 'Address',
                                        value: int) -> None:
        event_signature = ICX_TRANSFER_EVENT_LOG
        arguments = [from_, to, value]
        indexed_args_count = 3
        EventLogEmitter.emit_event_log(context, from_, event_signature, arguments, indexed_args_count)
