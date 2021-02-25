# Copyright 2021 ICON Foundation
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

from abc import abstractmethod
from typing import Optional

from .internal_call import ChainScore
from ..base.address import Address, BUILTIN_SCORE_ADDRESS_MAPPER
from ..base.exception import AccessDeniedException
from ..database.db import IconScoreDatabase
from ..icon_constant import IconServiceFlag
from ..iconscore.icon_score_base import IconScoreBase


class IconScoreDeployTXParams(object):

    def __init__(self, tx_hash: bytes, score_address: 'Address'):
        self._tx_hash = tx_hash
        self._score_address = score_address

    @property
    def tx_hash(self) -> bytes:
        return self._tx_hash

    @property
    def score_address(self) -> 'Address':
        return self._score_address


class IconScoreDeployInfo(object):

    def __init__(self, score_address: 'Address', current_tx_hash: bytes, next_tx_hash: bytes):
        self._score_address = score_address
        self._current_tx_hash = current_tx_hash
        self._next_tx_hash = next_tx_hash

    @property
    def score_address(self) -> 'Address':
        return self._score_address

    @property
    def current_tx_hash(self) -> bytes:
        return self._current_tx_hash

    @property
    def next_tx_hash(self) -> bytes:
        return self._next_tx_hash


class IconSystemScoreBase(IconScoreBase):

    @abstractmethod
    def on_install(self, **kwargs) -> None:
        super().on_install(**kwargs)

    @abstractmethod
    def on_update(self, **kwargs) -> None:
        super().on_update(**kwargs)

    @abstractmethod
    def __init__(self, db: 'IconScoreDatabase') -> None:
        super().__init__(db)
        if not self.is_builtin_score(self.address):
            raise AccessDeniedException(f"Not a builtin SCORE")

    @staticmethod
    def is_builtin_score(score_address: 'Address') -> bool:
        return score_address in BUILTIN_SCORE_ADDRESS_MAPPER.values()

    def get_icon_service_flag(self) -> int:
        return IconServiceFlag.FEE \
               | IconServiceFlag.AUDIT \
               | IconServiceFlag.SCORE_PACKAGE_VALIDATOR

    def get_deploy_tx_params(self, tx_hash: bytes) -> Optional['IconScoreDeployTXParams']:
        address = ChainScore.txHashToAddress(self._context, self.address, tx_hash)
        return IconScoreDeployTXParams(tx_hash, address)

    def get_deploy_info(self, score_address: 'Address') -> Optional['IconScoreDeployInfo']:
        current, _next = ChainScore.addressToTxHashes(self._context, self.address, score_address)
        return IconScoreDeployInfo(score_address, current, _next)

    def is_score_active(self, score_address: 'Address') -> bool:
        status = ChainScore.getScoreStatus(self._context, self.address, score_address)
        if status is None:
            return False
        current = status.get('current', None)
        return current is not None and current['status'] == 'active'
