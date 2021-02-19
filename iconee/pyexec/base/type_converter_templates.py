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

from enum import IntEnum


class ParamType(IntEnum):
    BLOCK = 0

    INVOKE_TRANSACTION = 100
    ACCOUNT_DATA = 101
    CALL_DATA = 102
    DEPLOY_DATA = 103
    TRANSACTION_PARAMS_DATA = 104

    INVOKE = 200

    QUERY = 300
    ICX_CALL = 301
    ICX_GET_BALANCE = 302
    ICX_GET_TOTAL_SUPPLY = 303
    ICX_GET_SCORE_API = 304
    ISE_GET_STATUS = 305

    WRITE_PRECOMMIT = 400
    REMOVE_PRECOMMIT = 500

    VALIDATE_TRANSACTION = 600


class ValueType(IntEnum):
    IGNORE = 0
    LATER = 1
    INT = 2
    STRING = 3
    BOOL = 4
    ADDRESS = 5
    BYTES = 6

    # For backward compatibility (TestNet)
    ADDRESS_OR_MALFORMED_ADDRESS = 7
    HEXADECIMAL = 8


type_convert_templates = {}
CONVERT_USING_SWITCH_KEY = 'CONVERT_USING_SWITCH_KEY'
SWITCH_KEY = "SWITCH_KEY"
KEY_CONVERTER = 'KEY_CONVERTER'


class ConstantKeys:
    BLOCK_HEIGHT = "blockHeight"
    BLOCK_HASH = "blockHash"
    TIMESTAMP = "timestamp"
    PREV_BLOCK_HASH = "prevBlockHash"

    NAME = "name"
    ADDRESS = "address"
    BALANCE = "balance"

    METHOD = "method"
    PARAMS = "params"

    CONTENT_TYPE = "contentType"
    CONTENT = "content"

    TX_HASH = "txHash"
    VERSION = "version"
    FROM = "from"
    TO = "to"
    VALUE = "value"
    STEP_LIMIT = "stepLimit"
    FEE = "fee"
    NONCE = "nonce"
    SIGNATURE = "signature"

    DATA_TYPE = "dataType"
    DATA = "data"
    CALL = "call"
    DEPLOY = "deploy"

    OLD_TX_HASH = "tx_hash"

    GENESIS_DATA = "genesisData"
    ACCOUNTS = "accounts"
    MESSAGE = "message"

    BLOCK = "block"
    TRANSACTIONS = "transactions"

    FILTER = "filter"

    ICX_CALL = "icx_call"
    ICX_GET_BALANCE = "icx_getBalance"
    ICX_GET_TOTAL_SUPPLY = "icx_getTotalSupply"
    ICX_GET_SCORE_API = "icx_getScoreApi"
    ISE_GET_STATUS = "ise_getStatus"


type_convert_templates[ParamType.BLOCK] = {
    ConstantKeys.BLOCK_HEIGHT: ValueType.INT,
    ConstantKeys.BLOCK_HASH: ValueType.BYTES,
    ConstantKeys.TIMESTAMP: ValueType.INT,
    ConstantKeys.PREV_BLOCK_HASH: ValueType.BYTES,
}

type_convert_templates[ParamType.ACCOUNT_DATA] = {
    ConstantKeys.NAME: ValueType.STRING,
    ConstantKeys.ADDRESS: ValueType.ADDRESS,
    ConstantKeys.BALANCE: ValueType.INT
}

type_convert_templates[ParamType.CALL_DATA] = {
    ConstantKeys.METHOD: ValueType.STRING,
    ConstantKeys.PARAMS: ValueType.LATER
}

type_convert_templates[ParamType.DEPLOY_DATA] = {
    ConstantKeys.CONTENT_TYPE: ValueType.STRING,
    ConstantKeys.CONTENT: ValueType.IGNORE,
    ConstantKeys.PARAMS: ValueType.LATER
}

