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

from ..base.address import Address, GETAPI_DUMMY_ADDRESS
from .db import ContextDatabase, ProxyDatabase, DummyDatabase


class ContextDatabaseFactory(object):

    class Mode(IntEnum):
        SINGLE_DB = 0
        MULTIPLE_DB = 1

    _proxy = None
    _mode: 'Mode' = Mode.SINGLE_DB
    _shared_context_db: 'ContextDatabase' = None

    @classmethod
    def open(cls, proxy, mode: 'Mode'):
        cls._proxy = proxy
        cls._mode = mode

    @classmethod
    def get_shared_db(cls) -> ContextDatabase:
        if cls._shared_context_db is None:
            key_value_db = ProxyDatabase(cls._proxy)
            cls._shared_context_db = ContextDatabase(key_value_db)
        return cls._shared_context_db

    @classmethod
    def create_by_address(cls, address: 'Address') -> ContextDatabase:
        if address == GETAPI_DUMMY_ADDRESS:
            return ContextDatabase(DummyDatabase())
        elif cls._mode == cls.Mode.SINGLE_DB:
            return cls.get_shared_db()
        else:
            return cls.create_by_name(address.body.hex())

    @classmethod
    def create_by_name(cls, name: str) -> ContextDatabase:
        if cls._mode == cls.Mode.SINGLE_DB:
            return cls.get_shared_db()
        else:
            raise BaseException('Not supported')

    @classmethod
    def close(cls):
        if cls._shared_context_db:
            cls._shared_context_db = None
