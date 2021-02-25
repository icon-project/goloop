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

from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .address import Address


class Transaction(object):
    """Contains transaction info
    """

    def __init__(self,
                 tx_hash: Optional[bytes] = None,
                 index: int = 0,
                 origin: Optional['Address'] = None,
                 timestamp: int = None,
                 nonce: int = None) -> None:
        """Transaction class for icon score context
        """
        self._hash = tx_hash
        self._index = index
        self._origin = origin
        self._timestamp = timestamp
        self._nonce = nonce

    @property
    def origin(self) -> 'Address':
        """
        The account who created the transaction.
        """
        return self._origin

    @property
    def index(self) -> int:
        """
        Transaction index in a block
        """
        return self._index

    @property
    def hash(self) -> bytes:
        """
        Transaction hash
        """
        return self._hash

    @property
    def timestamp(self) -> int:
        """
        Timestamp of a transaction request in microseconds
        This is NOT a block timestamp
        """
        return self._timestamp

    @property
    def nonce(self) -> int:
        """
        (optional)
        nonce of a transaction request.
        random value
        """
        return self._nonce

    def __str__(self) -> str:
        hash_hex = 'None' if self._hash is None else f'0x{self._hash.hex()}'
        return f'hash({hash_hex}) ' \
            f'index({self.index}) ' \
            f'origin({self.origin}) ' \
            f'timestamp({hex(self.timestamp)}) ' \
            f'nonce({self.nonce})'
