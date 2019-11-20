# -*- coding: utf-8 -*-

# Copyright 2018 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from iconservice import *


def params_type_converter(param_type: str, value: any):
    if param_type == "int":
        param = _convert_value_int(value)
    elif param_type == "str":
        param = _convert_value_string(value)
    elif param_type == "bool":
        param = _convert_value_bool(value)
    elif param_type == "Address":
        param = _convert_value_address(value)
    elif param_type == "bytes":
        param = _convert_value_bytes(value)
    else:
        raise IconScoreException(
            f"{param_type} is not supported type (only int, str, bool, Address, bytes are supported)")
    return param


def _convert_value_int(value) -> int:
    if isinstance(value, int):
        result = value
    elif isinstance(value, str):
        if value.startswith('0x') or value.startswith('-0x'):
            result = int(value, 16)
        else:
            result = int(value)
    else:
        raise IconScoreException("type and value's actual type are not match.")
    return result


def _convert_value_string(value) -> str:
    if isinstance(value, str):
        return value
    else:
        raise IconScoreException("type and value's actual type are not match.")


def _convert_value_bool(value) -> bool:
    if isinstance(value, bool):
        result = value
    elif isinstance(value, str):
        result = bool(_convert_value_int(value))
    else:
        raise IconScoreException("type and value's actual type are not match.")
    return result


def _convert_value_address(value) -> 'Address':
    if isinstance(value, str):
        return Address.from_string(value)
    else:
        raise IconScoreException("type and value's actual type are not match.")


def _convert_value_bytes(value) -> bytes:
    # as JSON format doesn't accept bytes type, don't check if is instance of bytes.
    if isinstance(value, str):
        if value.startswith('0x'):
            result = bytes.fromhex(value[2:])
        else:
            result = bytes.fromhex(value)
    else:
        raise IconScoreException("type and value's actual type are not match.")
    return result
