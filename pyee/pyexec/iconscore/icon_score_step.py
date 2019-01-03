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

from enum import Enum, auto
from threading import Lock
from typing import TYPE_CHECKING

from ..icon_constant import MAX_EXTERNAL_CALL_COUNT
from ..utils import to_camel_case
from ..base.exception import IconServiceBaseException, ExceptionCode, InvalidRequestException

if TYPE_CHECKING:
    from iconservice.iconscore.icon_score_context import IconScoreContextType


class AutoValueEnum(Enum):
    # noinspection PyMethodParameters
    def _generate_next_value_(name, start, count, last_values):
        # Generates value from the camel-cased name
        return to_camel_case(name.lower())


class StepType(AutoValueEnum):
    # DEFAULT = auto()
    CONTRACT_CALL = auto()
    # CONTRACT_CREATE = auto()
    # CONTRACT_UPDATE = auto()
    # CONTRACT_DESTRUCT = auto()
    # CONTRACT_SET = auto()
    GET = auto()
    SET = auto()
    REPLACE = auto()
    DELETE = auto()
    INPUT = auto()
    EVENT_LOG = auto()
    API_CALL = auto()


# class IconScoreStepCounterFactory(object):
#     """Creates a step counter for the transaction
#     """
#
#     def __init__(self) -> None:
#         self._lock = Lock()
#         self._step_price = 0
#         self._step_costs = {}
#         self._max_step_limits = {}
#
#     def set_step_properties(self, step_price=None, step_costs=None, max_step_limits=None):
#         """Sets the STEP properties if exists
#
#         :param step_price: step price
#         :param step_costs: step cost dict
#         :param max_step_limits: max step limit dict
#         """
#         with self._lock:
#             if step_price is not None:
#                 self._step_price = step_price
#             if step_costs is not None:
#                 self._step_costs = step_costs
#             if max_step_limits is not None:
#                 self._max_step_limits = max_step_limits
#
#     def get_step_price(self):
#         """Returns the step price
#
#         :return: step price
#         """
#         with self._lock:
#             return self._step_price
#
#     def set_step_price(self, step_price: int):
#         """Sets the step price
#
#         :param step_price: step price
#         """
#         with self._lock:
#             self._step_price = step_price
#
#     def get_step_cost(self, step_type: 'StepType') -> int:
#         with self._lock:
#             return self._step_costs.get(step_type, 0)
#
#     def set_step_cost(self, step_type: 'StepType', value: int):
#         """Sets the step cost for specific action.
#
#         :param step_type: specific action
#         :param value: step cost
#         """
#         with self._lock:
#             self._step_costs[step_type] = value
#
#     def get_max_step_limit(self, context_type: 'IconScoreContextType') -> int:
#         """Returns the max step limit
#
#         :return: the max step limit
#         """
#         with self._lock:
#             return self._max_step_limits.get(context_type, 0)
#
#     def set_max_step_limit(
#             self, context_type: 'IconScoreContextType', max_step_limit: int):
#         """Sets the max step limit
#
#         :param context_type: context type
#         :param max_step_limit: the max step limit for the context type
#         """
#         with self._lock:
#             self._max_step_limits[context_type] = max_step_limit
#
#     def create(self, context_type: 'IconScoreContextType') -> 'IconScoreStepCounter':
#         """Creates a step counter for the transaction
#
#         :param context_type: context type
#         :return: step counter
#         """
#         with self._lock:
#             step_price: int = self._step_price
#             # Copying a `dict` so as not to change step costs when processing a transaction.
#             step_costs: dict = self._step_costs.copy()
#             max_step_limit: int = self._max_step_limits.get(context_type, 0)
#
#         return IconScoreStepCounter(step_price, step_costs, max_step_limit)


