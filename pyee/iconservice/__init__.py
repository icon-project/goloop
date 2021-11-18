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

""" Default service packages for implementing ICON SCORE """

from abc import ABCMeta, abstractmethod, ABC
from functools import wraps
from inspect import isfunction
from typing import List, Union, Optional, Dict

from typing_extensions import TypedDict

from pyexec.logger import Logger

from pyexec.base.address import Address, AddressPrefix, SYSTEM_SCORE_ADDRESS, ZERO_SCORE_ADDRESS
from pyexec.base.exception import IconScoreException
from pyexec.icon_constant import IconServiceFlag

from pyexec.iconscore.icon_container_db import VarDB, DictDB, ArrayDB
from pyexec.iconscore.icon_score_base import (IconScoreBase, IconScoreDatabase,
                                              interface, eventlog, external, payable, isolated)
from pyexec.iconscore.icon_score_base2 import (revert, sha3_256, sha_256, json_loads, json_dumps,
                                               recover_key, create_address_with_key,
                                               InterfaceScore, create_interface_score)
