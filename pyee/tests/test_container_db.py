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
from pyexec.database.db import IconScoreDatabase, get_encoded_key
from pyexec.database.factory import ContextDatabaseFactory
from pyexec.icon_constant import IconScoreContextType
from pyexec.iconscore.icon_container_db import (
    ARRAY_DB_ID, DICT_DB_ID, VAR_DB_ID,
    ArrayDB, DictDB, VarDB, ContainerUtil
)
from pyexec.iconscore.icon_score_context import ContextContainer, IconScoreContext
from pyexec.utils import sha3_256


class DummyProxy(object):
    def __init__(self):
        self._db = {}

    def get_value(self, k: bytes) -> bytes:
        ret = self._db.get(k)
        return ret

    def set_value(self, k: bytes, v: bytes, cb):
        self._db[k] = v


class TestContainerDB(unittest.TestCase):

    @staticmethod
    def get_score_db():
        address = Address.from_data(AddressPrefix.EOA, os.urandom(20))
        context_db = ContextDatabaseFactory.create_by_address(address)
        return context_db, IconScoreDatabase(address, context_db)

    def setUp(self):
        ContextDatabaseFactory.open(DummyProxy(), ContextDatabaseFactory.Mode.SINGLE_DB)
        context = IconScoreContext(IconScoreContextType.INVOKE)
        ContextContainer._push_context(context)

    def test_dict_db(self):
        cdb, db = self.get_score_db()
        ctx = db._context
        cases = [
            ['name1', ['s1'], 'k1', b'<v1>', [DICT_DB_ID, 'name1', 's1', 'k1']],
            ['name2', [], 'k1', b'<v2>', [DICT_DB_ID, 'name2', 'k1']],
        ]
        for c in cases:
            keys = c[1]
            ddb = db
            for i in range(len(keys)):
                ddb = ddb.get_sub_db(ContainerUtil.encode_key(keys[i]))

            dbase = DictDB(c[0], ddb, value_type=bytes)
            dbase[c[2]] = c[3]

            key = b''.join(map(lambda x: get_encoded_key(ContainerUtil.encode_key(x)), c[4]))
            self.assertEqual(c[3], cdb.get(ctx, sha3_256(key)))

    def test_array_db(self):
        cdb, db = self.get_score_db()
        ctx = db._context
        cases = [
            ['name1', ['s1'], [b'<v1>'], [ARRAY_DB_ID, 'name1', 's1']],
            ['name2', [], [b'<v2>', b'<v3>'], [ARRAY_DB_ID, 'name2']],
        ]
        for c in cases:
            keys = c[1]
            ddb = db
            for i in range(len(keys)):
                ddb = ddb.get_sub_db(ContainerUtil.encode_key(keys[i]))

            values = c[2]
            dbase = ArrayDB(c[0], ddb, value_type=bytes)
            for v in values:
                dbase.put(v)

            key = b''.join(map(lambda x: get_encoded_key(ContainerUtil.encode_key(x)), c[3]))
            self.assertEqual(ContainerUtil.encode_value(len(values)), cdb.get(ctx, sha3_256(key)))
            for i in range(len(values)):
                key2 = key + get_encoded_key(ContainerUtil.encode_key(i))
                self.assertEqual(values[i], cdb.get(ctx, sha3_256(key2)))

    def test_var_db(self):
        cdb, db = self.get_score_db()
        ctx = db._context
        cases = [
            ['name1', ['s1'], b'<v1>', [VAR_DB_ID, 'name1', 's1']],
            ['name2', [], b'<v2>', [VAR_DB_ID, 'name2']],
        ]
        for c in cases:
            keys = c[1]
            ddb = db
            for i in range(len(keys)):
                ddb = ddb.get_sub_db(ContainerUtil.encode_key(keys[i]))

            dbase = VarDB(c[0], ddb, value_type=bytes)
            dbase.set(c[2])

            key = b''.join(map(lambda x: get_encoded_key(ContainerUtil.encode_key(x)), c[3]))
            self.assertEqual(ContainerUtil.encode_value(c[2]), cdb.get(ctx, sha3_256(key)))

    def test_dict_db_vs_sub_db(self):
        cdb, db = self.get_score_db()
        cases = [
            ['<name1>', ['<sub1>', '<sub2>', '<sub3>'], b'<value1>'],
            ['<name2>', [1, '<sub2>', '<sub3>'], b'<value2>'],
            ['<name3>', [1, '<sub2>', '<sub3>', 128], b'<value3>'],
        ]
        for c in cases:
            keys = c[1]
            depth = len(keys)

            # set data only with dictionary
            sd = DictDB(c[0], db, value_type=bytes, depth=depth)
            for i in range(depth-1):
                sd = sd[keys[i]]
            sd[keys[depth-1]] = c[2]

            # check stored value in sub_db
            for sub_depth in range(depth):
                ddb = db
                for j in range(sub_depth):
                    ddb = ddb.get_sub_db(ContainerUtil.encode_key(keys[j]))

                d = DictDB(c[0], ddb, value_type=bytes, depth=depth-sub_depth)
                for k in range(sub_depth, depth-1):
                    d = d[keys[k]]
                self.assertEqual(d[keys[depth-1]], c[2])


if __name__ == '__main__':
    unittest.main()
