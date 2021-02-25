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

from enum import Flag, unique
from typing import TypeVar

from ..base.address import Address

T = TypeVar('T')
BaseType = TypeVar('BaseType', bool, int, str, bytes, Address)

CONST_CLASS_API = '__api'
CONST_CLASS_ELEMENT_METADATA = '__element_metadata'

CONST_SCORE_FLAG = '__score_flag'
CONST_INDEXED_ARGS_COUNT = '__indexed_args_count'

FORMAT_IS_NOT_FUNCTION_OBJECT = "isn't function object: {}, cls: {}"
FORMAT_IS_NOT_DERIVED_OF_OBJECT = "isn't derived of {}"
FORMAT_DECORATOR_DUPLICATED = "can't duplicated {} decorator func: {}, cls: {}"

STR_FALLBACK = 'fallback'
STR_ON_INSTALL = 'on_install'
STR_ON_UPDATE = 'on_update'


@unique
class ScoreFlag(Flag):
    NONE = 0

    # Used for external functions
    READONLY = 0x01
    EXTERNAL = 0x02
    PAYABLE = 0x04
    FALLBACK = 0x08
    ISOLATED = 0x10
    FUNC = 0xFF

    # Used for other declarations
    EVENTLOG = 0x100
    INTERFACE = 0x200

    ALL = 0xFFFF