type_convert_templates[ParamType.TRANSACTION_PARAMS_DATA] = {
    ConstantKeys.VERSION: ValueType.INT,
    ConstantKeys.TX_HASH: ValueType.BYTES,
    ConstantKeys.FROM: ValueType.ADDRESS,
    ConstantKeys.TO: ValueType.ADDRESS_OR_MALFORMED_ADDRESS,
    ConstantKeys.VALUE: ValueType.HEXADECIMAL,
    ConstantKeys.STEP_LIMIT: ValueType.INT,
    ConstantKeys.FEE: ValueType.HEXADECIMAL,
    ConstantKeys.TIMESTAMP: ValueType.INT,
    ConstantKeys.NONCE: ValueType.INT,
    ConstantKeys.SIGNATURE: ValueType.IGNORE,
    ConstantKeys.DATA_TYPE: ValueType.STRING,
    ConstantKeys.DATA: {
        CONVERT_USING_SWITCH_KEY: {
            SWITCH_KEY: ConstantKeys.DATA_TYPE,
            ConstantKeys.CALL: type_convert_templates[ParamType.CALL_DATA],
            ConstantKeys.DEPLOY: type_convert_templates[ParamType.DEPLOY_DATA]
        }
    },
    KEY_CONVERTER: {
        ConstantKeys.OLD_TX_HASH: ConstantKeys.TX_HASH
    }
}

type_convert_templates[ParamType.INVOKE_TRANSACTION] = {
    ConstantKeys.METHOD: ValueType.STRING,
    ConstantKeys.PARAMS: type_convert_templates[ParamType.TRANSACTION_PARAMS_DATA],
    ConstantKeys.GENESIS_DATA: {
        ConstantKeys.ACCOUNTS: [
            type_convert_templates[ParamType.ACCOUNT_DATA]
        ],
        ConstantKeys.MESSAGE: ValueType.STRING
    }
}

type_convert_templates[ParamType.INVOKE] = {
    ConstantKeys.BLOCK: type_convert_templates[ParamType.BLOCK],
    ConstantKeys.TRANSACTIONS: [
        type_convert_templates[ParamType.INVOKE_TRANSACTION]
    ]
}

type_convert_templates[ParamType.ICX_CALL] = {
    ConstantKeys.VERSION: ValueType.INT,
    ConstantKeys.FROM: ValueType.ADDRESS,
    ConstantKeys.TO: ValueType.ADDRESS,
    ConstantKeys.DATA_TYPE: ValueType.STRING,
    ConstantKeys.DATA: ValueType.LATER
}
type_convert_templates[ParamType.ICX_GET_BALANCE] = {
    ConstantKeys.VERSION: ValueType.INT,
    ConstantKeys.ADDRESS: ValueType.ADDRESS_OR_MALFORMED_ADDRESS
}
type_convert_templates[ParamType.ICX_GET_TOTAL_SUPPLY] = {
    ConstantKeys.VERSION: ValueType.INT
}
type_convert_templates[ParamType.ICX_GET_SCORE_API] = type_convert_templates[ParamType.ICX_GET_BALANCE]

type_convert_templates[ParamType.ISE_GET_STATUS] = {
    ConstantKeys.FILTER: [ValueType.STRING]
}

type_convert_templates[ParamType.QUERY] = {
    ConstantKeys.METHOD: ValueType.STRING,
    ConstantKeys.PARAMS: {
        CONVERT_USING_SWITCH_KEY: {
            SWITCH_KEY: ConstantKeys.METHOD,
            ConstantKeys.ICX_CALL: type_convert_templates[ParamType.ICX_CALL],
            ConstantKeys.ICX_GET_BALANCE: type_convert_templates[ParamType.ICX_GET_BALANCE],
            ConstantKeys.ICX_GET_TOTAL_SUPPLY: type_convert_templates[ParamType.ICX_GET_TOTAL_SUPPLY],
            ConstantKeys.ICX_GET_SCORE_API: type_convert_templates[ParamType.ICX_GET_SCORE_API],
            ConstantKeys.ISE_GET_STATUS: type_convert_templates[ParamType.ISE_GET_STATUS]
        }
    }
}

type_convert_templates[ParamType.WRITE_PRECOMMIT] = {
    ConstantKeys.BLOCK_HEIGHT: ValueType.INT,
    ConstantKeys.BLOCK_HASH: ValueType.BYTES
}
type_convert_templates[ParamType.REMOVE_PRECOMMIT] = type_convert_templates[ParamType.WRITE_PRECOMMIT]

type_convert_templates[ParamType.VALIDATE_TRANSACTION] = {
    ConstantKeys.METHOD: ValueType.STRING,
    ConstantKeys.PARAMS: type_convert_templates[ParamType.TRANSACTION_PARAMS_DATA]
}
