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

from enum import IntFlag, unique, IntEnum

CHARSET_ENCODING = 'utf-8'

# 32bytes == 256bit
DEFAULT_BYTE_SIZE = 32
DATA_BYTE_ORDER = 'big'  # big endian

# Reserved EventLog
ICX_TRANSFER_EVENT_LOG = 'ICXTransfer(Address,Address,int)'
GOV_REJECTED_EVENT_LOG = 'Rejected(str,str)'


# Revisions
@unique
class Revision(IntEnum):
    TWO = 2
    THREE = 3
    FOUR = 4
    IISS = 5
    DECENTRALIZATION = 6
    FIX_TOTAL_ELECTED_PREP_DELEGATED = 7
    VALUE_MASK = 0xFF

    COMPACT_JSON = 0x100
    LEGACY_INPUT_JSON = 0x40000

    JSON_COSTING = COMPACT_JSON | LEGACY_INPUT_JSON

    @classmethod
    def to_value(cls, revision: int) -> int:
        return revision & cls.VALUE_MASK

    def is_set(self, revision: int):
        return (self.value & revision) != 0


class IconServiceFlag(IntFlag):
    FEE = 1
    AUDIT = 2
    DEPLOYER_WHITE_LIST = 4
    SCORE_PACKAGE_VALIDATOR = 8


@unique
class IconScoreContextType(IntEnum):
    # Write data to db directly
    DIRECT = 0
    # Record data to cache and after confirming the block, write them to db
    INVOKE = 1
    # Record data to cache for estimation of steps, discard cache after estimation.
    ESTIMATION = 2
    # Not possible to write data to db
    QUERY = 3


@unique
class IconScoreFuncType(IntEnum):
    # ReadOnly function
    READONLY = 0
    # Writable function
    WRITABLE = 1


class Status(IntEnum):
    SUCCESS = 0
    FAILURE = 1
