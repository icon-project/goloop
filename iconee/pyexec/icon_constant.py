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

from enum import IntFlag, unique, IntEnum, Enum

CHARSET_ENCODING = 'utf-8'

# 32bytes == 256bit
DEFAULT_BYTE_SIZE = 32
DATA_BYTE_ORDER = 'big'  # big endian

# Reserved EventLog
ICX_TRANSFER_EVENT_LOG = 'ICXTransfer(Address,Address,int)'
GOV_REJECTED_EVENT_LOG = 'Rejected(str,str)'


# Revisions
class Revision(IntEnum):
    TWO = 2
    THREE = 3
    FOUR = 4
    IISS = 5
    DECENTRALIZATION = 6
    FIX_TOTAL_ELECTED_PREP_DELEGATED = 7

    # Revision 8
    REALTIME_P2P_ENDPOINT_UPDATE = 8
    OPTIMIZE_DIRTY_PREP_UPDATE = 8

    # Revision 9
    FIX_EMAIL_VALIDATION = 9
    DIVIDE_NODE_ADDRESS = 9
    FIX_BURN_EVENT_SIGNATURE = 9
    ADD_LOGS_BLOOM_ON_BASE_TX = 9
    SCORE_FUNC_PARAMS_CHECK = 9
    SYSTEM_SCORE_ENABLED = 9
    CHANGE_MAX_DELEGATIONS_TO_100 = 9
    PREVENT_DUPLICATED_ENDPOINT = 9
    SET_IREP_VIA_NETWORK_PROPOSAL = 9
    MULTIPLE_UNSTAKE = 9
    FIX_COIN_PART_BYTES_ENCODING = 9
    STRICT_SCORE_DECORATOR_CHECK = 9

    FIX_UNSTAKE_BUG = 10
    LOCK_ADDRESS = 10

    FIX_BALANCE_BUG = 11

    BURN_V2_ENABLED = 12
    IMPROVED_PRE_VALIDATOR = 12
    VERIFY_ASSET_INTEGRITY = 12
    USE_RLP = 12

    ICON2 = 13

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


class IconNetworkValueType(Enum):
    SERVICE_CONFIG = b'service_config'

    STEP_PRICE = b'step_price'
    STEP_COSTS = b'step_costs'
    MAX_STEP_LIMITS = b'max_step_limits'

    REVISION_CODE = b'revision_code'
    REVISION_NAME = b'revision_name'

    SCORE_BLACK_LIST = b'score_black_list'
    IMPORT_WHITE_LIST = b'import_white_list'

    IREP = b'irep'

    @classmethod
    def gs_migration_type_list(cls) -> list:
        return [
            cls.SERVICE_CONFIG,
            cls.STEP_PRICE,
            cls.STEP_COSTS,
            cls.MAX_STEP_LIMITS,
            cls.REVISION_CODE,
            cls.REVISION_NAME,
            cls.SCORE_BLACK_LIST,
            cls.IMPORT_WHITE_LIST,
        ]

    @classmethod
    def gs_migration_count(cls) -> int:
        return len(cls.gs_migration_type_list())


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
