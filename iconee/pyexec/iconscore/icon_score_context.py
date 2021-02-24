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

import threading
import warnings
from typing import TYPE_CHECKING, Optional, List, Any

from ..base.address import Address
from ..base.block import Block
from ..base.message import Message
from ..base.transaction import Transaction
from ..icon_constant import IconScoreFuncType, IconScoreContextType
from .icon_score_step import IconScoreStepCounter
from .internal_call import ChainScore

if TYPE_CHECKING:
    from .icon_score_base import IconScoreBase

_thread_local_data = threading.local()


class ContextContainer(object):
    """
    Every class which inherits ContextContainer can share IconScoreContext instance
    in the current thread.
    """

    @staticmethod
    def _get_context() -> Optional['IconScoreContext']:
        context_stack: List['IconScoreContext'] = getattr(_thread_local_data, 'context_stack', None)

        if context_stack is not None and len(context_stack) > 0:
            # pylint: disable=unsubscriptable-object
            return context_stack[-1]
        else:
            return None

    @staticmethod
    def _push_context(context: 'IconScoreContext') -> None:
        context_stack: List['IconScoreContext'] = getattr(_thread_local_data, 'context_stack', None)

        if context_stack is None:
            context_stack = []
            setattr(_thread_local_data, 'context_stack', context_stack)

        context_stack.append(context)

    @staticmethod
    def _pop_context() -> 'IconScoreContext':
        """Delete the last pushed context of the current thread
        """
        context_stack: List['IconScoreContext'] = getattr(_thread_local_data, 'context_stack', None)

        if context_stack is not None and len(context_stack) > 0:
            return context_stack.pop()
        else:
            raise AssertionError('Failed to pop a context out of context_stack')

    @staticmethod
    def _clear_context() -> None:
        setattr(_thread_local_data, 'context_stack', None)


class ContextGetter(object):
    """
    Classes which refers to IconScoreContext should inherit ContextGetter
    """
    @property
    def _context(self) -> 'IconScoreContext':
        return ContextContainer._get_context()


class IconScoreContext(object):

    def __init__(self, typ: IconScoreContextType):
        self.type = typ

        self.code: Optional[str] = None
        self.to: Optional['Address'] = None
        self.method: Optional[str] = None
        self.params: Optional[Any] = None

        self.block: Optional['Block'] = None
        self.tx: Optional['Transaction'] = None
        self.msg: Optional['Message'] = None
        self.owner: Optional['Address'] = None
        self.step_counter: Optional['IconScoreStepCounter'] = None
        self.fee_sharing_proportion = 0

        self.revision: int = 0
        self.func_type: Optional['IconScoreFuncType'] = None

    def set_invoke_params(self, code: str, to: 'Address', method: str, params: Any):
        self.code = code
        self.to = to
        self.method = method
        self.params = params

    @property
    def readonly(self):
        return self.type == IconScoreContextType.QUERY or \
               self.func_type == IconScoreFuncType.READONLY

    def set_func_type_by_icon_score(self, icon_score: 'IconScoreBase', func_name: str):
        is_func_readonly = getattr(icon_score, '_IconScoreBase__is_func_readonly')
        if func_name is not None and is_func_readonly(func_name):
            self.func_type = IconScoreFuncType.READONLY
        else:
            self.func_type = IconScoreFuncType.WRITABLE

    def deploy(self, tx_hash: bytes) -> None:
        warnings.warn("Do not use this legacy function.", DeprecationWarning, stacklevel=2)
        ChainScore.acceptScore(self, self.to, tx_hash)
