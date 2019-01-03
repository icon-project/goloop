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

from typing import TYPE_CHECKING, List, Optional

from .icon_score_eventlog import EventLog
from ..utils.bloom import BloomFilter
from ..base.address import Address
from ..base.block import Block
from ..icon_constant import DATA_BYTE_ORDER

if TYPE_CHECKING:
    from ..base.transaction import Transaction


class TransactionResult(object):
    """ A data class of a transaction result.
    """

    SUCCESS = 1
    FAILURE = 0

    class Failure(object):
        def __init__(self, code: int, message: str):
            self.code = int(code)
            self.message = str(message)

    def __init__(
            self,
            tx: 'Transaction',
            block: 'Block',
            to: Optional['Address'] = None,
            score_address: Optional['Address'] = None,
            step_used: int = 0,
            step_price: int = 0,
            event_logs: Optional[List['EventLog']] = None,
            logs_bloom: Optional[BloomFilter] = None,
            status: int = FAILURE) -> None:
        """Constructor

        :param tx: transaction
        :param block: a block that the transaction belongs to
        :param to: a recipient address
        :param score_address:hex string that represent the contract address
            if the transaction`s target is contract
        :param step_used: the amount of steps used in the transaction
        :param event_logs: the amount of steps used in the transaction
        :param logs_bloom: bloom filter data of event logs
        :param status: a status of result. 1 (success) or 0 (failure)
        """
        self.tx_hash = tx.hash
        self.block_height = block.height
        self.block_hash = block.hash
        self.tx_index = tx.index
        self.to = to
        self.score_address = score_address
        self.step_used = step_used
        self.step_price = step_price
        self.event_logs = event_logs
        self.logs_bloom = logs_bloom
        self.status = status

        # failure object which has code(int) and message(str) attributes
        # It is only available on self.status == FAILURE
        self.failure = None

        # Traces are managed in TransactionResult but not passed to chain engine
        self.traces = None

    def __str__(self) -> str:
        return '\n'.join([f'{k}: {v}' for k, v in self.__dict__.items()])

    def to_dict(self, casing: Optional = None) -> dict:
        """
        Returns properties as `dict`
        :return: a dict
        """
        new_dict = {}
        for key, value in self.__dict__.items():
            # Excludes properties which have `None` value
            if value is None:
                continue

            new_key = casing(key) if casing else key
            if key == 'event_logs':
                new_dict[new_key] = [v.to_dict(casing) for v in value if
                                     isinstance(v, EventLog)]
            elif isinstance(value, BloomFilter):
                new_dict[new_key] = int(value).to_bytes(256, byteorder=DATA_BYTE_ORDER)
            elif key == 'failure' and value:
                if self.status == self.FAILURE:
                    new_dict[new_key] = {
                        'code': value.code,
                        'message': value.message
                    }
            elif key == 'traces':
                # traces are excluded from dict property
                continue
            else:
                new_dict[new_key] = value

        return new_dict
