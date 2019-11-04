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

import hashlib
from threading import Lock
from typing import Optional

from .icon_score_base import IconScoreBase
from .icon_score_loader import IconScoreLoader
from ..base.address import Address
from ..base.exception import InvalidParamsException
from ..database.db import IconScoreDatabase
from ..database.factory import ContextDatabaseFactory

TAG = 'ScoreMapper'


class IconScoreInfo(object):
    """ Contains information of an ICON SCORE class
    """

    def __init__(self, score_class: callable, code_path: str) -> None:
        self._score_class = score_class
        self._code_path = code_path

    @property
    def score_class(self) -> callable:
        return self._score_class

    @property
    def code_path(self) -> str:
        return self._code_path


class IconScoreMapperObject(dict):
    def __getitem__(self, key: bytes) -> 'IconScoreInfo':
        """operator[] overriding

        :param key:
        :return: IconScoreInfo instance
        """
        self._check_key_type(key)
        return super().__getitem__(key)

    def __setitem__(self, key: bytes, value: 'IconScoreInfo') -> None:
        """
        :param key:
        :param value: IconScoreInfo
        """
        self._check_key_type(key)
        self._check_value_type(value)
        super().__setitem__(key, value)

    @staticmethod
    def _check_key_type(key: bytes) -> None:
        if not isinstance(key, bytes):
            raise InvalidParamsException(f'{key} is not bytes type.')

    @staticmethod
    def _check_value_type(info: 'IconScoreInfo') -> None:
        if not isinstance(info, IconScoreInfo):
            raise InvalidParamsException(f'{info} is not IconScoreInfo type.')


class IconScoreMapper(object):
    """Icon score information mapping table

    This instance should be used as a singleton

    key: hash of code_path
    value: IconScoreInfo
    """

    def __init__(self, is_lock: bool = False) -> None:
        """Constructor
        """
        self._objects = IconScoreMapperObject()
        self._lock = Lock()
        self._is_lock = is_lock

    def __contains__(self, key: bytes):
        if self._is_lock:
            with self._lock:
                return key in self._objects
        else:
            return key in self._objects

    def __setitem__(self, key, value):
        if self._is_lock:
            with self._lock:
                self._objects[key] = value
        else:
            self._objects[key] = value

    def get(self, key: bytes):
        if self._is_lock:
            with self._lock:
                return self._objects.get(key)
        else:
            return self._objects.get(key)

    def get_icon_score(self, address: 'Address', code_path: str) -> Optional['IconScoreBase']:
        """
        :param address:
        :param code_path:
        :return: IconScoreBase object
        """
        key = hashlib.sha3_256(code_path.encode()).digest()
        score_info: IconScoreInfo = self.get(key)

        if score_info is None:
            score_class = IconScoreLoader.load_module(code_path)
            if score_class is None:
                return None
            self.put_score_info(key, score_class, code_path)
        else:
            score_class = score_info.score_class

        score_db = self._create_icon_score_database(address)
        return score_class(score_db)

    def put_score_info(self,
                       key: bytes,
                       score_class: callable,
                       code_path: str) -> None:
        self[key] = IconScoreInfo(score_class, code_path)

    @staticmethod
    def _create_icon_score_database(address: 'Address') -> 'IconScoreDatabase':
        """ Create IconScoreDatabase instance
            with icon_score_address and ContextDatabase

        :param address: icon_score_address
        """
        context_db = ContextDatabaseFactory.create_by_address(address)
        score_db = IconScoreDatabase(address, context_db)
        return score_db
