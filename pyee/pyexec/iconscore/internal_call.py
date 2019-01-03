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

from .icon_score_eventlog import EventLogEmitter
from .icon_score_step import StepType
from ..base.address import Address
from ..base.exception import ExceptionCode, IconScoreException
from ..icon_constant import ICX_TRANSFER_EVENT_LOG, Status
from ..utils import check_error_response

if TYPE_CHECKING:
    from .icon_score_context import IconScoreContext


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

        context.step_counter.apply_step(StepType.CONTRACT_CALL, 1)

        print(f'<== InternalCall.message_call: from={addr_from} to={addr_to} amount={amount} func_name={func_name}')
        print(f'    arg_params={arg_params}')
        print(f'    kw_params={kw_params}')
        new_limit = context.step_counter.step_remained()
        status, step_used, result = \
            cls._proxy.call(addr_to, amount, new_limit, func_name, arg_params)
        print(f'==> call result: {status}, {step_used}, {result}')

        if step_used > new_limit:
            context.step_counter.add_step(new_limit)
            raise AssertionError('Used step must be lower than the given limit')
        context.step_counter.add_step(step_used)

        if status == Status.SUCCESS:
            if amount > 0:
                InternalCall.emit_event_log_for_icx_transfer(context, addr_from, addr_to, amount)
            return result
        else:
            if check_error_response(result):
                error = result.get('error')
                code = error.get('code', ExceptionCode.INTERNAL_ERROR)
                message = error.get('message')
                raise IconScoreException(message, code)
            else:
                raise AssertionError('Result must be an error response')

    @staticmethod
    def emit_event_log_for_icx_transfer(context: 'IconScoreContext',
                                        from_: 'Address',
                                        to: 'Address',
                                        value: int) -> None:
        event_signature = ICX_TRANSFER_EVENT_LOG
        arguments = [from_, to, value]
        indexed_args_count = 3
        EventLogEmitter.emit_event_log(context, from_, event_signature, arguments, indexed_args_count)