class OutOfStepException(IconServiceBaseException):
    """ An Exception which is thrown when steps are exhausted.
    """

    def __init__(self, step_limit: int, step_used: int,
                 requested_step: int, step_type: StepType) -> None:
        """Constructor

        :param step_limit: step limit of the transaction
        :param step_used: used steps in the transaction
        :param requested_step: consuming steps before the exception is thrown
        :param step_type: step type that
        the exception has been thrown when processing
        """
        self.__step_limit: int = step_limit
        self.__step_used = step_used
        self.__requested_step = requested_step
        self.__step_type = step_type

    @property
    def code(self) -> int:
        return ExceptionCode.SCORE_ERROR

    @property
    def message(self) -> str:
        """
        Returns the exception message
        :return: the exception message
        """
        return f'Out of step: {self.__step_type.value}'

    @property
    def step_limit(self) -> int:
        """
        Returns step limit of the transaction
        :return: step limit of the transaction
        """
        return self.__step_limit

    @property
    def step_used(self) -> int:
        """
        Returns used steps before the exception is thrown in the transaction
        :return: used steps in the transaction
        """
        return self.__step_used

    @property
    def requested_step(self) -> int:
        """
        Returns consuming steps before the exception is thrown.
        :return: Consuming steps before the exception is thrown.
        """
        return self.__requested_step


class IconScoreStepCounter(object):
    """ Counts steps in a transaction
    """

    def __init__(self,
                 step_costs: dict,
                 step_limit: int) -> None:
        """Constructor

        :param step_costs: a dict of base step costs
        :param step_limit: step limit for current context type
        """
        converted_step_costs = {}
        for key, value in step_costs.items():
            try:
                converted_step_costs[StepType(key)] = value
            except ValueError:
                # Pass the unknown step type
                pass
        self._step_costs: dict = converted_step_costs
        self._step_limit: int = step_limit
        self._step_used: int = 0
        # self._external_call_count: int = 0

    # @property
    # def step_price(self) -> int:
    #     """
    #     Returns the step price
    #     :return: step price
    #     """
    #     return self._step_price

    # @property
    # def max_step_limit(self) -> int:
    #     """
    #     Returns max step limit for current context
    #     :return: max step limit
    #     """
    #     return self._max_step_limit

    @property
    def step_limit(self) -> int:
        """
        Returns step limit of the transaction
        :return: step limit of the transaction
        """
        return self._step_limit

    @property
    def step_used(self) -> int:
        """
        Returns used steps in the transaction
        :return: used steps in the transaction
        """
        # return max(self._step_used,
        #            self._step_costs.get(StepType.DEFAULT, 0))
        return self._step_used

    def step_remained(self) -> int:
        return self._step_limit - self._step_used

    def apply_step(self, step_type: StepType, count: int) -> int:
        """ Increases steps for given step cost
        """

        # if step_type == StepType.CONTRACT_CALL:
        #     self._external_call_count += 1
        #     if self._external_call_count > MAX_EXTERNAL_CALL_COUNT:
        #         raise InvalidRequestException('Too many external calls')

        step_to_apply = self._step_costs.get(step_type, 0) * count
        if step_to_apply + self._step_used > self._step_limit:
            step_used = self._step_used
            self._step_used = self._step_limit
            raise OutOfStepException(
                self._step_limit, step_used, step_to_apply, step_type)

        self._step_used += step_to_apply
        return self._step_used

    def add_step(self, amount: int) -> int:
        # Assuming amount is always less than the current limit
        self._step_used += amount
        return self._step_used

    # def reset(self, step_limit: int):
    #     """
    #
    #     :return:
    #     """
    #     self._step_limit: int = min(step_limit, self._max_step_limit)
    #     self._step_used: int = 0
    #     self._external_call_count: int = 0

    # def set_step_price(self, step_price: int):
    #     """Sets the step price
    #
    #     :param step_price: step price
    #     """
    #     self._step_price = step_price

    # def set_step_costs(self, step_costs: dict):
    #     """Sets the step costs dict
    #
    #     :param step_costs: step costs dict
    #     """
    #     self._step_costs = step_costs

    # def set_max_step_limit(self, max_step_limit: int):
    #     """Sets the max step limit for current context
    #
    #     :param max_step_limit: max step limit
    #     """
    #     self._max_step_limit = max_step_limit
