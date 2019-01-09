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

from struct import Struct
from typing import Optional
from ..icon_constant import DATA_BYTE_ORDER, DEFAULT_BYTE_SIZE


class Block(object):
    """Block Information included in IconScoreContext
    """
    _VERSION = 0
    # leveldb account value structure (bigendian, 1 + 32 + 32 + 32 + 32 bytes)
    # version(1)
    # | height(DEFAULT_BYTE_SIZE)
    # | hash(DEFAULT_BYTE_SIZE)
    # | timestamp(DEFAULT_BYTE_SIZE)
    # | prev_hash(DEFAULT_BYTE_SIZE)

    _struct = Struct(f'>B{DEFAULT_BYTE_SIZE}s{DEFAULT_BYTE_SIZE}s{DEFAULT_BYTE_SIZE}s{DEFAULT_BYTE_SIZE}s')

    def __init__(self, block_height: int, timestamp: int,
                 block_hash: bytes = None, prev_hash: bytes = None) -> None:
        """Constructor

        :param block_height: block height
        :param timestamp: block timestamp in microsecond
        :param block_hash: block hash
        :param prev_hash: prev block hash
        """
        self._height = block_height
        self._timestamp = timestamp
        self._hash = block_hash
        self._prev_hash = prev_hash

    @property
    def height(self) -> int:
        return self._height

    @property
    def hash(self) -> bytes:
        return self._hash

    @property
    def timestamp(self) -> int:
        return self._timestamp

    @property
    def prev_hash(self) -> bytes:
        return self._prev_hash

    @staticmethod
    def from_dict(params: dict):
        block_height = params.get('blockHeight')
        block_hash = params.get('blockHash')
        timestamp = params.get('timestamp', 0)
        prev_hash = params.get('prevBlockHash', b'\x00' * 32)
        return Block(block_height, timestamp, block_hash, prev_hash)

    @staticmethod
    def from_block(block: 'Block'):
        block_height = block.height
        block_hash = block.hash
        timestamp = block.timestamp
        prev_hash = block.prev_hash
        return Block(block_height, timestamp, block_hash, prev_hash)

    @staticmethod
    def from_bytes(buf: bytes) -> 'Block':
        """Create Account object from bytes data

        :param buf: (bytes) bytes data including Account information
        :return: (Account) account object
        """
        byteorder = DATA_BYTE_ORDER

        version, block_height_bytes, block_hash_bytes, \
            timestamp_bytes, block_prev_hash_bytes = \
            Block._struct.unpack(buf)

        block_height = int.from_bytes(block_height_bytes, byteorder)
        block_hash = block_hash_bytes
        timestamp = int.from_bytes(timestamp_bytes, byteorder)
        byte_prev_hash = block_prev_hash_bytes

        if int(bytes.hex(byte_prev_hash), 16) == 0:
            byte_prev_hash = None
        prev_block_hash = byte_prev_hash

        block = Block(block_height, timestamp, block_hash, prev_block_hash)
        return block

    def to_bytes(self) -> bytes:
        """Convert block object to bytes

        :return: data including information of block object
        """

        byteorder = DATA_BYTE_ORDER
        # for extendability
        block_height_bytes = self._height.to_bytes(DEFAULT_BYTE_SIZE, byteorder)
        block_hash_bytes = self._hash
        timestamp_bytes = self._timestamp.to_bytes(DEFAULT_BYTE_SIZE, byteorder)

        tmp_prev_hash = self._prev_hash
        if tmp_prev_hash is None:
            tmp_prev_hash = bytes(DEFAULT_BYTE_SIZE)
        prev_block_hash_bytes = tmp_prev_hash

        return Block._struct.pack(
            self._VERSION,
            block_height_bytes,
            block_hash_bytes,
            timestamp_bytes,
            prev_block_hash_bytes)

    def __bytes__(self) -> bytes:
        """operator bytes() overriding

        :return: binary data including information of account object
        """
        return self.to_bytes()

    def __str__(self) -> str:
        hash_hex = 'None' if self._hash is None else f'0x{self._hash.hex()}'
        prev_hash_hex = \
            'None' if self._prev_hash is None else f'0x{self._prev_hash.hex()}'

        return f'height({self._height}) ' \
            f'hash({hash_hex}) ' \
            f'timestamp({hex(self._timestamp)}) ' \
            f'prev_hash({prev_hash_hex})'
