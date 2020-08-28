# Copyright 2020 ICON Foundation
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

import os
import unittest

from pyexec.base.address import Address, AddressPrefix
from pyexec.database.db import IconScoreDatabase
from pyexec.database.factory import ContextDatabaseFactory
from pyexec.icon_constant import IconScoreContextType
from pyexec.iconscore.icon_container_db import (
    ARRAY_DB_ID, DICT_DB_ID, VAR_DB_ID,
    ArrayDB, DictDB, VarDB, get_encoded_key
)
from pyexec.iconscore.icon_score_context import ContextContainer, IconScoreContext


class DummyProxy(object):
    def __init__(self):
        self._db = {}

    def get_value(self, k: bytes) -> bytes:
        ret = self._db.get(k)
        return ret

    def set_value(self, k: bytes, v: bytes, cb):
        self._db[k] = v


class TestContainerDB(unittest.TestCase):
    _dict_cases = [
        ['<name>', ['<sub>'], b'<value>'],
        ['<name2>', ['<sub1>', '<sub2>'], b'<value2>'],
        ['<name3>', ['<sub1>', '<sub2>', '<sub3>'], b'<value3>']
    ]

    @staticmethod
    def get_score_db():
        address = Address.from_data(AddressPrefix.EOA, os.urandom(20))
        context_db = ContextDatabaseFactory.create_by_address(address)
        return IconScoreDatabase(address, context_db)

    def setUp(self):
        ContextDatabaseFactory.open(DummyProxy(), ContextDatabaseFactory.Mode.SINGLE_DB)
        context = IconScoreContext(IconScoreContextType.INVOKE)
        ContextContainer._push_context(context)

    def test_dict_db(self):
        db = self.get_score_db()
        for c in self._dict_cases:
            _subs = c[1]
            _depth = len(_subs)
            _dict = DictDB(c[0], db, value_type=bytes, depth=_depth)
            for i in range(_depth - 1):
                _dict = _dict[_subs[i]]
            _dict[_subs[-1]] = c[2]

            _prefix = DICT_DB_ID + get_encoded_key(c[0])
            for i in range(_depth):
                _prefix += get_encoded_key(_subs[i])
            value = db.get(_prefix)
            self.assertEqual(c[2], value)

    def test_array_db(self):
        db = self.get_score_db()
        cases = [
            ['<name>', [b'v0']],
            ['<name2>', [b'v1', b'v2']],
            ['<name3>', [b'<value1>', b'<value2>', b'<value3>']]
        ]
        for c in cases:
            _subs = c[1]
            _array = ArrayDB(c[0], db, value_type=bytes)
            for i in range(len(_subs)):
                _array.put(_subs[i])
            self.assertEqual(len(_subs), len(_array))

            _prefix = ARRAY_DB_ID + get_encoded_key(c[0])
            values = []
            for i in range(len(_subs)):
                values.append(db.get(_prefix + get_encoded_key(i)))
            self.assertEqual(_subs, values)

    def test_var_db(self):
        db = self.get_score_db()
        cases = [
            ['<name>', b'<value>'],
            ['<name2>', b'<value2>'],
            ['<name3>', b'<value3>']
        ]
        for c in cases:
            _name = c[0]
            _var = VarDB(_name, db, value_type=bytes)
            _var.set(c[1])

            _prefix = VAR_DB_ID + get_encoded_key(_name)
            value = db.get(_prefix)
            self.assertEqual(c[1], value)

    def test_sub_db(self):
        db = self.get_score_db()
        for c in self._dict_cases:
            _subs = c[1]
            sub_db = db
            for i in range(len(_subs)):
                sub_db = sub_db.get_sub_db(_subs[i])

            # VarDB
            _var = VarDB(c[0], sub_db, value_type=bytes)
            _var.set(c[2])

            _prefix = VAR_DB_ID + get_encoded_key(c[0])
            for i in range(len(_subs)):
                _prefix += get_encoded_key(_subs[i])
            value = db.get(_prefix)
            self.assertEqual(c[2], value)

            # ArrayDB
            _array = ArrayDB(c[0], sub_db, value_type=bytes)
            _array.put(c[2])

            _prefix = ARRAY_DB_ID + get_encoded_key(c[0])
            for i in range(len(_subs)):
                _prefix += get_encoded_key(_subs[i])
            value = db.get(_prefix + get_encoded_key(b'\x00'))
            self.assertEqual(c[2], value)

            # DictDB
            _dict = DictDB(c[0], sub_db, value_type=bytes)
            _dict[b'<key>'] = c[2]

            _prefix = DICT_DB_ID + get_encoded_key(c[0])
            for i in range(len(_subs)):
                _prefix += get_encoded_key(_subs[i])
            _prefix += get_encoded_key(b'<key>')
            value = db.get(_prefix)
            self.assertEqual(c[2], value)


if __name__ == '__main__':
    unittest.main()
