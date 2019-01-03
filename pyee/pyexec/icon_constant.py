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

ICON_SERVICE_LOG_TAG = 'IconService'
ICON_EXCEPTION_LOG_TAG = f'{ICON_SERVICE_LOG_TAG}_Exception'
ICON_DEPLOY_LOG_TAG = f'{ICON_SERVICE_LOG_TAG}_Deploy'
ICON_LOADER_LOG_TAG = f'{ICON_SERVICE_LOG_TAG}_Loader'
ICX_LOG_TAG = f'{ICON_SERVICE_LOG_TAG}_Icx'
ICON_DB_LOG_TAG = f'{ICON_SERVICE_LOG_TAG}_DB'
ICON_INNER_LOG_TAG = f'IconInnerService'

JSONRPC_VERSION = '2.0'
CHARSET_ENCODING = 'utf-8'

# 32bytes == 256bit
DEFAULT_BYTE_SIZE = 32
DATA_BYTE_ORDER = 'big'  # big endian
# Fixed fee is 0.01 icx.
FIXED_FEE = 10 ** 16
# Max data field size
MAX_DATA_SIZE = 512 * 1024

# Max external call count(1 is default SCORE call, 1024 is external call in the SCORE)
MAX_EXTERNAL_CALL_COUNT = 1 + 1024

# Max call stack size
MAX_CALL_STACK_SIZE = 64

ICON_DEX_DB_NAME = 'icon_dex'

ICX_TRANSFER_EVENT_LOG = 'ICXTransfer(Address,Address,int)'

ICON_SCORE_QUEUE_NAME_FORMAT = "IconScore.{channel_name}.{amqp_key}"
ICON_SERVICE_PROCTITLE_FORMAT = "icon_service." \
                                "{scoreRootPath}." \
                                "{stateDbRootPath}." \
                                "{channel}.{amqpKey}." \
                                "{amqpTarget}"

BUILTIN_SCORE_ADDRESS_MAPPER = {'governance': "cx0000000000000000000000000000000000000001"}

REVISION_2 = 2
REVISION_3 = 3


class ConfigKey:
    BUILTIN_SCORE_OWNER = 'builtinScoreOwner'
    SERVICE = 'service'
    SERVICE_FEE = 'fee'
    SERVICE_AUDIT = 'audit'
    SERVICE_DEPLOYER_WHITE_LIST = 'deployerWhiteList'
    SERVICE_SCORE_PACKAGE_VALIDATOR = 'scorePackageValidator'
    SCORE_ROOT_PATH = 'scoreRootPath'
    STATE_DB_ROOT_PATH = 'stateDbRootPath'
    CHANNEL = 'channel'
    AMQP_KEY = 'amqpKey'
    AMQP_TARGET = 'amqpTarget'
    CONFIG = 'config'
    TBEARS_MODE = 'tbearsMode'


class EnableThreadFlag(IntFlag):
    INVOKE = 1
    QUERY = 2
    VALIDATE = 4


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


ENABLE_THREAD_FLAG = EnableThreadFlag.INVOKE | EnableThreadFlag.QUERY | EnableThreadFlag.VALIDATE


class Status(IntEnum):
    SUCCESS = 0
    FAILURE = 1
