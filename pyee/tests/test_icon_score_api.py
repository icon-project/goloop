# Copyright 2019 ICON Foundation
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

import base64
import hashlib
import unittest

from pyexec.base.address import Address
from pyexec.icon_constant import REVISION_COMPACT_JSON
from pyexec.iconscore.icon_score_base2 import ScoreApiStepRatio
from pyexec.iconscore.icon_score_base2 import _create_address_with_key, _recover_key
from pyexec.iconscore.icon_score_base2 import create_address_with_key, recover_key
from pyexec.iconscore.icon_score_base2 import sha3_256, sha_256, json_dumps, json_loads
from pyexec.iconscore.icon_score_context import ContextContainer
from pyexec.iconscore.icon_score_context import IconScoreContext, IconScoreContextType
from pyexec.iconscore.icon_score_step import StepType, IconScoreStepCounter


def create_msg_hash(tx: dict, excluded_keys: tuple) -> bytes:
    keys = [key for key in tx if key not in excluded_keys]
    keys.sort()

    msg = 'icx_sendTransaction'
    for key in keys:
        value: str = tx[key]
        msg += f'.{key}.{value}'

    return hashlib.sha3_256(msg.encode('utf-8')).digest()


class TestIconScoreApi(unittest.TestCase):
    def setUp(self):
        # The transaction in block 1000 of TestNet
        self.tx_v2 = {
            'from': 'hxdbc9f726ad776d9a43d5bad387eff01325178fa3',
            'to': 'hx0fb148785e4a5d77d16429c7ed2edae715a4453a',
            'value': '0x324e964b3eca80000',
            'fee': '0x2386f26fc10000',
            'timestamp': '1519709385120909',
            'tx_hash': '1257b9ea76e716b145463f0350f534f973399898a18a50d391e7d2815e72c950',
            'signature': 'WiRTA/tUNGVByc8fsZ7+U9BSDX4BcBuv2OpAuOLLbzUiCcovLPDuFE+PBaT8ovmz5wg+Bjr7rmKiu7Rl8v0DUQE=',
        }

        # The transaction in block 100000 of MainNet
        self.tx_v3 = {
            'version': '0x3',
            'nid': '0x1',
            'from': 'hx522bff55a62e0c75a1b51855b0802cfec6a92e84',
            'to': 'hx11de4e28be4845de3ea392fd8d758655bf766ca7',
            'value': '0x71afd498d0000',
            'stepLimit': '0xf4240',
            'timestamp': '0x57a4e5556cc03',
            'signature': 'fcEMXqEGlqEivXXr7YtD/F1RXgxSXF+R4gVrGKxT1zxi3HukX4NzkSl9/Es1G+nyZx+kviTAtQFUrA+/T0NrfAA=',
            'txHash': '6c71ac77b2d130a1f81d234e814974e85cabb0a3ec462c66ff3f820502d0ded2'
        }

        self.step_costs = {
            StepType.GET: 0,
            StepType.SET: 320,
            StepType.REPLACE: 80,
            StepType.DELETE: -240,
            StepType.EVENT_LOG: 100,
            StepType.API_CALL: 10_000
        }
        self.step_limit = 1_000_000_000

        self.context = self._create_context()
        ContextContainer._push_context(self.context)

    def _create_context(self):
        context = IconScoreContext(IconScoreContextType.INVOKE)
        step_counter = IconScoreStepCounter(self.step_costs, self.step_limit,
                                            self._dummy_refund_handler)
        context.step_counter = step_counter
        context.revision = REVISION_COMPACT_JSON
        return context

    def _dummy_refund_handler(self):
        pass

    def _calc_step_cost(self, ratio: ScoreApiStepRatio) -> int:
        step_cost: int = self.step_costs[StepType.API_CALL] * ratio // ScoreApiStepRatio.SHA3_256
        self.assertTrue(isinstance(step_cost, int))
        self.assertTrue(step_cost > 0)

        return step_cost

    def tearDown(self):
        ContextContainer._pop_context()

    def test_recover_key_v2_and_create_address_with_key(self):
        signature: bytes = base64.b64decode(self.tx_v2['signature'])
        self.assertIsInstance(signature, bytes)
        self.assertTrue(len(signature) > 0)

        msg_hash: bytes = create_msg_hash(self.tx_v2, ('tx_hash', 'signature'))
        self.assertEqual(msg_hash, bytes.fromhex(self.tx_v2['tx_hash']))

        uncompressed_public_key: bytes = _recover_key(msg_hash, signature, compressed=False)
        self.assertIsInstance(uncompressed_public_key, bytes)
        self.assertEqual(65, len(uncompressed_public_key))
        self.assertEqual(0x04, uncompressed_public_key[0])

        address: Address = _create_address_with_key(uncompressed_public_key)
        self.assertEqual(self.tx_v2['from'], str(address))

        compressed_public_key: bytes = _recover_key(msg_hash, signature, compressed=True)
        self.assertIsInstance(compressed_public_key, bytes)
        self.assertEqual(33, len(compressed_public_key))
        self.assertIn(compressed_public_key[0], (0x02, 0x03))

        address: Address = _create_address_with_key(compressed_public_key)
        self.assertEqual(self.tx_v2['from'], str(address))

    def test_recover_key_v3_and_create_address_with_key(self):
        signature: bytes = base64.b64decode(self.tx_v3['signature'])
        self.assertIsInstance(signature, bytes)
        self.assertTrue(len(signature) > 0)

        msg_hash: bytes = create_msg_hash(self.tx_v3, ('txHash', 'signature'))
        self.assertEqual(msg_hash, bytes.fromhex(self.tx_v3['txHash']))

        uncompressed_public_key: bytes = _recover_key(msg_hash, signature, compressed=False)
        self.assertIsInstance(uncompressed_public_key, bytes)
        self.assertEqual(65, len(uncompressed_public_key))
        self.assertEqual(0x04, uncompressed_public_key[0])

        address: Address = _create_address_with_key(uncompressed_public_key)
        self.assertEqual(self.tx_v3['from'], str(address))

        compressed_public_key: bytes = _recover_key(msg_hash, signature, compressed=True)
        self.assertIsInstance(compressed_public_key, bytes)
        self.assertEqual(33, len(compressed_public_key))
        self.assertIn(compressed_public_key[0], (0x02, 0x03))

        address: Address = _create_address_with_key(compressed_public_key)
        self.assertEqual(self.tx_v3['from'], str(address))

    def test_recover_key_step_with_tx_v3(self):
        step_cost: int = self._calc_step_cost(ScoreApiStepRatio.RECOVER_KEY)

        signature: bytes = base64.b64decode(self.tx_v3['signature'])
        self.assertIsInstance(signature, bytes)
        self.assertTrue(len(signature) > 0)

        msg_hash: bytes = create_msg_hash(self.tx_v3, ('txHash', 'signature'))
        self.assertEqual(msg_hash, bytes.fromhex(self.tx_v3['txHash']))

        uncompressed_public_key: bytes = recover_key(msg_hash, signature, compressed=False)
        self.assertIsInstance(uncompressed_public_key, bytes)
        self.assertEqual(65, len(uncompressed_public_key))
        self.assertEqual(0x04, uncompressed_public_key[0])

        step_used: int = self.context.step_counter.step_used
        self.assertEqual(step_cost, step_used)

        self.context.step_counter._step_used = 0

        compressed_public_key: bytes = recover_key(msg_hash, signature, compressed=True)
        self.assertIsInstance(compressed_public_key, bytes)
        self.assertEqual(33, len(compressed_public_key))
        self.assertIn(compressed_public_key[0], (0x02, 0x03))

        step_used: int = self.context.step_counter.step_used
        self.assertEqual(step_cost, step_used)

    def test_create_address_with_key_step_with_tx_v3(self):
        uncompressed_step_cost: int = self._calc_step_cost(ScoreApiStepRatio.CREATE_ADDRESS_WITH_UNCOMPRESSED_KEY)
        compressed_step_cost: int = self._calc_step_cost(ScoreApiStepRatio.CREATE_ADDRESS_WITH_COMPRESSED_KEY)
        self.assertTrue(uncompressed_step_cost != compressed_step_cost)

        signature: bytes = base64.b64decode(self.tx_v3['signature'])
        self.assertIsInstance(signature, bytes)
        self.assertTrue(len(signature) > 0)

        msg_hash: bytes = create_msg_hash(self.tx_v3, ('txHash', 'signature'))
        self.assertEqual(msg_hash, bytes.fromhex(self.tx_v3['txHash']))

        uncompressed_public_key: bytes = recover_key(msg_hash, signature, compressed=False)
        self.assertIsInstance(uncompressed_public_key, bytes)
        self.assertEqual(65, len(uncompressed_public_key))
        self.assertEqual(0x04, uncompressed_public_key[0])

        self.context.step_counter._step_used = 0

        address: Address = create_address_with_key(uncompressed_public_key)
        self.assertEqual(self.tx_v3['from'], str(address))

        step_used: int = self.context.step_counter.step_used
        self.assertEqual(uncompressed_step_cost, step_used)

        compressed_public_key: bytes = recover_key(msg_hash, signature, compressed=True)
        self.assertIsInstance(compressed_public_key, bytes)
        self.assertEqual(33, len(compressed_public_key))
        self.assertIn(compressed_public_key[0], (0x02, 0x03))

        self.context.step_counter._step_used = 0

        address: Address = create_address_with_key(compressed_public_key)
        self.assertEqual(self.tx_v3['from'], str(address))

        step_used: int = self.context.step_counter.step_used
        self.assertEqual(compressed_step_cost, step_used)

    def test_sha3_256_step(self):
        step_cost: int = self._calc_step_cost(ScoreApiStepRatio.SHA3_256)

        for i in range(0, 512):
            chunks = i // 32
            if i % 32 > 0:
                chunks += 1

            data: bytes = b'\x00' * i
            hash_value: bytes = sha3_256(data)
            assert hash_value == hashlib.sha3_256(data).digest()

            expected_step: int = step_cost + step_cost * chunks // 10
            step_used: int = self.context.step_counter.step_used
            self.assertEqual(expected_step, step_used)

            self.context.step_counter._step_used = 0

    def test_sha_256_step(self):
        step_cost: int = self._calc_step_cost(ScoreApiStepRatio.SHA_256)

        for i in range(0, 512):
            chunks = i // 32
            if i % 32 > 0:
                chunks += 1

            data: bytes = b'\x00' * i
            hash_value: bytes = sha_256(data)
            assert hash_value == hashlib.sha256(data).digest()

            expected_step: int = step_cost + step_cost * chunks // 10
            step_used: int = self.context.step_counter.step_used
            self.assertEqual(expected_step, step_used)

            self.context.step_counter._step_used = 0

    def test_json_dumps_step(self):
        step_cost: int = self._calc_step_cost(ScoreApiStepRatio.JSON_DUMPS)

        for i in range(1, 100):
            obj = {}

            for j in range(i):
                obj[f'key{j}'] = f'value{j}'

            text: str = json_dumps(obj)

            expected_step: int = step_cost + step_cost * len(text.encode('utf-8')) // 100
            step_used: int = self.context.step_counter.step_used
            self.assertEqual(expected_step, step_used)

            obj2: dict = json_loads(text)
            self.assertEqual(obj, obj2)

            self.context.step_counter._step_used = 0

    def test_json_loads_step(self):
        step_cost: int = self._calc_step_cost(ScoreApiStepRatio.JSON_LOADS)

        for i in range(1, 100):
            obj = {}

            for j in range(i):
                obj[f'key{j}'] = f'value{j}'

            text: str = json_dumps(obj)

            self.context.step_counter._step_used = 0

            obj2: dict = json_loads(text)
            self.assertEqual(obj, obj2)

            expected_step: int = step_cost + step_cost * len(text.encode('utf-8')) // 100
            step_used: int = self.context.step_counter.step_used
            self.assertEqual(expected_step, step_used)


if __name__ == '__main__':
    unittest.main()
