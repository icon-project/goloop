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

from ..base.exception import IconServiceBaseException, ExceptionCode
from ..utils import to_camel_case


class AutoValueEnum(Enum):
    # noinspection PyMethodParameters
    # pylint: disable=no-self-argument,no-member
    def _generate_next_value_(name, start, count, last_values):
        # Generates value from the camel-cased name
        return to_camel_case(name.lower())


class StepType(AutoValueEnum):
    SCHEMA = auto()
    # version 0
    GET = auto()
    SET = auto()
    REPLACE = auto()  # obsolete in v1
    DELETE = auto()
    EVENT_LOG = auto()  # obsolete in v1
    API_CALL = auto()
    # version 1
    GET_BASE = auto()
    SET_BASE = auto()
    DELETE_BASE = auto()
    LOG_BASE = auto()
    LOG = auto()


class OutOfStepException(IconServiceBaseException):
    """ An Exception which is thrown when steps are exhausted.
    """

    def __init__(self,
                 step_limit: int,
                 step_used: int,
                 requested_step: int,
                 step_type: StepType) -> None:
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
        return ExceptionCode.OUT_OF_STEP

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
                 step_limit: int,
                 refund_handler: callable) -> None:
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
        self._refund_handler = refund_handler

        self._schema: int = self.get_step_cost(StepType.SCHEMA)
        if self._schema == 0:
            self._step_base = {}
            # cache the refund cost (for v0)
            # assuming cost_set is greater than cost_replace
            cost_set: int = self.get_step_cost(StepType.SET)
            cost_replace: int = self.get_step_cost(StepType.REPLACE)
            self._refund_cost = cost_replace - cost_set
        else:
            self._step_base = {
                StepType.GET: self.get_step_cost(StepType.GET_BASE),
                StepType.SET: self.get_step_cost(StepType.SET_BASE),
                StepType.DELETE: self.get_step_cost(StepType.DELETE_BASE),
                StepType.LOG: self.get_step_cost(StepType.LOG_BASE)
            }
            # cache the refund base (for v1)
            replace_base = (self.get_base_step(StepType.SET) + self.get_base_step(StepType.DELETE)) // 2
            self._refund_base = replace_base - self.get_base_step(StepType.SET)

    @property
    def schema(self) -> int:
        return self._schema

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
        return self._step_used

    @property
    def step_remained(self) -> int:
        """ Returns the remained steps
        """
        self._refund_handler()
        return self._step_limit - self._step_used

    def apply_step(self, step_type: StepType, count: int) -> int:
        """ Increases steps for given step cost
        """
        base = self.get_base_step(step_type)
        step: int = base + self.get_step_cost(step_type) * count
        if step == 0:
            return self._step_used
        return self.consume_step(step_type, step)

    def consume_step(self, step_type: StepType, step: int) -> int:
        step_used: int = self._step_used + step

        while step_used > self._step_limit:
            if self._refund_handler():
                step_used = self._step_used + step
            else:
                step_used = self._step_used
                self._step_used = self._step_limit
                raise OutOfStepException(
                    self._step_limit, step_used, step, step_type)

        self._step_used = step_used
        return step_used

    def refund_step(self, count: int) -> None:
        if self.schema == 0:
            steps: int = self._refund_cost * count
        else:
            steps: int = self._refund_base + self.get_step_cost(StepType.DELETE) * count
        self.add_step(steps)

    def add_step(self, amount: int) -> None:
        # Assuming amount is always less than the current limit
        self._step_used += amount

    def get_base_step(self, step_type: StepType) -> int:
        return self._step_base.get(step_type, 0)

    def get_step_cost(self, step_type: StepType) -> int:
        return self._step_costs.get(step_type, 0)
